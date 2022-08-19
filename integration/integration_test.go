package integration_test

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	gotest "testing"

	"github.com/nsf/jsondiff"
	"github.com/teamkeel/keel/nodedeps"
	"github.com/teamkeel/keel/testhelpers"
	"github.com/teamkeel/keel/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *gotest.T) {
	entries, err := ioutil.ReadDir("./testdata")
	require.NoError(t, err)

	for _, e := range entries {
		t.Run(e.Name(), func(t *gotest.T) {
			workingDir, err := testhelpers.WithTmpDir(filepath.Join("./testdata", e.Name()))

			require.NoError(t, err)

			packageJson, err := nodedeps.NewPackageJson(filepath.Join(workingDir, "package.json"))

			require.NoError(t, err)

			// todo: to save time during test suite run, do the package.json creation
			// plus injection only once per suite.
			err = packageJson.Inject(map[string]string{
				"@teamkeel/testing": "*",
				"@teamkeel/sdk":     "*",
				"@teamkeel/runtime": "*",
				"ts-node":           "*",
				// https://typestrong.org/ts-node/docs/swc/
				"@swc/core":           "*",
				"regenerator-runtime": "*",
			}, true)

			require.NoError(t, err)

			ch, err := testing.Run(workingDir)
			require.NoError(t, err)

			events := []*testing.Event{}
			for newEvents := range ch {
				events = append(events, newEvents...)
			}

			actual, err := json.Marshal(events)
			require.NoError(t, err)

			expected, err := ioutil.ReadFile(filepath.Join("./testdata", e.Name(), "expected.json"))
			require.NoError(t, err)

			opts := jsondiff.DefaultConsoleOptions()

			diff, explanation := jsondiff.Compare(expected, actual, &opts)
			if diff == jsondiff.FullMatch {
				return
			}

			assert.Fail(t, "actual test output did not match expected", explanation)
		})
	}
}