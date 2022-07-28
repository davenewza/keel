package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/stretchr/testify/require"
	"github.com/teamkeel/keel/migrations"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"github.com/teamkeel/keel/schema"
	"github.com/teamkeel/keel/schema/reader"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRuntime(t *testing.T) {
	// We connect to the "main" database here only so we can create a new database
	// for each sub-test
	mainDB, err := gorm.Open(
		postgres.Open(fmt.Sprintf(dbConnString, "keel")),
		&gorm.Config{})
	require.NoError(t, err)

	for _, tCase := range testCases {

		t.Run(tCase.name, func(t *testing.T) {

			// Make a database name for this test
			re := regexp.MustCompile(`[^\w]`)
			dbName := strings.ToLower(re.ReplaceAllString(tCase.name, ""))

			// Drop the database if it already exists. The normal dropping of it at the end of the
			// test case is bypassed if you quit a debug run of the test in VS Code.
			require.NoError(t, mainDB.Exec("DROP DATABASE if exists "+dbName).Error)

			// Create the database and drop at the end of the test
			err = mainDB.Exec("CREATE DATABASE " + dbName).Error
			require.NoError(t, err)
			defer func() {
				require.NoError(t, mainDB.Exec("DROP DATABASE "+dbName).Error)
			}()

			// Connect to the newly created test database and close connection
			// at the end of the test. We need to explicitly close the connection
			// so the mainDB connection can drop the database.
			testDB, err := gorm.Open(
				postgres.Open(fmt.Sprintf(dbConnString, dbName)),
				&gorm.Config{})
			require.NoError(t, err)
			defer func() {
				conn, err := testDB.DB()
				require.NoError(t, err)
				conn.Close()
			}()

			// Migrate the database to this test case's schema.
			schema := protoSchema(t, tCase.keelSchema)
			m := migrations.New(schema, nil)
			require.NoError(t, m.Apply(testDB))

			// Construct the runtime API Handler.
			handler := NewHandler(schema)

			// Assemble the query to send from the test case data.
			reqBody := queryAsJSONPayload(t, tCase.gqlOperation, tCase.variables)
			request := Request{
				Context: runtimectx.NewContext(testDB),
				URL: url.URL{
					Path: "/Test",
				},
				Body: []byte(reqBody),
			}

			// Call the handler, and capture the response.
			response, err := handler(&request)
			body := string(response.Body)
			require.NoError(t, err)

			// Do the specified assertion on the data returned, if one is specified.
			if tCase.assertData != nil {
				var r respFields
				require.NoError(t, json.Unmarshal([]byte(body), &r))
				tCase.assertData(t, r.Data)
			}

			// Do the specified assertion on the errors returned, if one is specified.
			if tCase.assertErrors != nil {
				var r respFields
				require.NoError(t, json.Unmarshal([]byte(body), &r))
				tCase.assertErrors(t, r.Errors)
			}

			// Do the specified assertion on the resultant database contents, if one is specified.
			if tCase.assertDatabase != nil {
				var r respFields
				require.NoError(t, json.Unmarshal([]byte(body), &r))
				tCase.assertDatabase(t, testDB, r.Data)
			}
		})
	}
}

type respFields struct {
	Data   map[string]any             `json:"data"`
	Errors []gqlerrors.FormattedError `json:"errors"`
}

const dbConnString = "host=localhost port=8001 user=postgres password=postgres dbname=%s sslmode=disable"

func protoSchema(t *testing.T, keelSchema string) *proto.Schema {
	builder := &schema.Builder{}
	schema, err := builder.MakeFromInputs(&reader.Inputs{
		SchemaFiles: []reader.SchemaFile{
			{
				Contents: keelSchema,
			},
		},
	})
	require.NoError(t, err)
	return schema
}

func queryAsJSONPayload(t *testing.T, mutationString string, vars map[string]any) (asJSON string) {
	d := map[string]any{
		"query":     mutationString,
		"variables": vars,
	}
	b, err := json.Marshal(d)
	require.NoError(t, err)
	return string(b)
}

type testCase struct {
	name           string
	keelSchema     string
	databaseSetup  func(t *testing.T, db *gorm.DB)
	gqlOperation   string
	variables      map[string]any
	assertData     func(t *testing.T, data map[string]any)
	assertErrors   func(t *testing.T, errors []gqlerrors.FormattedError)
	assertDatabase func(t *testing.T, db *gorm.DB, data map[string]any)
}

var testCases = []testCase{
	{
		name: "create_operation_happy",
		keelSchema: `
			model Person {
				fields {
					name Text
				}
				operations {
					get getPerson(id)
					create createPerson() with (name)
				}
			}
			api Test {
				@graphql
				models {
					Person
				}
			}
		`,
		gqlOperation: `
			mutation CreatePerson($name: String!) {
				createPerson(input: {name: $name}) {
					id
					name
				}
			}
		`,
		variables: map[string]any{
			"name": "Fred",
		},
		assertData: func(t *testing.T, data map[string]any) {
			assertValueAtPath(t, data, "createPerson.name", "Fred")
		},
		assertErrors: func(t *testing.T, errors []gqlerrors.FormattedError) {
		},
		assertDatabase: func(t *testing.T, db *gorm.DB, data map[string]any) {
			id := getValueAtPath(t, data, "createPerson.id")
			var name string
			err := db.Table("person").Where("id = ?", id).Pluck("name", &name).Error
			require.NoError(t, err)
			require.Equal(t, "Fred", name)
		},
	},

	{
		name: "create_operation_errors",
		keelSchema: `
			model Person {
				fields {
					name Text
				}
				operations {
					get getPerson(id)
					create createPerson() with (name)
				}
			}
			api Test {
				@graphql
				models {
					Person
				}
			}
		`,
		gqlOperation: `
			mutation CreatePerson($name: String!) {
				createPerson(input: {name: $name}) {
					nosuchfield
				}
			}
		`,
		variables: map[string]any{
			"name": "Fred",
		},
		assertErrors: func(t *testing.T, errors []gqlerrors.FormattedError) {
			require.Len(t, errors, 1)
			require.Equal(t, "Cannot query field \"nosuchfield\" on type \"Person\".", errors[0].Message)
		},
	},
}
