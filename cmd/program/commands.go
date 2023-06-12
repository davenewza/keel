package program

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/radovskyb/watcher"
	"github.com/teamkeel/keel/cmd/cliconfig"
	"github.com/teamkeel/keel/cmd/database"
	"github.com/teamkeel/keel/codegen"
	"github.com/teamkeel/keel/config"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/migrations"
	"github.com/teamkeel/keel/node"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema"
	"github.com/teamkeel/keel/schema/reader"
)

func NextMsgCommand(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

type LoadSchemaMsg struct {
	Schema      *proto.Schema
	Config      *config.ProjectConfig
	SchemaFiles []reader.SchemaFile
	Secrets     map[string]string
	Err         error
}

func LoadSchema(dir, environment string) tea.Cmd {
	return func() tea.Msg {
		b := schema.Builder{}
		s, err := b.MakeFromDirectory(dir)

		absolutePath, filepathErr := filepath.Abs(dir)
		if filepathErr != nil {
			err = filepathErr
		}

		cliConfig := cliconfig.New(&cliconfig.Options{
			WorkingDir: dir,
		})

		secrets, configErr := cliConfig.GetSecrets(absolutePath, environment)
		if configErr != nil {
			err = configErr
		}

		if b.Config == nil {
			b.Config = &config.ProjectConfig{}
		}

		invalid, invalidSecrets := b.Config.ValidateSecrets(secrets)
		if invalid {
			err = fmt.Errorf("missing secrets from local config in ~/.keel/config.yaml: %s", strings.Join(invalidSecrets, ", "))
		}

		msg := LoadSchemaMsg{
			Schema:      s,
			Config:      b.Config,
			SchemaFiles: b.SchemaFiles(),
			Secrets:     secrets,
			Err:         err,
		}

		return msg
	}
}

type GenerateMsg struct {
	Err            error
	Status         int
	GeneratedFiles codegen.GeneratedFiles
	Log            string
}

// Generate takes care of ensuring that all of the required steps to get a working Keel project with functions
// are executed including:
// - package.json + tsconfig.json generated
// - Installing deps via npm
// - Generating dynamic sdk + testing packages
// - Scaffolding any missing custom functions
func Generate(dir string, schema *proto.Schema, nodePackagesPath string, output chan tea.Msg) tea.Cmd {
	return func() tea.Msg {

		generatedFiles := codegen.GeneratedFiles{}

		if !node.HasFunctions(schema) && !node.HasTests(dir) {
			return GenerateMsg{
				Status: StatusNotGenerated,
			}
		}

		output <- GenerateMsg{
			Status: StatusBootstrapping,
			Log:    "Determining changes",
		}

		// make sure we have all correct deps
		files, err := node.Bootstrap(dir, node.WithPackagesPath(nodePackagesPath))

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		err = files.Write(dir)

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		output <- GenerateMsg{
			Status: StatusBootstrapping,
			Log:    "Satisfying dependencies",
		}

		if len(files) > 0 {
			for _, file := range files {
				output <- GenerateMsg{
					Status: StatusBootstrapping,
					Log:    fmt.Sprintf("Created %s", file.Path),
				}
			}
		} else {
			output <- GenerateMsg{
				Status: StatusBootstrapping,
				Log:    "Nothing to bootstrap",
			}
		}

		output <- GenerateMsg{
			Status: StatusNpmInstalling,
		}

		npmInstall := exec.Command("npm", "install", "--progress=false")
		npmInstall.Dir = dir
		stdoutPipe, _ := npmInstall.StdoutPipe()
		stderrPipe, _ := npmInstall.StderrPipe()

		// combine both stderr and stdout into a single reader which can be scanned
		multi := io.MultiReader(stdoutPipe, stderrPipe)

		err = npmInstall.Start()
		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		// Next, we construct a line scanner which will allow us to iterate
		// over the individual new lines written out into the stdout / stderr pipes
		scanner := bufio.NewScanner(multi)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			m := scanner.Text()

			output <- GenerateMsg{
				Log:    m,
				Status: StatusNpmInstalling,
			}
		}

		err = npmInstall.Wait()

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		files, err = node.Generate(context.TODO(), schema, node.WithDevelopmentServer(true))

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		err = files.Write(dir)

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		output <- GenerateMsg{
			Log:    "Generated @teamkeel/sdk",
			Status: StatusGeneratingNodePackages,
		}

		output <- GenerateMsg{
			Log:    "Generated .build directory",
			Status: StatusGeneratingNodePackages,
		}

		output <- GenerateMsg{
			Log:    "Generated @teamkeel/testing",
			Status: StatusGeneratingNodePackages,
		}

		// generate prisma client from the prisma schema we generated in the previous step
		cmd := exec.Command("node_modules/.bin/prisma", "generate", "--schema", ".build/schema.prisma")
		cmd.Dir = dir
		b, err := cmd.CombinedOutput()
		if err != nil {
			return GenerateMsg{
				Err: &TypeScriptError{
					Output: string(b),
					Err:    err,
				},
			}
		}

		output <- GenerateMsg{
			Log:    "Generated prisma schema",
			Status: StatusGeneratingNodePackages,
		}

		scaffoldedFiles, err := node.Scaffold(dir, schema)

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		err = scaffoldedFiles.Write(dir)

		if err != nil {
			return GenerateMsg{
				Err: err,
			}
		}

		generatedFiles = append(generatedFiles, scaffoldedFiles...)

		if len(generatedFiles) > 0 {
			for _, f := range scaffoldedFiles {

				output <- GenerateMsg{
					Log:    f.Path,
					Status: StatusScaffolding,
				}
			}

		} else {
			output <- GenerateMsg{
				Log:    "No functions to generate!",
				Status: StatusScaffolding,
			}
		}

		output <- GenerateMsg{
			GeneratedFiles: generatedFiles,
			Status:         StatusGenerated,
		}

		return nil
	}
}

type CheckDependenciesMsg struct {
	Err error
}

func CheckDependencies() tea.Cmd {
	return func() tea.Msg {
		err := node.CheckNodeVersion()

		return CheckDependenciesMsg{
			Err: err,
		}
	}
}

type ParsePrivateKeyMsg struct {
	PrivateKey *rsa.PrivateKey
	Err        error
}

func ParsePrivateKey(path string) tea.Cmd {
	return func() tea.Msg {
		if path != "" {
			privateKeyPem, err := os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				return ParsePrivateKeyMsg{
					Err: fmt.Errorf("cannot locate private key file at: %s", path),
				}
			} else if err != nil {
				return ParsePrivateKeyMsg{
					Err: fmt.Errorf("cannot read private key file: %s", err.Error()),
				}
			}

			privateKeyBlock, _ := pem.Decode(privateKeyPem)
			if privateKeyBlock == nil {
				return ParsePrivateKeyMsg{
					Err: errors.New("private key PEM either invalid or empty"),
				}
			}

			privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
			if err != nil {
				return ParsePrivateKeyMsg{
					Err: err,
				}
			}

			return ParsePrivateKeyMsg{
				PrivateKey: privateKey,
			}

		}

		return ParsePrivateKeyMsg{
			PrivateKey: nil,
		}
	}
}

type StartDatabaseMsg struct {
	ConnInfo *db.ConnectionInfo
	Err      error
}

func StartDatabase(reset bool, mode int, projectDirectory string) tea.Cmd {
	return func() tea.Msg {
		connInfo, err := database.Start(!reset, projectDirectory)
		if err != nil {
			return StartDatabaseMsg{
				Err: err,
			}
		}

		if mode != ModeTest {
			return StartDatabaseMsg{
				ConnInfo: connInfo,
			}
		}

		mainDB, err := sql.Open("pgx/v5", connInfo.String())
		if err != nil {
			return StartDatabaseMsg{
				Err: err,
			}
		}

		_, err = mainDB.Exec(`
			DROP DATABASE IF EXISTS keel_test
		`)
		if err != nil {
			return StartDatabaseMsg{
				Err: err,
			}
		}

		_, err = mainDB.Exec(`
			CREATE DATABASE keel_test
		`)
		if err != nil {
			return StartDatabaseMsg{
				Err: err,
			}
		}

		return StartDatabaseMsg{
			ConnInfo: connInfo.WithDatabase("keel_test"),
		}
	}
}

type SetupFunctionsMsg struct {
	Err error
}

func SetupFunctions(dir string, nodePackagesPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := node.Bootstrap(dir, node.WithPackagesPath(nodePackagesPath))
		if err != nil {
			return SetupFunctionsMsg{
				Err: err,
			}
		}

		err = files.Write(dir)

		if err != nil {
			return SetupFunctionsMsg{
				Err: err,
			}
		}

		npmInstall := exec.Command("npm", "install")
		npmInstall.Dir = dir
		err = npmInstall.Run()

		if err != nil {
			return SetupFunctionsMsg{
				Err: err,
			}
		}
		return SetupFunctionsMsg{}
	}
}

type UpdateFunctionsMsg struct {
	Err error
}

type TypeScriptError struct {
	Output string
	Err    error
}

func (t *TypeScriptError) Error() string {
	return fmt.Sprintf("TypeScript error: %s", t.Err.Error())
}

func UpdateFunctions(schema *proto.Schema, dir string) tea.Cmd {
	return func() tea.Msg {
		files, err := node.Generate(context.TODO(), schema, node.WithDevelopmentServer(true))
		if err != nil {
			return UpdateFunctionsMsg{Err: err}
		}

		err = files.Write(dir)
		if err != nil {
			return UpdateFunctionsMsg{Err: err}
		}

		cmd := exec.Command("node_modules/.bin/prisma", "generate", "--schema", ".build/schema.prisma")
		cmd.Dir = dir
		b, err := cmd.CombinedOutput()
		if err != nil {
			return UpdateFunctionsMsg{
				Err: &TypeScriptError{
					Output: string(b),
					Err:    err,
				},
			}
		}

		cmd = exec.Command("npx", "tsc", "--noEmit", "--pretty")
		cmd.Dir = dir

		b, err = cmd.CombinedOutput()
		if err != nil {
			return UpdateFunctionsMsg{
				Err: &TypeScriptError{
					Output: string(b),
					Err:    err,
				},
			}
		}

		return UpdateFunctionsMsg{}
	}
}

type RunMigrationsMsg struct {
	Err     error
	Changes []*migrations.DatabaseChange
}

type ApplyMigrationsError struct {
	Err error
}

func (a *ApplyMigrationsError) Error() string {
	return a.Err.Error()
}

func RunMigrations(schema *proto.Schema, database db.Database) tea.Cmd {
	return func() tea.Msg {

		m, err := migrations.New(context.Background(), schema, database)
		if err != nil {
			return RunMigrationsMsg{
				Err: &ApplyMigrationsError{
					Err: err,
				},
			}
		}

		msg := RunMigrationsMsg{
			Changes: m.Changes,
		}

		if !m.HasModelFieldChanges() {
			return msg
		}

		err = m.Apply(context.Background())
		if err != nil {
			msg.Err = &ApplyMigrationsError{
				Err: err,
			}
		}

		return msg
	}
}

type StartFunctionsMsg struct {
	Err    error
	Server *node.DevelopmentServer
}

type StartFunctionsError struct {
	Err    error
	Output string
}

func (s *StartFunctionsError) Error() string {
	return s.Err.Error()
}

type FunctionsOutputMsg struct {
	Output string
}

func StartFunctions(m *Model) tea.Cmd {
	return func() tea.Msg {
		envType := "development"
		if m.Mode == ModeTest {
			envType = "test"
		}

		envVars := m.Config.GetEnvVars(envType)
		envVars["KEEL_DB_CONN_TYPE"] = "pg"
		envVars["KEEL_DB_CONN"] = m.DatabaseConnInfo.String()

		if m.TracingEnabled {
			envVars["KEEL_TRACING_ENABLED"] = "true"
			envVars["OTEL_RESOURCE_ATTRIBUTES"] = "service.name=functions"
		}

		output := &FunctionsOutputWriter{
			// Initially buffer output inside the writer in case there's an error
			Buffer: true,
			ch:     m.functionsLogCh,
		}
		server, err := node.RunDevelopmentServer(m.ProjectDir, &node.ServerOpts{
			EnvVars: envVars,
			Output:  output,
		})
		if err != nil {
			return StartFunctionsMsg{
				Err: &StartFunctionsError{
					Output: strings.Join(output.Output, "\n"),
					Err:    err,
				},
			}
		}

		// Stop buffering output now we know the process started.
		// All future output will be written to the given channel
		output.Buffer = false

		return StartFunctionsMsg{
			Server: server,
		}
	}
}

type RuntimeRequestMsg struct {
	w    http.ResponseWriter
	r    *http.Request
	done chan bool
}

func StartRuntimeServer(port string, ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		runtimeServer := http.Server{
			Addr: fmt.Sprintf(":%s", port),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				done := make(chan bool, 1)
				ch <- RuntimeRequestMsg{
					w:    w,
					r:    r,
					done: done,
				}
				<-done
			}),
		}
		_ = runtimeServer.ListenAndServe()
		return nil
	}
}

type FunctionsOutputWriter struct {
	Output []string
	Buffer bool
	ch     chan tea.Msg
}

func (f *FunctionsOutputWriter) Write(p []byte) (n int, err error) {
	str := string(p)

	if len(strings.Split(str, "\n")) > 1 {
		str = fmt.Sprintf("\n%s", str)
	}

	if f.Buffer {
		f.Output = append(f.Output, str)
	} else {
		f.ch <- FunctionsOutputMsg{
			Output: str,
		}
	}

	return len(p), nil
}

type WatcherMsg struct {
	Err   error
	Path  string
	Event string
}

func StartWatcher(dir string, ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		w := watcher.New()
		w.SetMaxEvents(1)
		w.FilterOps(watcher.Write, watcher.Remove)

		ignored := []string{
			"node_modules/",
			".build/",
		}

		w.AddFilterHook(func(info os.FileInfo, fullPath string) error {
			for _, v := range ignored {
				if strings.Contains(fullPath, v) {
					return watcher.ErrSkip
				}
			}

			return nil
		})

		go func() {
			for {
				select {
				case event := <-w.Event:
					ch <- WatcherMsg{
						Path:  event.Path,
						Event: event.Op.String(),
					}
				case <-w.Closed:
					return
				}
			}
		}()

		err := w.AddRecursive(dir)
		if err != nil {
			return WatcherMsg{
				Err: err,
			}
		}

		_ = w.Start(time.Millisecond * 100)
		return nil
	}
}

type RunTestsMsg struct {
	Err    error
	Output string
}

func RunTests(dir string, port string, cfg *config.ProjectConfig, conn *db.ConnectionInfo, pattern string) tea.Cmd {
	return func() tea.Msg {
		args := []string{
			"vitest",
			"run",
			"--color",
			"--reporter", "verbose",
			"--config", "./.build/vitest.config.mjs",
		}

		if pattern != "" {
			args = append(args, "--testNamePattern", pattern)
		}

		cmd := exec.Command("npx", args...)
		cmd.Dir = dir
		cmd.Env = os.Environ()

		envVars := cfg.GetEnvVars("test")
		envVars["KEEL_TESTING_ACTIONS_API_URL"] = fmt.Sprintf("http://localhost:%s/testingactionsapi/json", port)
		envVars["KEEL_DB_CONN_TYPE"] = "pg"
		envVars["KEEL_DB_CONN"] = conn.String()
		envVars["NODE_OPTIONS"] = "--no-warnings"

		for key, value := range envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		b, err := cmd.CombinedOutput()
		return RunTestsMsg{
			Output: string(b),
			Err:    err,
		}
	}
}

// LoadSecrets lists secrets from the given file and returns a command
func LoadSecrets(path, environment string) (map[string]string, error) {
	projectPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	config := cliconfig.New(&cliconfig.Options{
		WorkingDir: projectPath,
	})

	secrets, err := config.GetSecrets(path, environment)
	if err != nil {
		return nil, err

	}
	return secrets, nil
}

func SetSecret(path, environment, key, value string) error {
	projectPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	config := cliconfig.New(&cliconfig.Options{
		WorkingDir: projectPath,
	})

	return config.SetSecret(path, environment, key, value)
}

func RemoveSecret(path, environment, key string) error {
	projectPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	config := cliconfig.New(&cliconfig.Options{
		WorkingDir: projectPath,
	})

	return config.RemoveSecret(path, environment, key)
}

func isDirEmpty(name string) bool {
	// we can ignore the error
	entries, _ := os.ReadDir(name)

	// if len is 0 then dir is empty or doesn't exist
	return len(entries) == 0
}
