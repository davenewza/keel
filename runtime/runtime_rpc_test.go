package runtime_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime"
	"github.com/teamkeel/keel/runtime/jsonschema"
	"github.com/teamkeel/keel/runtime/runtimectx"
	rtt "github.com/teamkeel/keel/runtime/runtimetest"
	"github.com/teamkeel/keel/testhelpers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRuntimeRPC(t *testing.T) {
	// We connect to the "main" database here only so we can create a new database
	// for each sub-test

	for _, tCase := range rpcTestCases {
		t.Run(tCase.name, func(t *testing.T) {
			schema := protoSchema(t, tCase.keelSchema)

			// Use the docker compose database
			dbConnInfo := &db.ConnectionInfo{
				Host:     "localhost",
				Port:     "8001",
				Username: "postgres",
				Database: "keel",
				Password: "postgres",
			}

			handler := runtime.NewHandler(schema)

			request := &http.Request{
				URL: &url.URL{
					Path:     "/test/json/" + tCase.Path,
					RawQuery: tCase.QueryParams,
				},
				Method: tCase.Method,
				Body:   io.NopCloser(strings.NewReader(tCase.Body)),
			}

			ctx := request.Context()

			dbName := testhelpers.DbNameForTestName(tCase.name)
			testDB, err := testhelpers.SetupDatabaseForTestCase(ctx, dbConnInfo, schema, dbName)
			require.NoError(t, err)

			database, err := db.NewFromConnection(ctx, testDB)
			require.NoError(t, err)

			ctx = runtimectx.WithDatabase(ctx, database)
			request = request.WithContext(ctx)

			// We are still using gorm database to assert againt the data in the test databases.
			gormDb, err := gorm.Open(postgres.New(postgres.Config{Conn: testDB}), &gorm.Config{})
			require.NoError(t, err)

			// Apply the database prior-set up mandated by this test case.
			if tCase.databaseSetup != nil {
				tCase.databaseSetup(t, gormDb)
			}

			// Call the handler, and capture the response.
			response := handler(request)
			body := string(response.Body)
			var res map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &res))

			// Do the specified assertion on the resultant database contents, if one is specified.
			if tCase.assertDatabase != nil {
				tCase.assertDatabase(t, gormDb, res)
			}

			if response.Status != 200 && tCase.assertError == nil {
				t.Errorf("method %s returned non-200 (%d) but no assertError function provided", tCase.Path, response.Status)
			}
			if tCase.assertError != nil {
				tCase.assertError(t, res, response.Status)
			}

			// Do the specified assertion on the data returned, if one is specified.
			if tCase.assertResponse != nil {
				tCase.assertResponse(t, res)
			}

			op := proto.FindOperation(schema, tCase.Path)

			_, result, err := jsonschema.ValidateResponse(ctx, schema, op, res)

			assert.NoError(t, err)

			if !result.Valid() {
				msg := ""

				for _, err := range result.Errors() {
					msg += fmt.Sprintf("%s\n", err.String())
				}
				assert.Fail(t, msg)
			}
		})
	}
}

var rpcTestCases = []rpcTestCase{
	{
		name: "rpc_list_http_get",
		keelSchema: `
		model Thing {
			operations {
				list listThings()
			}
			@permission(
				expression: true,
				actions: [list]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row1 := initRow(map[string]any{
				"id": "id_123",
			})
			require.NoError(t, db.Table("thing").Create(row1).Error)
		},
		Path:   "listThings",
		Method: http.MethodGet,
		assertResponse: func(t *testing.T, res map[string]any) {
			results := res["results"].([]interface{})
			require.Len(t, results, 1)
			pageInfo := res["pageInfo"].(map[string]any)

			hasNextPage := pageInfo["hasNextPage"].(bool)
			require.Equal(t, false, hasNextPage)
		},
	},
	{
		name: "rpc_list_http_post",
		keelSchema: `
		model Thing {
			fields {
				text Text
			}
			operations {
				list listThings(text)
			}
			@permission(
				expression: true,
				actions: [list]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row := initRow(map[string]any{
				"id":   "id_1",
				"text": "foobar",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
			row = initRow(map[string]any{
				"id":   "id_2",
				"text": "foobaz",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
			row = initRow(map[string]any{
				"id":   "id_3",
				"text": "boop",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
		},
		Path:   "listThings",
		Body:   `{"where": { "text": { "startsWith": "foo" } }}`,
		Method: http.MethodPost,
		assertResponse: func(t *testing.T, res map[string]any) {
			results := res["results"].([]interface{})
			require.Len(t, results, 2)
			pageInfo := res["pageInfo"].(map[string]any)

			hasNextPage := pageInfo["hasNextPage"].(bool)
			require.Equal(t, false, hasNextPage)
		},
	},
	{
		name: "rpc_list_paging",
		keelSchema: `
		model Thing {
			fields {
				text Text
			}
			operations {
				list listThings()
			}
			@permission(
				expression: true,
				actions: [list]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row1 := initRow(map[string]any{
				"id":   "id_1",
				"text": "foobar",
			})
			require.NoError(t, db.Table("thing").Create(row1).Error)
			row2 := initRow(map[string]any{
				"id":   "id_2",
				"text": "foobaz",
			})
			require.NoError(t, db.Table("thing").Create(row2).Error)
			row3 := initRow(map[string]any{
				"id":   "id_3",
				"text": "boop",
			})
			require.NoError(t, db.Table("thing").Create(row3).Error)
			row4 := initRow(map[string]any{
				"id":   "id_4",
				"text": "boop",
			})
			require.NoError(t, db.Table("thing").Create(row4).Error)
		},
		Path:   "listThings",
		Body:   `{"where": { }, "first": 2}`,
		Method: http.MethodPost,
		assertResponse: func(t *testing.T, res map[string]any) {
			results := res["results"].([]interface{})
			require.Len(t, results, 2)

			pageInfo := res["pageInfo"].(map[string]any)

			hasNextPage := pageInfo["hasNextPage"].(bool)
			require.Equal(t, true, hasNextPage)

			assert.Equal(t, "id_2", pageInfo["endCursor"].(string))

			totalCount := pageInfo["totalCount"].(float64)
			assert.Equal(t, float64(4), totalCount)
		},
	},
	{
		name: "rpc_get_http_get",
		keelSchema: `
		model Thing {
			operations {
				get getThing(id)
			}
			@permission(
				expression: true,
				actions: [get]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row := initRow(map[string]any{
				"id": "id_1",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
		},
		Path:        "getThing",
		QueryParams: "id=id_1",
		Method:      http.MethodGet,
		assertResponse: func(t *testing.T, data map[string]any) {
			require.Equal(t, data["id"], "id_1")
		},
	},
	{
		name: "rpc_get_http_post",
		keelSchema: `
		model Thing {
			operations {
				get getThing(id)
			}
			@permission(
				expression: true,
				actions: [get]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row := initRow(map[string]any{
				"id": "id_1",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
		},
		Path:   "getThing",
		Body:   `{"id": "id_1"}`,
		Method: http.MethodPost,
		assertResponse: func(t *testing.T, data map[string]any) {
			require.Equal(t, data["id"], "id_1")
		},
	},
	{
		name: "rpc_create_http_post",
		keelSchema: `
		model Thing {
			fields {
				text Text
			}
			operations {
				create createThing() with (text)
			}
			@permission(
				expression: true,
				actions: [create]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		Path:   "createThing",
		Body:   `{"text": "foo"}`,
		Method: http.MethodPost,
		assertDatabase: func(t *testing.T, db *gorm.DB, data interface{}) {
			res := data.(map[string]any)
			id := res["id"]

			row := map[string]any{}
			err := db.Table("thing").Where("id = ?", id).Scan(&row).Error
			require.NoError(t, err)

			require.Equal(t, "foo", row["text"])
		},
	},
	{
		name: "rpc_update_http_post",
		keelSchema: `
		model Thing {
			fields {
				text Text
			}
			operations {
				update updateThing(id) with (text)
			}
			@permission(
				expression: true,
				actions: [update]
			)
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		Path:   "updateThing",
		Body:   `{"where": {"id": "id_1"}, "values": {"text": "new value"}}`,
		Method: http.MethodPost,
		databaseSetup: func(t *testing.T, db *gorm.DB) {
			row := initRow(map[string]any{
				"id":   "id_1",
				"text": "foo",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
			row = initRow(map[string]any{
				"id":   "id_2",
				"text": "bar",
			})
			require.NoError(t, db.Table("thing").Create(row).Error)
		},
		assertDatabase: func(t *testing.T, db *gorm.DB, data interface{}) {
			res := data.(map[string]any)
			// check returned values
			require.Equal(t, "id_1", res["id"])
			require.Equal(t, "new value", res["text"])

			// check row 1 changed
			row := map[string]any{}
			err := db.Table("thing").Where("id = ?", "id_1").Scan(&row).Error
			require.NoError(t, err)
			require.Equal(t, "new value", row["text"])

			// check row 2 did not change
			row = map[string]any{}
			err = db.Table("thing").Where("id = ?", "id_2").Scan(&row).Error
			require.NoError(t, err)
			require.Equal(t, "bar", row["text"])
		},
	},
	{
		name: "rpc_json_schema_errors",
		keelSchema: `
		model Thing {
			operations {
				get getThing(id)
			}
		}
		api Test {
			models {
				Thing
			}
		}
	`,
		Path:   "getThing",
		Body:   `{"total": "nonsense"}`,
		Method: http.MethodPost,
		assertError: func(t *testing.T, data map[string]any, statusCode int) {
			assert.Equal(t, statusCode, http.StatusBadRequest)
			assert.Equal(t, "ERR_INVALID_INPUT", data["code"])
			rtt.AssertValueAtPath(t, data, "data.errors[0].field", "(root)")
			rtt.AssertValueAtPath(t, data, "data.errors[0].error", "id is required")
			rtt.AssertValueAtPath(t, data, "data.errors[1].field", "(root)")
			rtt.AssertValueAtPath(t, data, "data.errors[1].error", "Additional property total is not allowed")
		},
	},
}