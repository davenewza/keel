package jsonschema_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/jsonschema"
	"github.com/teamkeel/keel/schema"
)

func TestValidateRequest(t *testing.T) {
	type fixture struct {
		name    string
		opName  string
		request string
		errors  map[string]string
	}

	type fixtureGroup struct {
		name   string
		schema string
		cases  []fixture
	}

	fixtures := []fixtureGroup{
		{
			name: "get action",
			schema: `
				model Person {
					fields {
						name Text @unique
					}
					operations {
						get getPerson(id)
						get getBestBeatle() {
							@where(person.name == "John Lennon")
						}
					}
				}
			`,
			cases: []fixture{
				{
					name:    "valid - with input",
					request: `{"id": "1234"}`,
					opName:  "getPerson",
				},
				{
					name:    "valid - without input",
					request: `{}`,
					opName:  "getBestBeatle",
				},

				// errors
				{
					name:    "missing input",
					request: `{}`,
					opName:  "getPerson",
					errors: map[string]string{
						"(root)": "id is required",
					},
				},
				{
					name:    "wrong type",
					request: `{"id": 1234}`,
					opName:  "getPerson",
					errors: map[string]string{
						"id": "Invalid type. Expected: string, given: integer",
					},
				},
				{
					name:    "null",
					request: `{"id": null}`,
					opName:  "getPerson",
					errors: map[string]string{
						"id": "Invalid type. Expected: string, given: null",
					},
				},
				{
					name:    "valid inputs with additional properties",
					request: `{"id": "1234", "foo": "bar"}`,
					opName:  "getPerson",
					errors: map[string]string{
						"(root)": "Additional property foo is not allowed",
					},
				},
				{
					name:    "additional properties when no inputs expected",
					request: `{"id": "1234"}`,
					opName:  "getBestBeatle",
					errors: map[string]string{
						"(root)": "Additional property id is not allowed",
					},
				},
			},
		},
		{
			name: "delete action",
			schema: `
				model Person {
					fields {
						name Text @unique
					}
					operations {
						delete deletePerson(id)
						delete deleteBob() {
							@where(person.name == "Bob")
						}
					}
				}
			`,
			cases: []fixture{
				{
					name:    "valid - with input",
					request: `{"id": "1234"}`,
					opName:  "deletePerson",
				},
				{
					name:    "valid - without input",
					request: `{}`,
					opName:  "deleteBob",
				},

				// errors
				{
					name:    "missing input",
					request: `{}`,
					opName:  "deletePerson",
					errors: map[string]string{
						"(root)": "id is required",
					},
				},
				{
					name:    "wrong type",
					request: `{"id": 1234}`,
					opName:  "deletePerson",
					errors: map[string]string{
						"id": "Invalid type. Expected: string, given: integer",
					},
				},
				{
					name:    "null",
					request: `{"id": null}`,
					opName:  "deletePerson",
					errors: map[string]string{
						"id": "Invalid type. Expected: string, given: null",
					},
				},
				{
					name:    "valid inputs with additional properties",
					request: `{"id": "1234", "foo": "bar"}`,
					opName:  "deletePerson",
					errors: map[string]string{
						"(root)": "Additional property foo is not allowed",
					},
				},
				{
					name:    "additional properties when no inputs expected",
					request: `{"id": "1234"}`,
					opName:  "deleteBob",
					errors: map[string]string{
						"(root)": "Additional property id is not allowed",
					},
				},
			},
		},
		{
			name: "create action",
			schema: `
				model Person {
					fields {
						name Text
						birthday Date?
					}
					operations {
						create createPerson() with (name)
						create createPersonWithDOB() with (name, birthday)
						create createPersonWithOptionalDOB() with (name, birthday?)
					}
				}
			`,
			cases: []fixture{
				{
					name:    "valid - basic",
					request: `{"name": "Jon"}`,
					opName:  "createPerson",
				},
				{
					name:    "valid - input for optional field provided",
					request: `{"name": "Jon", "birthday": "1986-03-18"}`,
					opName:  "createPersonWithDOB",
				},
				{
					name:    "valid - input for optional field provided as null",
					request: `{"name": "Jon", "birthday": null}`,
					opName:  "createPersonWithDOB",
				},
				{
					name:    "valid - ommitting optional input",
					request: `{"name": "Jon"}`,
					opName:  "createPersonWithOptionalDOB",
				},
				{
					name:    "valid - providing optional input for optional field as null",
					request: `{"name": "Jon", "birthday": null}`,
					opName:  "createPersonWithOptionalDOB",
				},
				{
					name:    "valid - providing optional input for optional field",
					request: `{"name": "Jon", "birthday": "1986-03-18"}`,
					opName:  "createPersonWithOptionalDOB",
				},

				// errors
				{
					name:    "missing input",
					request: `{}`,
					opName:  "createPerson",
					errors: map[string]string{
						"(root)": "name is required",
					},
				},
				{
					name:    "missing required input for optional field",
					request: `{"name": "Jon"}`,
					opName:  "createPersonWithDOB",
					errors: map[string]string{
						"(root)": "birthday is required",
					},
				},
				{
					name:    "null",
					request: `{"name": null}`,
					opName:  "createPerson",
					errors: map[string]string{
						"name": "Invalid type. Expected: string, given: null",
					},
				},
				{
					name:    "wrong type",
					request: `{"name": 1234}`,
					opName:  "createPerson",
					errors: map[string]string{
						"name": "Invalid type. Expected: string, given: integer",
					},
				},
				{
					name:    "wrong format for date",
					request: `{"name": "Jon", "birthday": "18th March 1986"}`,
					opName:  "createPersonWithDOB",
					errors: map[string]string{
						"birthday": "Does not match format 'date'",
					},
				},
				{
					name:    "additional properties",
					request: `{"name": "Bob", "foo": "bar"}`,
					opName:  "createPerson",
					errors: map[string]string{
						"(root)": "Additional property foo is not allowed",
					},
				},
			},
		},
		{
			name: "update action",
			schema: `
				model Person {
					fields {
						identity Identity @unique
						name Text
						nickName Text?
					}
					operations {
						update updateName(id) with (name)
						update updateNameAndNickname(id) with (name, nickName)
						update updateNameOrNickname(id) with (name?, nickName?)
						update updateMyPerson() {
							@where(person.identity == ctx.identity)
							@set(person.name = "Hello")
						}
						update updateMyPersonWithName() with (name) {
							@where(person.identity == ctx.identity)
						}
					}
				}
			`,
			cases: []fixture{
				{
					name:    "valid - one input",
					request: `{"where": {"id": "1234"}, "values": {"name": "Jon"}}`,
					opName:  "updateName",
				},
				{
					name:    "valid - two inputs",
					request: `{"where": {"id": "1234"}, "values": {"name": "Jon", "nickName": "Johnny"}}`,
					opName:  "updateNameAndNickname",
				},
				{
					name:    "valid - two inputs - null for optional field",
					request: `{"where": {"id": "1234"}, "values": {"name": "Jon", "nickName": null}}`,
					opName:  "updateNameAndNickname",
				},
				{
					name:    "valid - two inputs - both optional - both provided",
					request: `{"where": {"id": "1234"}, "values": {"name": "Jon", "nickName": "Johnny"}}`,
					opName:  "updateNameOrNickname",
				},
				{
					name:    "valid - two inputs - both optional - one provided",
					request: `{"where": {"id": "1234"}, "values": {"nickName": "Johnny"}}`,
					opName:  "updateNameOrNickname",
				},
				{
					name:    "valid - two inputs - both optional - neither provided",
					request: `{"where": {"id": "1234"}, "values": {}}`,
					opName:  "updateNameOrNickname",
				},
				{
					name:    "valid - no inputs - empty request is ok",
					request: `{}`,
					opName:  "updateMyPerson",
				},
				{
					name:    "valid - no inputs - empty where and values is ok",
					request: `{"where": {}, "values": {}}`,
					opName:  "updateMyPerson",
				},
				{
					name:    "valid - values but no where",
					request: `{"values": {"name": "Jon"}}`,
					opName:  "updateMyPersonWithName",
				},

				// errors
				{
					name:    "missing required value",
					request: `{"where": {"id": "1234"}, "values": {}}`,
					opName:  "updateName",
					errors: map[string]string{
						"values": "name is required",
					},
				},
				{
					name:    "missing required where",
					request: `{"where": {}, "values": {"name": "Jon"}}`,
					opName:  "updateName",
					errors: map[string]string{
						"where": "id is required",
					},
				},
				{
					name:    "incorrect type for value",
					request: `{"where": {"id": "1234"}, "values": {"name": true}}`,
					opName:  "updateName",
					errors: map[string]string{
						"values.name": "Invalid type. Expected: string, given: boolean",
					},
				},
				{
					name:    "incorrect type for where",
					request: `{"where": {"id": 1234}, "values": {"name": "Jon"}}`,
					opName:  "updateName",
					errors: map[string]string{
						"where.id": "Invalid type. Expected: string, given: integer",
					},
				},
			},
		},
		{
			name: "list action",
			schema: `
				enum Genre {
					Romance
					Horror
				}
				model Author {
					fields {
						name Text
					}
				}
				model Book {
					fields {
						author Author
						title Text
						genre Genre
						price Number
						available Boolean
						releaseDate Date
					}
					operations {
						list listBooks(id?, title?, genre?, price?, available?, createdAt?, releaseDate?)
						list booksByTitleAndGenre(title: Text, genre: Genre, minPrice: Number?) {
							@where(book.title == title)
							@where(book.genre == genre)
							@where(book.price > minPrice)
						}
					}
				}
			`,
			cases: []fixture{
				{
					name:    "valid - no inputs",
					opName:  "listBooks",
					request: `{"where": {}}`,
				},
				{
					name:    "valid - text equals",
					opName:  "listBooks",
					request: `{"where": {"title": {"equals": "Great Gatsby"}}}`,
				},
				{
					name:    "valid - text starts with",
					opName:  "listBooks",
					request: `{"where": {"title": {"startsWith": "Great Gatsby"}}}`,
				},
				{
					name:    "valid - text ends with",
					opName:  "listBooks",
					request: `{"where": {"title": {"startsWith": "Great Gatsby"}}}`,
				},
				{
					name:    "valid - text contains",
					opName:  "listBooks",
					request: `{"where": {"title": {"startsWith": "Great Gatsby"}}}`,
				},
				{
					name:    "valid - text one of",
					opName:  "listBooks",
					request: `{"where": {"title": {"oneOf": ["Great Gatsby", "Ulysses"]}}}`,
				},
				{
					name:    "valid - text multi",
					opName:  "listBooks",
					request: `{"where": {"title": {"startsWith": "Great", "endsWith": "Gatsby"}}}`,
				},
				{
					name:    "valid - enum equals",
					opName:  "listBooks",
					request: `{"where": {"genre": {"equals": "Romance"}}}`,
				},
				{
					name:    "valid - enum one of",
					opName:  "listBooks",
					request: `{"where": {"genre": {"oneOf": ["Romance", "Horror"]}}}`,
				},
				{
					name:    "valid - number equals",
					opName:  "listBooks",
					request: `{"where": {"price": {"equals": 10}}}`,
				},
				{
					name:    "valid - number less than",
					opName:  "listBooks",
					request: `{"where": {"price": {"lessThan": 10}}}`,
				},
				{
					name:    "valid - number greater than",
					opName:  "listBooks",
					request: `{"where": {"price": {"greaterThan": 10}}}`,
				},
				{
					name:    "valid - number less than or equals",
					opName:  "listBooks",
					request: `{"where": {"price": {"lessThanOrEquals": 10}}}`,
				},
				{
					name:    "valid - number greater than or equals",
					opName:  "listBooks",
					request: `{"where": {"price": {"greaterThanOrEquals": 10}}}`,
				},
				{
					name:    "valid - boolean equals",
					opName:  "listBooks",
					request: `{"where": {"available": {"equals": true}}}`,
				},
				{
					name:    "valid - timestamp before",
					opName:  "listBooks",
					request: `{"where": {"createdAt": {"before": "2022-12-02T12:28:29.609Z"}}}`,
				},
				{
					name:    "valid - timestamp after",
					opName:  "listBooks",
					request: `{"where": {"createdAt": {"after": "2022-12-02T12:28:29.609Z"}}}`,
				},
				{
					name:    "valid - date equals",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"equals": "2022-12-02"}}}`,
				},
				{
					name:    "valid - date before",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"before": "2022-12-02"}}}`,
				},
				{
					name:    "valid - date on or before",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"onOrBefore": "2022-12-02"}}}`,
				},
				{
					name:    "valid - date after",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"after": "2022-12-02"}}}`,
				},
				{
					name:    "valid - date on or after",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"onOrAfter": "2022-12-02"}}}`,
				},
				{
					name:    "valid - id equals",
					opName:  "listBooks",
					request: `{"where": {"id": {"equals": "123456789"}}}`,
				},
				{
					name:    "valid - id one of",
					opName:  "listBooks",
					request: `{"where": {"id": {"oneOf": ["123456789"]}}}`,
				},
				{
					name:    "valid - non-query types",
					opName:  "booksByTitleAndGenre",
					request: `{"where": {"title": "Some title", "genre": "Horror", "minPrice": 10}}`,
				},

				// errors
				{
					name:    "text unknown filter",
					opName:  "listBooks",
					request: `{"where": {"title": {"isSimilarTo": "Sci-fi"}}}`,
					errors: map[string]string{
						"where.title": `Additional property isSimilarTo is not allowed`,
					},
				},
				{
					name:    "enum equals not valid enum",
					opName:  "listBooks",
					request: `{"where": {"genre": {"equals": "Sci-fi"}}}`,
					errors: map[string]string{
						"where.genre.equals": `where.genre.equals must be one of the following: "Romance", "Horror"`,
					},
				},
				{
					name:    "enum one of not valid enum",
					opName:  "listBooks",
					request: `{"where": {"genre": {"oneOf": ["Sci-fi"]}}}`,
					errors: map[string]string{
						"where.genre.oneOf.0": `where.genre.oneOf.0 must be one of the following: "Romance", "Horror"`,
					},
				},
				{
					name:    "timestamp invalid format",
					opName:  "listBooks",
					request: `{"where": {"createdAt": {"after": "not-a-date-time"}}}`,
					errors: map[string]string{
						"where.createdAt.after": `Does not match format 'date-time'`,
					},
				},
				{
					name:    "date invalid format",
					opName:  "listBooks",
					request: `{"where": {"releaseDate": {"after": "not-a-date-time"}}}`,
					errors: map[string]string{
						"where.releaseDate.after": `Does not match format 'date'`,
					},
				},
				{
					name:    "using query types for explicit filters",
					opName:  "booksByTitleAndGenre",
					request: `{"where": {"title": {"contains": "Some title"}, "genre": {"equals": "Horror"}}}`,
					errors: map[string]string{
						"where.title": `Invalid type. Expected: string, given: object`,
						"where.genre": `Invalid type. Expected: string, given: object`,
					},
				},
			},
		},
		{
			name:   "authenticate",
			schema: `model Whatever {}`,
			cases: []fixture{
				{
					name:    "valid",
					opName:  "authenticate",
					request: `{"emailPassword": {"email": "foo@bar.com", "password": "pa33w0rd"}}`,
				},
			},
		},
	}

	for _, group := range fixtures {
		for _, f := range group.cases {
			group := group
			f := f
			t.Run(group.name+"/"+f.name, func(t *testing.T) {

				builder := schema.Builder{}
				schema, err := builder.MakeFromString(group.schema)
				require.NoError(t, err)

				var req map[string]any
				err = json.Unmarshal([]byte(f.request), &req)
				require.NoError(t, err)

				op := proto.FindOperation(schema, f.opName)

				result, err := jsonschema.ValidateRequest(context.Background(), schema, op, req)
				require.NoError(t, err)
				require.NotNil(t, result)

				if len(f.errors) == 0 {
					assert.True(t, result.Valid(), "expected request to be valid")
				}

				for _, e := range result.Errors() {
					// this will be the full path to where the error is located
					// for example "(root).someProp.someOtherProp"
					jsonPath := e.Context().String()

					// When we're not at the root, we don't really need that
					// prefix, so trim it off
					if jsonPath != "(root)" {
						jsonPath = strings.TrimPrefix(jsonPath, "(root).")
					}

					expected, ok := f.errors[jsonPath]
					if !ok {
						assert.Fail(t, "unexpected error", "%s - %s", jsonPath, e.Description())
						continue
					}

					assert.Equal(t, expected, e.Description(), "error for path %s did not match expected", jsonPath)
					delete(f.errors, jsonPath)
				}

				// f.errors should now be empty, if not mark test as failed
				for path, description := range f.errors {
					assert.Fail(t, "expected error was not returned", "%s - %s", path, description)
				}
			})
		}
	}

}