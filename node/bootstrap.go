package node

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/samber/lo"
	codegenerator "github.com/teamkeel/keel/node/codegen"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema"
)

// Public API of this package:
// - Bootstrap: generates a package.json for a project if one isn't currently provided containing all of the dependencies required by the Keel JS runtime
// - GeneratePackages - generates dynamic JavaScript + TypeScript definitions based on a Keel schema. Generates both SDK generated code, as well as additional code required to make the testing framework work correctly.
// - GenerateDevelopmentServer - generates handler code used to run the Keel JavaScript runtime in local development
// - RunDevelopmentServer - runs the handler / server code generated by GenerateDevelopmentServer.

// Bootstrap is responsible for creating a simple package.json file in the target directory, containing all of the dependencies required by the Keel JavaScript runtime.
func Bootstrap(dir string) error {
	_, err := os.Stat(filepath.Join(dir, "package.json"))

	if err != nil && errors.Is(err, os.ErrExist) {
		return err
	}

	// Make a package JSON for the temp dir that references my-package
	// as a local package
	err = os.WriteFile(filepath.Join(dir, "package.json"), []byte(fmt.Sprintf(
		`{
		"name": "%s",
		"dependencies": {
			"@teamkeel/testing": "*",
			"@teamkeel/sdk":     "*",
			"@teamkeel/runtime": "*",
			"ts-node":           "*",
			"@swc/core":           "*",
			"regenerator-runtime": "*"
		}
	}`, filepath.Base(dir),
	)), 0644)

	if err != nil {
		return err
	}

	npmInstall := exec.Command("npm", "install", "--progress=false", "--no-audit")
	npmInstall.Dir = dir

	o, err := npmInstall.CombinedOutput()

	if err != nil {
		fmt.Print(string(o))
		return err
	}

	return nil
}

// Generates @teamkeel/sdk and @teamkeel/testing NPM packages in the target directory's node_modules
func GeneratePackages(dir string) error {
	builder := schema.Builder{}

	schema, err := builder.MakeFromDirectory(dir)

	if err != nil {
		return err
	}

	// Dont do any code generation if there are no functions in the schema
	// or any Keel tests defined
	if !hasFunctions(schema) && !hasTests(dir) {
		return nil
	}

	cg := codegenerator.NewGenerator(schema, dir)

	_, err = cg.GenerateSDK()

	if err != nil {
		return err
	}

	_, err = cg.GenerateTesting()

	if err != nil {
		return err
	}

	return nil
}

// GenerateDevelopmentServer is responsible for generating a single index.js file in .build in the target directory. This file contains the code responsible for handling requests to a custom function, and also instantiates a simple HTTP server wrapper around the handler.
func GenerateDevelopmentServer(dir string) error {
	builder := schema.Builder{}

	schema, err := builder.MakeFromDirectory(dir)

	if err != nil {
		return err
	}

	// Dont do any code generation if there are no functions in the schema
	// or any Keel tests defined
	if !hasFunctions(schema) && !hasTests(dir) {
		return nil
	}

	cg := codegenerator.NewGenerator(schema, dir)

	_, err = cg.GenerateDevelopmentHandler()

	if err != nil {
		return err
	}

	return nil
}

type RuntimeServer interface {
	Kill() error
}

type DevelopmentServer struct {
	proc *os.Process
}

func (ds *DevelopmentServer) Kill() error {
	return ds.proc.Kill()
}

type ServerOpts struct {
	Port    int
	EnvVars map[string]any
	Stdout  io.Writer
	Stderr  io.Writer
}

// RunDevelopmentServer will start a new node runtime server serving/handling custom function requests
func RunDevelopmentServer(dir string, options *ServerOpts) (RuntimeServer, error) {
	// 1. run dev server with ts-node.
	handlerPath := filepath.Join(dir, codegenerator.BuildDirName, "index.js")

	cmd := exec.Command("npx", "ts-node", handlerPath)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	if options != nil {
		cmd.Stdout = options.Stdout
		cmd.Stderr = options.Stderr

		cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", options.Port))

		for key, value := range options.EnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	err := cmd.Start()

	if err != nil {
		return nil, err
	}

	return &DevelopmentServer{
		proc: cmd.Process,
	}, nil
}

func hasFunctions(sch *proto.Schema) bool {
	var ops []*proto.Operation

	for _, model := range sch.Models {
		ops = append(ops, model.Operations...)
	}

	return lo.SomeBy(ops, func(o *proto.Operation) bool {
		return o.Implementation == proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM
	})
}

func hasTests(dir string) bool {
	fs := os.DirFS(dir)

	// the only potential error returned from glob here is bad pattern,
	// which we know not to be true
	testFiles, _ := doublestar.Glob(fs, "**/*.test.ts")

	// there could be other *.test.ts files unrelated to the Keel testing framework,
	// so for each test, we do a naive check that the file contents includes a match
	// for the string "@teamkeel/testing"
	return lo.SomeBy(testFiles, func(path string) bool {
		b, err := os.ReadFile(path)

		if err != nil {
			return false
		}

		// todo: improve this check as its pretty naive
		return strings.Contains(string(b), "@teamkeel/testing")
	})
}
