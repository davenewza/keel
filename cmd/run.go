package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/teamkeel/keel/cmd/database"
	"github.com/teamkeel/keel/functions"
	"github.com/teamkeel/keel/migrations"
	"github.com/teamkeel/keel/node"
	"github.com/teamkeel/keel/runtime"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"github.com/teamkeel/keel/schema"
	"github.com/teamkeel/keel/schema/validation/errorhandling"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gorm.io/driver/postgres"
)

const dbConnString = "postgres://%s:%s@%s:%s/%s"

// The Run command does this:
//
//   - Starts Postgres in a docker container.
//   - Loads the Keel schema files, validates them, and watches for changes
//   - When the Keel schema files are valid migrations are generated and run
//     against the database and a new runtime handler is created
//   - Starts an HTTP server which when the Keel schema files are currently
//     valid delegates the requests to the runtime handler. When there are
//     validation errors in the schema files then an error is returned.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run your Keel App locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		b := &schema.Builder{}
		useExistingContainer := !runCmdFlagReset
		dbConn, dbConnInfo, err := database.Start(useExistingContainer)

		if err != nil {
			if portErr, ok := err.(database.ErrPortInUse); ok {
				color.Red("Unable to start database: %s\n", portErr.Error())
				color.Yellow("To create a fresh database container on a different port re-run this command with --reset\n\n")
				return nil
			}
			panic(err)
		}
		defer database.Stop()

		logLevel := logger.Warn
		if runCmdFlagVerbose {
			logLevel = logger.Info
		}

		// todo: unify db logging with custom functions
		logger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logLevel,    // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Disable color
			},
		)

		db, err := gorm.Open(postgres.New(postgres.Config{
			Conn: dbConn,
		}), &gorm.Config{
			Logger: logger,
		})
		if err != nil {
			panic(err)
		}

		opts := []node.BootstrapOption{}
		if os.Getenv("KEEL_LOCAL_PACKAGES_PATH") != "" {
			opts = append(opts, node.WithPackagesPath(os.Getenv("KEEL_LOCAL_PACKAGES_PATH")))
		}

		err = node.Bootstrap(inputDir, opts...)
		if err != nil {
			panic(err)
		}

		var mutex sync.Mutex
		var functionsServer *node.DevelopmentServer
		var functionsTransport functions.Transport

		// We run a Node.js server in the background to handle requests to the
		// Functions runtime, which in turn routes requests so that they can be
		// passed to individual custom functions
		restartFunctionServer := func() {
			if functionsServer != nil {
				_ = functionsServer.Kill()
			}

			dbConnString := fmt.Sprintf(dbConnString, dbConnInfo.Username, dbConnInfo.Password, dbConnInfo.Host, dbConnInfo.Port, dbConnInfo.Database)

			functionsServer, err = node.RunDevelopmentServer(inputDir, &node.ServerOpts{
				EnvVars: map[string]string{
					"DB_CONN_TYPE": "pg",
					"DB_CONN":      dbConnString,
				},
			})
			if err != nil {
				fmt.Print(err.Error())
				panic(err)
			}

			functionsTransport = functions.NewHttpTransport(functionsServer.URL)
		}

		currSchema, err := migrations.GetCurrentSchema(context.Background(), db)
		if err != nil {
			panic(err)
		}

		reloadSchema := func(changedFile string) {
			mutex.Lock()
			defer mutex.Unlock()

			clearTerminal()
			printRunHeader(inputDir, dbConnInfo)

			if changedFile != "" {
				fmt.Println("Detected change to:", changedFile)
			}

			fmt.Println("📂 Loading schema files")

			protoSchema, err := b.MakeFromDirectory(inputDir)

			if err != nil {
				errs, ok := err.(*errorhandling.ValidationErrors)

				if !ok {
					panic(err)
				}

				out, err := errs.ToAnnotatedSchema(b.SchemaFiles())
				if err != nil {
					panic(err)
				}

				color.New(color.FgRed).Printf("\nThere is an error in your schema:\n")

				fmt.Printf("\n%s\n", out)

				fmt.Print("\a")

				// currSchema = nil
				return
			}

			fmt.Println("✅ Schema is valid")

			m := migrations.New(protoSchema, currSchema)

			if m.HasModelFieldChanges() {
				fmt.Println("💿 Applying migrations")
				err = m.Apply(db)
				if err != nil {
					panic(err)
				}

				printMigrationChanges(m.Changes)
			} else {
				fmt.Println("💿 Applying changes")
				err = m.Apply(db)
				if err != nil {
					panic(err)
				}
			}

			files, err := node.Generate(context.Background(), inputDir, node.WithDevelopmentServer(true))
			if err != nil {
				panic(err)
			}

			err = files.Write()
			if err != nil {
				panic(err)
			}

			if node.HasFunctions(protoSchema) {
				// kill the old node server hosting the old code, and
				// spawn a new node server for the new version of the code
				restartFunctionServer()
			}

			currSchema = protoSchema
			fmt.Println("🎉 You're ready to roll")
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			mutex.Lock()
			defer mutex.Unlock()

			fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)

			if strings.HasSuffix(r.URL.Path, "/graphiql") {
				handler := playground.Handler("GraphiQL", strings.TrimSuffix(r.URL.Path, "/graphiql")+"/graphql")
				handler(w, r)
				return
			}

			ctx := r.Context()
			ctx = runtimectx.WithDatabase(ctx, db)
			if functionsTransport != nil {
				ctx = functions.WithFunctionsTransport(ctx, functionsTransport)
			}
			r = r.WithContext(ctx)

			runtime.NewHttpHandler(currSchema).ServeHTTP(w, r)
		})

		reloadSchema("")

		// this needs to be executed here because
		// reloadSchema populates the currSchema
		hasCustomFunctions := node.HasFunctions(currSchema)

		stopWatcher, err := onSchemaFileChanges(inputDir, hasCustomFunctions, reloadSchema)
		if err != nil {
			panic(err)
		}
		defer stopWatcher()

		go http.ListenAndServe(":"+runCmdFlagPort, http.DefaultServeMux)

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		// Kill the Functions node server when the command exits
		if functionsServer != nil {
			functionsServer.Kill()
		}
		fmt.Println("\n👋 Bye bye")
		return nil
	},
}

func clearTerminal() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout

	// We do not mind if the clear command fails
	// because clear isn't implemented in some terminals
	// and it fails in the VSCode debugger so we don't want
	// to panic as we were doing originally.
	cmd.Run()
}

func printRunHeader(dir string, dbConnInfo *database.ConnectionInfo) {
	fmt.Printf("Watching schema files in: %s\n", color.CyanString(dir))

	psql := color.CyanString("psql postgresql://%s:%s@%s:%s/%s",
		dbConnInfo.Username,
		dbConnInfo.Password,
		dbConnInfo.Host,
		dbConnInfo.Port,
		dbConnInfo.Database)

	endpoint := color.CyanString("http://localhost:%s\n", runCmdFlagPort)

	fmt.Printf("Connect to the database: %s\n", psql)
	fmt.Printf("Application running at: %s\n", endpoint)
	fmt.Printf("Press CTRL-C to exit\n\n")
}

func printMigrationChanges(changes []*migrations.DatabaseChange) {
	var t string

	for _, ch := range changes {
		fmt.Printf(" - ")
		switch ch.Type {
		case migrations.ChangeTypeAdded:
			t = color.YellowString(ch.Type)
		case migrations.ChangeTypeRemoved:
			t = color.RedString(ch.Type)
		case migrations.ChangeTypeModified:
			t = color.GreenString(ch.Type)
		}
		fmt.Printf(" %s %s", t, ch.Model)
		if ch.Field != "" {
			fmt.Printf(".%s", ch.Field)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
}

// reactToSchemaChanges should be called in its own goroutine. It has a blocking
// channel select loop that waits for and receives file system events, or errors.
func onSchemaFileChanges(dir string, hasCustomFunctions bool, cb func(changedFile string)) (func() error, error) {
	// The run command remains quiescent now, until the user changes their schema, so we establish
	// a watcher on the schema directorty.
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch {
				case strings.HasSuffix(event.Name, ".keel"):
					cb(event.Name)
				case strings.HasSuffix(event.Name, ".ts"):
					cb(event.Name)
				case !isRelevantEventType(event.Op):
					// Ignore
				default:

				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("error received on watcher error channel: %v\n", err)
				// If we get an internal error from the watcher - we simply report the details
				// and allow the watching to continue. We leave it to the user to decide if
				// they want to quit the run command.
			}
		}
	}()

	// The watcher documentation suggests we tell the watcher about the directory to watch,
	// AFTER we have constructed it, and registered a handler.
	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}

	if hasCustomFunctions {
		err = watcher.Add(filepath.Join(dir, "functions"))

		if err != nil {
			if os.IsNotExist(err) {
				// todo: maybe create this directory
				return nil, errors.New("'functions' directory not found")
			}

			return nil, err
		}
	}

	return watcher.Close, nil
}

func isRelevantEventType(op fsnotify.Op) bool {
	relevant := []fsnotify.Op{fsnotify.Create, fsnotify.Write, fsnotify.Remove}
	// The irrelevant ones are Rename and Chmod.
	return lo.Contains(relevant, op)
}

var runCmdFlagReset bool
var runCmdFlagVerbose bool
var runCmdFlagPort string

func init() {
	rootCmd.AddCommand(runCmd)

	defaultDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	runCmd.Flags().StringVarP(&inputDir, "dir", "d", defaultDir, "the directory containing the Keel schema files")
	runCmd.Flags().BoolVar(&runCmdFlagReset, "reset", false, "if set the database will be reset")
	runCmd.Flags().BoolVarP(&runCmdFlagVerbose, "verbose", "v", false, "print database logs")
	runCmd.Flags().StringVar(&runCmdFlagPort, "port", "8000", "the port to run the Keel application on")
}
