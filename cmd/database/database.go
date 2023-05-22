package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"database/sql"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/util"
)

type ErrPortInUse struct {
	Port string
}

func (e ErrPortInUse) Error() string {
	return fmt.Sprintf("port %s is in use", e.Port)
}

// BringUpPostgresLocally spins up a PostgreSQL server locally and returns
// a connection to it.
//
// It is the client's responsibility to call db.Close() on the returned
// connection when done with it.
//
// It deploys it with Docker.

// You can specify if it should use the existing container, so as to retain its mounted
// volume, and thus stored data. This is good for the Run command. Conversely, the /runtime/gql/api test
// suite, needs to have a virgin container for each test case, to avoid conflicts with prior state.
//
// But it can also cope with a virgin run when even the required Docker image
// is not in the local docker registery.
//
// It sets the password for that user to "postgres".
// It sets the default database name to a hash of the project directory.
func Start(useExistingContainer bool, projectDirectory string) (*db.ConnectionInfo, error) {
	// Bring up the container, if it isn't already up.
	connectionInfo, err := bringUpContainer(useExistingContainer)
	if err != nil {
		return nil, err
	}

	// Generate unique database name and append it to connectionInfo.
	dbName, err := generateDatabaseName(projectDirectory)
	if err != nil {
		return nil, err
	}
	connectionInfo = connectionInfo.WithDatabase(dbName)

	// Create database if it doesn't exist.
	err = createDatabaseIfNotExists(connectionInfo)
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

// Stop stops the postgres container - having checked first
// that such a container exists, and it is running.
func Stop() error {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	container, err := findContainer(dockerClient)
	if err != nil {
		return err
	}
	if container == nil {
		return nil
	}
	running, err := isContainerRunning(dockerClient, container)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	stopTimeout := 5 * time.Second
	// Note that ContainerStop() gracefully stops the container by choice, but then forcibly after the timeout.
	err = dockerClient.ContainerStop(context.Background(), container.ID, &stopTimeout)
	if err != nil {
		return err
	}
	return nil
}

func bringUpContainer(useExistingContainer bool) (*db.ConnectionInfo, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	postgresImage, err := findImage(dockerClient)
	if err != nil {
		return nil, err
	}

	if postgresImage == nil {
		fmt.Println("Pulling postgres Docker image...")
		reader, err := dockerClient.ImagePull(context.Background(), postgresImageName+":"+postgresTag, types.ImagePullOptions{})
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		// ImagePull() is async - and the suggested protocol for waiting for it to complete is
		// to read from the returned reader, until you reach EOF.
		awaitReadCompletion(reader)
	}

	postgresContainer, err := findContainer(dockerClient)
	if err != nil {
		return nil, err
	}

	// If we want to create a fresh container, but there is already one registered
	// with that name, we have to remove it first to be able to re create it.
	if postgresContainer != nil && !useExistingContainer {
		err = dockerClient.ContainerRemove(
			context.Background(),
			postgresContainer.ID,
			types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			})

		if err != nil {
			return nil, err
		}

		postgresContainer = nil
	}

	var port string
	usingExistingContainer := postgresContainer != nil

	if postgresContainer != nil {
		container, err := dockerClient.ContainerInspect(context.Background(), postgresContainer.ID)
		if err != nil {
			return nil, err
		}

		// find the port this container is bound to
		for p, bindings := range container.HostConfig.PortBindings {
			if p.Proto() == "tcp" && p.Port() == "5432" && len(bindings) > 0 {
				port = bindings[0].HostPort
			}
		}

	} else {
		containerConfig := &container.Config{
			Image: postgresImageName + ":" + postgresTag,
			Env: []string{
				"POSTGRES_PASSWORD=postgres",
				"POSTGRES_USER=postgres",
			},
		}

		// get a free port
		port, err = util.GetFreePort("5432")
		if err != nil {
			return nil, err
		}

		_, err = dockerClient.ContainerCreate(
			context.Background(),
			containerConfig,
			makeHostConfig(port),
			nil,
			nil,
			keelPostgresContainerName)
		if err != nil {
			return nil, err
		}

		postgresContainer, _ = findContainer(dockerClient)
	}

	// See if container is running
	isRunning, err := isContainerRunning(dockerClient, postgresContainer)
	if err != nil {
		return nil, err
	}

	if !isRunning {
		err := dockerClient.ContainerStart(
			context.Background(),
			postgresContainer.ID,
			types.ContainerStartOptions{})
		if err != nil {
			if usingExistingContainer && strings.Contains(err.Error(), "port is already allocated") {
				return nil, ErrPortInUse{port}
			}
			return nil, err
		}
	}

	return &db.ConnectionInfo{
		Username: "postgres",
		Password: "postgres",
		Host:     "0.0.0.0",
		Port:     port,
	}, nil
}

func findImage(dockerClient *client.Client) (*types.ImageSummary, error) {
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	searchFor := strings.Join([]string{postgresImageName, postgresTag}, ":")
	for _, image := range images {
		tags := image.RepoTags
		if lo.Contains(tags, searchFor) {
			return &image, nil
		}
	}
	return nil, nil
}

func findContainer(dockerClient *client.Client) (container *types.Container, err error) {
	listOptions := types.ContainerListOptions{
		All: true,
	}
	containers, err := dockerClient.ContainerList(context.Background(), listOptions)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if lo.Contains(c.Names, "/"+keelPostgresContainerName) {
			return &c, nil
		}
	}

	return nil, nil
}

func isContainerRunning(dockerClient *client.Client, container *types.Container) (bool, error) {
	containerJSON, err := dockerClient.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return false, err
	}
	return containerJSON.State.Running, nil
}

// awaitReadCompletion reads from the given reader until it reaches EOF.
// It's used in the context of waiting for a docker image to be fetched, and
// is the method used in the docker SDK to wait for the fetch to be complete.
// We exploit it also to emit a growing row of dot characters to indicate
// progress.
func awaitReadCompletion(r io.Reader) {
	// Consuming the output in N-byte chunks gives us circa
	// a friendly number of read cycles - good for outputting a progress dot "." for each of them.
	buf := make([]byte, 2000)
	for {
		_, err := r.Read(buf)
		fmt.Printf(".")
		if err != nil {
			if err != io.EOF {
				panic(fmt.Sprintf("error from read operation: %v", err))

			}
			fmt.Printf("\n")
			return
		}
	}
}

func makeHostConfig(port string) *container.HostConfig {
	portBinding := nat.PortBinding{
		HostIP:   "",
		HostPort: port,
	}
	portMap := nat.PortMap{
		nat.Port("5432/tcp"): []nat.PortBinding{portBinding},
	}
	hostConfig := &container.HostConfig{
		PortBindings: portMap,
	}
	return hostConfig
}

// Generates a unique database name using a hash of the project's working directory
// For example: keel_48f77af86bffe7cdbb44308a70d11f8b
func generateDatabaseName(projectDirectory string) (string, error) {
	if strings.HasPrefix(projectDirectory, "~/") {
		home, _ := os.UserHomeDir()
		projectDirectory = filepath.Join(home, projectDirectory[2:])
	}

	// Ensure path is absolute and cleaned for determinism.
	projectDirectory, err := filepath.Abs(projectDirectory)
	if err != nil {
		return "", err
	}

	projectDirectory = strings.ToLower(projectDirectory)

	return fmt.Sprintf("keel_%x", md5.Sum([]byte(projectDirectory))), nil
}

// createDatabaseIfNotExists creates a database if it does not exist.
// It uses the database name found in the db.ConnectionInfo argument.
func createDatabaseIfNotExists(info *db.ConnectionInfo) error {
	server, err := sql.Open("pgx/v5", info.ServerString())
	if err != nil {
		return err
	}

	// ping() the database server until it is available.
	var pingError error
	for i := 0; i < 10; i++ {
		if pingError = server.Ping(); pingError == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	result := server.QueryRow(fmt.Sprintf("SELECT COUNT(*) as count FROM pg_database WHERE datname = '%s'", info.Database))
	if err != nil {
		return err
	}
	var count int
	err = result.Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = server.Exec(fmt.Sprintf("CREATE DATABASE %s;", info.Database))
		if err != nil {
			return err
		}
	}

	return err
}

const postgresImageName string = "postgres"
const postgresTag string = "11.13"
const keelPostgresContainerName string = "keel-run-postgres"
