package testing

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/teamkeel/keel/cmd/database"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/functions"
	"github.com/teamkeel/keel/node"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"github.com/teamkeel/keel/schema"
	"github.com/teamkeel/keel/testhelpers"
	"github.com/teamkeel/keel/util"
)

type TestOutput struct {
	Output  string
	Success bool
}

type RunnerOpts struct {
	Dir        string
	Pattern    string
	DbConnInfo *database.ConnectionInfo
}

func Run(opts *RunnerOpts) (*TestOutput, error) {
	builder := &schema.Builder{}
	schema, err := builder.MakeFromDirectory(opts.Dir)
	if err != nil {
		return nil, err
	}

	testApi := &proto.Api{
		// TODO: make random so doesn't clash
		Name: "TestingActionsApi",
	}
	for _, m := range schema.Models {
		testApi.ApiModels = append(testApi.ApiModels, &proto.ApiModel{
			ModelName: m.Name,
		})
	}

	schema.Apis = append(schema.Apis, testApi)

	context := context.Background()

	dbName := "keel_test"
	mainDB, err := testhelpers.SetupDatabaseForTestCase(context, opts.DbConnInfo, schema, dbName)
	if err != nil {
		return nil, err
	}

	dbConnString := opts.DbConnInfo.WithDatabase(dbName).String()

	files, err := node.Generate(context, opts.Dir, node.WithDevelopmentServer(true))
	if err != nil {
		return nil, err
	}

	err = files.Write()
	if err != nil {
		return nil, err
	}

	var functionsServer *node.DevelopmentServer
	var functionsTransport functions.Transport

	if node.HasFunctions(schema) {
		functionsServer, err = node.RunDevelopmentServer(opts.Dir, &node.ServerOpts{
			EnvVars: map[string]string{
				"DB_CONN_TYPE": "pg",
				"DB_CONN":      dbConnString,
			},
		})
		if err != nil {
			if functionsServer != nil && functionsServer.Output() != "" {
				return nil, errors.New(functionsServer.Output())
			}
			return nil, err
		}

		defer functionsServer.Kill()

		functionsTransport = functions.NewHttpTransport(functionsServer.URL)
	}

	runtimePort, err := util.GetFreePort()
	if err != nil {
		return nil, err
	}

	// Server to handle API calls to the runtime
	runtimeServer := http.Server{
		Addr: fmt.Sprintf(":%s", runtimePort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()
			database, _ := db.LocalFromConnection(ctx, mainDB)
			ctx = runtimectx.WithDatabase(ctx, database)
			if functionsTransport != nil {
				ctx = functions.WithFunctionsTransport(ctx, functionsTransport)
			}
			r = r.WithContext(ctx)

			runtime.NewHttpHandler(schema).ServeHTTP(w, r)
		}),
	}
	go runtimeServer.ListenAndServe()
	defer runtimeServer.Shutdown(context)

	cmd := exec.Command("npx", "tsc", "--noEmit", "--pretty")
	cmd.Dir = opts.Dir

	b, err := cmd.CombinedOutput()
	exitError := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitError) {
		return nil, err
	}
	if err != nil {
		return &TestOutput{Output: string(b), Success: false}, nil
	}

	if opts.Pattern == "" {
		opts.Pattern = "(.*)"
	}

	cmd = exec.Command("npx", "vitest", "run", "--color", "--reporter", "verbose", "--config", "./node_modules/@teamkeel/testing-runtime/vitest.config.mjs", "--testNamePattern", opts.Pattern)
	cmd.Dir = opts.Dir
	cmd.Env = append(os.Environ(), []string{
		fmt.Sprintf("KEEL_TESTING_ACTIONS_API_URL=http://localhost:%s/testingactionsapi/json", runtimePort),
		"DB_CONN_TYPE=pg",
		fmt.Sprintf("DB_CONN=%s", dbConnString),
	}...)

	b, err = cmd.CombinedOutput()
	if err != nil && !errors.As(err, &exitError) {
		return nil, err
	}

	return &TestOutput{Output: string(b), Success: err == nil}, nil
}
