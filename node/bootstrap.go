package node

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type bootstrapOptions struct {
	packagesPath string
}

// WithPackagesPath causes any @teamkeel packages to be installed
// from this path. The path should point to the directory that contains
// all the different @teamkeel packages.
func WithPackagesPath(p string) func(o *bootstrapOptions) {
	return func(o *bootstrapOptions) {
		o.packagesPath = p
	}
}

// Bootstrap sets dir up to use either custom functions or write tests. It will do nothing
// if there is already a package.json present in the directory.
func Bootstrap(dir string, opts ...func(o *bootstrapOptions)) error {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	// No error - we have a package.json so we're done
	if err == nil {
		return nil
	}
	// A "not exists" error is fine, that means we're generating a fresh package.json
	// Bail on all other errors
	if !os.IsNotExist(err) {
		return err
	}

	options := &bootstrapOptions{}
	for _, o := range opts {
		o(options)
	}

	functionsRuntimeVersion := "*"
	if options.packagesPath != "" {
		functionsRuntimeVersion = filepath.Join(options.packagesPath, "functions-runtime")
	}

	err = os.WriteFile(filepath.Join(dir, "package.json"), []byte(fmt.Sprintf(`{
		"name": "%s",
		"dependencies": {
			"@teamkeel/functions-runtime": "%s",
			"@types/node": "^18.11.18",
			"ts-node": "^10.9.1",
			"typescript": "^4.9.4"
		}
	}`, filepath.Base(dir), functionsRuntimeVersion)), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
		"compilerOptions": {
			"lib": ["ES2016"],
			"target": "ES2016",
			"esModuleInterop": true,
			"moduleResolution": "node",
			"skipLibCheck": true,
			"strictNullChecks": true,
			"types": ["node"],
			"allowJs": true
		},
		"include": ["**/*.ts"],
		"exclude": ["node_modules"]
	}`), 0644)
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
