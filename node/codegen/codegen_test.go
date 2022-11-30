package codegenerator_test

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/samber/lo"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	codegenerator "github.com/teamkeel/keel/node/codegen"
	"github.com/teamkeel/keel/schema"
)

type TestCase struct {
	Name          string
	Schema        string
	ExpectedFiles []*codegenerator.GeneratedFile
}

const (
	NO_CONTENT            string = "[NO-CONTENT]" // used to denote that an expected file should be empty, and therefore skipped
	DIVIDER               string = "=========================================================="
	LEVENSHTEIN_THRESHOLD int    = 5 // requires a maximum of five edits between actual and expected to be considered similar
)

func TestSdk(t *testing.T) {
	// When comparing actual vs expected src code contents, the expected value is
	// evaluated partially [e.g expect(actual).toInclude(expected)] so you do not need
	// to specify the full code to be generated, and instead match partially on whatever
	// you want to expect to be added for a given test.
	cases := []TestCase{
		{
			Name: "imports",
			Schema: `
				model Foo {}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						import { queryResolverFromEnv, Logger } from '@teamkeel/functions-runtime';
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						import {
							QueryConstraints,
							ChainableQuery,
							Query,
							FunctionError,
							FunctionCreateResponse,
							FunctionGetResponse,
							FunctionDeleteResponse,
							FunctionListResponse,
							FunctionUpdateResponse,
							FunctionAuthenticateResponse
						} from '@teamkeel/functions-runtime';
					`,
				},
			},
		},
		{
			Name: "base-types",
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						const queryLogger = new Logger();
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare type ID = string
						export declare type Timestamp = string
					`,
				},
			},
		},
		{
			Name: "model-generation-simple",
			Schema: `
			model Person {
				fields {
					name Text
					age Number
				}
			}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						export class PersonApi {
							constructor() {
								this.create = async (inputs) => {
									return this.db.create(inputs);
								};
								this.where = (conditions) => {
									return this.db.where(conditions);
								};
								this.delete = (id) => {
									return this.db.delete(id);
								};
								this.findOne = (query) => {
									return this.db.findOne(query);
								};
								this.update = (id, inputs) => {
									return this.db.update(id, inputs);
								};
								this.findMany = (query) => {
									return this.db.where(query).all();
								};
								this.db = new Query({
									tableName: 'person',
									queryResolver: queryResolverFromEnv(process.env),
									logger: queryLogger
								});
							}
						}
						export class IdentityApi {
							constructor() {
								this.create = async (inputs) => {
									return this.db.create(inputs);
								};
								this.where = (conditions) => {
									return this.db.where(conditions);
								};
								this.delete = (id) => {
									return this.db.delete(id);
								};
								this.findOne = (query) => {
									return this.db.findOne(query);
								};
								this.update = (id, inputs) => {
									return this.db.update(id, inputs);
								};
								this.findMany = (query) => {
									return this.db.where(query).all();
								};
								this.db = new Query({
									tableName: 'identity',
									queryResolver: queryResolverFromEnv(process.env),
									logger: queryLogger
								});
							}
						}
						export const api = {
							models: {
								person: new PersonApi(),
								identity: new IdentityApi(),
							}
						}`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export interface Person {
						  name: string
						  age: number
						  id: ID
						  createdAt: Date
						  updatedAt: Date
						}
						export declare type PersonQuery = {
						  name?: QueryConstraints.StringConstraint
						  age?: QueryConstraints.NumberConstraint
						  id?: QueryConstraints.IdConstraint
						  createdAt?: QueryConstraints.DateConstraint
						  updatedAt?: QueryConstraints.DateConstraint
						}
						export declare type PersonUniqueFields = {
							id?: QueryConstraints.IdConstraint
						}
					`,
				},
			},
		},
		{
			Name: "model-generation-custom-function",
			Schema: `model Person {
				fields {
					name Text
					age Number
				}

				functions {
					create createPerson() with(name, age)
					update updatePerson(id) with(name, age)
					delete deletePerson(id)
					list listPerson()
					get getPerson(id)
				}
			}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
					export const CreatePerson = (callback) => (inputs, api) => {
						return callback(inputs, api);
					};
					export const UpdatePerson = (callback) => (inputs, api) => {
						return callback(inputs, api);
					};
					export const DeletePerson = (callback) => (inputs, api) => {
						return callback(inputs, api);
					};
					export const ListPerson = (callback) => (inputs, api) => {
						return callback(inputs, api);
					};
					export const GetPerson = (callback) => (inputs, api) => {
						return callback(inputs, api);
					};`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare type CreatePersonReturnType = Promise<FunctionCreateResponse<Person>>;
						export declare type CreatePersonCallbackFunction = (inputs: CreatePersonInput, api: KeelApi) => CreatePersonReturnType;
						export declare const CreatePerson: (callback: CreatePersonCallbackFunction) => (inputs: CreatePersonInput, api: KeelApi) => CreatePersonReturnType
						export interface CreatePersonInput {
							name: string
							age: number
						}
						export declare type UpdatePersonReturnType = Promise<FunctionUpdateResponse<Person>>;
						export declare type UpdatePersonCallbackFunction = (inputs: UpdatePersonInput, api: KeelApi) => UpdatePersonReturnType;
						export declare const UpdatePerson: (callback: UpdatePersonCallbackFunction) => (inputs: UpdatePersonInput, api: KeelApi) => UpdatePersonReturnType
						export interface UpdatePersonInput {
							where: {
								id: ID
							}
							values: {
								name: string
								age: number
							}
						}
						export declare type DeletePersonReturnType = Promise<FunctionDeleteResponse<Person>>;
						export declare type DeletePersonCallbackFunction = (inputs: DeletePersonInput, api: KeelApi) => DeletePersonReturnType;
						export declare const DeletePerson: (callback: DeletePersonCallbackFunction) => (inputs: DeletePersonInput, api: KeelApi) => DeletePersonReturnType
						export interface DeletePersonInput {
							id: ID
						}
						export declare type ListPersonReturnType = Promise<FunctionListResponse<Person>>;
						export declare type ListPersonCallbackFunction = (inputs: ListPersonInput, api: KeelApi) => ListPersonReturnType;
						export declare const ListPerson: (callback: ListPersonCallbackFunction) => (inputs: ListPersonInput, api: KeelApi) => ListPersonReturnType
						export interface ListPersonInput {
							where: {
							}
						}
						export declare type GetPersonReturnType = Promise<FunctionGetResponse<Person>>;
						export declare type GetPersonCallbackFunction = (inputs: GetPersonInput, api: KeelApi) => GetPersonReturnType;
						export declare const GetPerson: (callback: GetPersonCallbackFunction) => (inputs: GetPersonInput, api: KeelApi) => GetPersonReturnType
					`,
				},
			},
		},
		{
			Name: "enum-generation",
			Schema: `
				enum TheBeatles {
					John
					Paul
					Ringo
					George
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path:     "index.js",
					Contents: NO_CONTENT,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare enum TheBeatles {
							John = "John",
							Paul = "Paul",
							Ringo = "Ringo",
							George = "George",
						}
					`,
				},
			},
		},
		{
			Name: "query-types",
			Schema: `
				model Person {
					fields {
						title Text
						subTitle Text?
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path:     "index.js",
					Contents: NO_CONTENT,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare type PersonQuery = {
							title?: QueryConstraints.StringConstraint
							subTitle?: QueryConstraints.StringConstraint
							id?: QueryConstraints.IdConstraint
							createdAt?: QueryConstraints.DateConstraint
							updatedAt?: QueryConstraints.DateConstraint
						}
						export declare type PersonUniqueFields = {
							id?: QueryConstraints.IdConstraint
						}
					`,
				},
			},
		},
		{
			Name: "input-types",
			Schema: `
				model Person {
					fields {
						title Text
						subTitle Text?
						age Number?
					}

					operations {
						create createPerson() with(title, subTitle?)
						delete deletePerson(id)
						list listPeople(title, age)
						get getPerson(id)
						update updatePerson(id) with(title, subTitle?)
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path:     "index.js",
					Contents: NO_CONTENT,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export interface CreatePersonInput {
							title: string
							subTitle?: string
						}
						export interface DeletePersonInput {
							id: ID
						}
						export interface ListPeopleInput {
							where: {
								title: StringConstraint
								age: NumberConstraint
							}
						}
						export interface GetPersonInput {
							id: ID
						}
						export interface UpdatePersonInput {
							where: {
								id: ID
							}
							values: {
								title: string
								subTitle?: string
							}
						}
						export interface AuthenticateInput {
							createIfNotExists?: boolean
							emailPassword: unknown
						}
					`,
				},
			},
		},
		{
			Name: "model-database-apis",
			Schema: `
				model Person {
					fields {
						title Text
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						export class PersonApi {
							constructor() {
								this.create = async (inputs) => {
									return this.db.create(inputs);
								};
								this.where = (conditions) => {
									return this.db.where(conditions);
								};
								this.delete = (id) => {
									return this.db.delete(id);
								};
								this.findOne = (query) => {
									return this.db.findOne(query);
								};
								this.update = (id, inputs) => {
									return this.db.update(id, inputs);
								};
								this.findMany = (query) => {
									return this.db.where(query).all();
								};
								this.db = new Query({
									tableName: 'person',
									queryResolver: queryResolverFromEnv(process.env),
									logger: queryLogger
								});
							}
						}
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare class PersonApi {
							private readonly db : Query<Person>;
							constructor();
							create: (inputs: Partial<Omit<Person, "id" | "createdAt" | "updatedAt">>) => Promise<FunctionCreateResponse<Person>>;
							where: (conditions: PersonQuery) => ChainableQuery<Person>;
							delete: (id: string) => Promise<FunctionDeleteResponse<Person>>;
							findOne: (query: PersonUniqueFields) => Promise<FunctionGetResponse<Person>>;
							update: (inputs: Partial<Omit<Person, "id" | "createdAt" | "updatedAt">>) => Promise<FunctionUpdateResponse<Person>>;
							findMany: (query: PersonQuery) => Promise<FunctionListResponse<Person>>;
						}
					`,
				},
			},
		},
		{
			Name: "top-level-api",
			Schema: `
				model Person {
					fields {
						title Text
					}
				}
				model Author {
					fields {
						name Text
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						export const api = {
							models: {
								person: new PersonApi(),
								author: new AuthorApi(),
								identity: new IdentityApi(),
							}
						}
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare type KeelApi = {
							models: {
								person: PersonApi,
								author: AuthorApi,
								identity: IdentityApi,
							}
						}
						export declare const api : KeelApi
					`,
				},
			},
		},
	}

	runCases(t, cases, func(cg *codegenerator.Generator) ([]*codegenerator.GeneratedFile, error) {
		return cg.GenerateSDK()
	})
}

func TestTesting(t *testing.T) {
	cases := []TestCase{
		{
			Name: "imports",
			Schema: `
				model Foo {}
			`,
			ExpectedFiles: []*codegenerator.SourceCode{
				{
					Path: "index.js",
					Contents: `
						import {
							Logger,
							queryResolverFromEnv,
						} from '@teamkeel/functions-runtime';
						import { ActionExecutor } from '@teamkeel/functions-testing';
					`,
				},
				{
					Path:     "index.d.ts",
					Contents: NO_CONTENT,
				},
			},
		},
		{
			Name: "plumbing",
			Schema: `
				model Foo {}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						const qr = queryResolverFromEnv(process.env);
						const parentPort = parseInt(process.env.PARENT_PORT, 10);
						const host = process.env.HOST || 'localhost';
						const queryLogger = new Logger();
						const actionExecutor = new ActionExecutor({ parentPort, host });
					`,
				},
				{
					Path:     "index.d.ts",
					Contents: NO_CONTENT,
				},
			},
		},
		{
			Name: "action-generation-operations",
			Schema: `
				model Post {
					fields {
						title Text
					}

					operations {
						create createPost() with(title)
						update updatePost(id) with(title)
						delete deletePost(id)
						list listPosts()
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
					class ActionsWithIdentity {
						constructor(identity) {
							if (identity === undefined) {
								throw new Error('valid identity was not provided to withIdentity.');
							}
							this.identity = identity;
							this.createPost = async (payload) => await actionExecutor.execute({
								actionName: "createPost",
								payload,
								identity: this.identity,
							});
							this.updatePost = async (payload) => await actionExecutor.execute({
								actionName: "updatePost",
								payload,
								identity: this.identity,
							});
							this.deletePost = async (payload) => await actionExecutor.execute({
								actionName: "deletePost",
								payload,
								identity: this.identity,
							});
							this.listPosts = async (payload) => await actionExecutor.execute({
								actionName: "listPosts",
								payload,
								identity: this.identity,
							});
							this.authenticate = async (payload) => await actionExecutor.execute({
								actionName: "authenticate",
								payload,
								identity: this.identity,
							});
						}
					}
					export class Actions {
						constructor() {
							this.withIdentity = (identity) => {
								return new ActionsWithIdentity(identity);
							};
							this.createPost = async (payload) => await actionExecutor.execute({
								actionName: "createPost",
								payload,
							});
							this.updatePost = async (payload) => await actionExecutor.execute({
								actionName: "updatePost",
								payload,
							});
							this.deletePost = async (payload) => await actionExecutor.execute({
								actionName: "deletePost",
								payload,
							});
							this.listPosts = async (payload) => await actionExecutor.execute({
								actionName: "listPosts",
								payload,
							});
							this.authenticate = async (payload) => await actionExecutor.execute({
								actionName: "authenticate",
								payload,
							});
						}
					}
					export const actions = new Actions();`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						import * as SDK from '@teamkeel/sdk';
						import * as ReturnTypes from '@teamkeel/functions-runtime/returnTypes';
						import { ActionExecutor } from '@teamkeel/functions-testing';

						declare class ActionsWithIdentity {
							private identity : SDK.Identity;

							constructor(identity: SDK.Identity | undefined)
							createPost: (payload) => ReturnTypes.FunctionCreateResponse<SDK.Post>
							updatePost: (payload) => ReturnTypes.FunctionUpdateResponse<SDK.Post>
							deletePost: (payload) => ReturnTypes.FunctionDeleteResponse<SDK.Post>
							listPosts: (payload) => ReturnTypes.FunctionListResponse<SDK.Post>
							authenticate: (payload) => ReturnTypes.FunctionAuthenticateResponse<SDK.Identity>
						}
						export declare class Actions {
							withIdentity: (identity: SDK.Identity | undefined) => ActionsWithIdentity
							createPost: (payload) => ReturnTypes.FunctionCreateResponse<SDK.Post>
							updatePost: (payload) => ReturnTypes.FunctionUpdateResponse<SDK.Post>
							deletePost: (payload) => ReturnTypes.FunctionDeleteResponse<SDK.Post>
							listPosts: (payload) => ReturnTypes.FunctionListResponse<SDK.Post>
							authenticate: (payload) => ReturnTypes.FunctionAuthenticateResponse<SDK.Identity>
						}
						export declare const actions : Actions;
					`,
				},
			},
		},
		{
			Name:   "testing-model-api-identity",
			Schema: "",
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						class IdentityApi {
							constructor() {
								this.create = async (inputs) => {
									const q = await this.query()
									return q.create(inputs);
								}
								this.where = (conditions) => {
									return new ChainableQuery({
										tableName: 'identity',
										queryResolver: qr,
										conditions: [conditions],
										logger: queryLogger,
									})
								}
								this.delete = async (id) => {
									const q = await this.query()
									return q.delete(id);
								}
								this.findOne = async (query) => {
									const q = await this.query()
									return q.findOne(query as any);
								}
								this.update = async (id, inputs) => {
									const q = await this.query()
									return q.update(id, inputs as any);
								}
								this.findMany = async (query) => {
									const q = await this.query()
									return q.where(query as any).all();
								}
								this.query = async () => {
									return new Query({
										tableName: 'identity',
										queryResolver: qr,
										logger: queryLogger,
									})
								}
							}
						}
						export const Identity = new IdentityApi();
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare class IdentityApi {
							private query : Query<SDK.Identity>;
							constructor();
							create: (inputs: Partial<Omit<SDK.Identity, "id" | "createdAt" | "updatedAt">>) => Promise<ReturnTypes.FunctionCreateResponse<SDK.Identity>>
							where: (conditions: SDK.IdentityQuery) => ChainableQuery<SDK.Identity>
							delete: (id: string) => Promise<ReturnTypes.FunctionDeleteResponse<SDK.Identity>>
							findOne: (query: SDK.Identity) => Promise<ReturnTypes.FunctionGetResponse<SDK.Identity>>
							update: (inputs: Partial<Omit<SDK.Identity, "id" | "createdAt" | "updatedAt">>) => Promise<ReturnTypes.FunctionUpdateResponse<SDK.Identity>>
							findMany: (query: SDK.IdentityQuery) => Promise<ReturnTypes.FunctionListResponse<SDK.Identity>>
						}
						export declare const Identity : IdentityApi;
					`,
				},
			},
		},
		{
			Name: "testing-model-api-generation",
			Schema: `
				model Post {
					fields {
						title Text?
						rating Number
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						class PostApi {
							constructor() {
								this.create = async (inputs) => {
									const q = await this.query()
									return q.create(inputs);
								}
						
								this.where = (conditions) => {
									return new ChainableQuery({
										tableName: 'post',
										queryResolver: qr,
										conditions: [conditions],
										logger: queryLogger,
									})
								}
						
								this.delete = async (id) => {
									const q = await this.query()
									return q.delete(id);
								}
						
								this.findOne = async (query) => {
									const q = await this.query()
									return q.findOne(query as any);
								}
						
								this.update = async (id, inputs) => {
									const q = await this.query()
									return q.update(id, inputs as any);
								}
						
								this.findMany = async (query) => {
									const q = await this.query()
									return q.where(query as any).all();
								}
						
								this.query = async () => {
									return new Query({
										tableName: 'post',
										queryResolver: qr,
										logger: queryLogger,
									})
								}
							}
						}

						export const Post = new PostApi();
					`,
				},
				{
					Path: "index.d.ts",
					Contents: `
						export declare class PostApi {
							private query : Query<SDK.Post>;
							constructor();
							create: (inputs: Partial<Omit<SDK.Post, "id" | "createdAt" | "updatedAt">>) => Promise<ReturnTypes.FunctionCreateResponse<SDK.Post>>
							where: (conditions: SDK.PostQuery) => ChainableQuery<SDK.Post>
							delete: (id: string) => Promise<ReturnTypes.FunctionDeleteResponse<SDK.Post>>
							findOne: (query: SDK.Post) => Promise<ReturnTypes.FunctionGetResponse<SDK.Post>>
							update: (inputs: Partial<Omit<SDK.Post, "id" | "createdAt" | "updatedAt">>) => Promise<ReturnTypes.FunctionUpdateResponse<SDK.Post>>
							findMany: (query: SDK.PostQuery) => Promise<ReturnTypes.FunctionListResponse<SDK.Post>>
						}
						export declare const Post : PostApi;
					`,
				},
			},
		},
	}

	runCases(t, cases, func(cg *codegenerator.Generator) ([]*codegenerator.GeneratedFile, error) {
		return cg.GenerateTesting()
	})
}

func TestDevelopmentHandler(t *testing.T) {
	cases := []TestCase{
		{
			Name: "basic",
			Schema: `
				model Post {
					fields {
						title Text
					}

					functions {
						create createPost() with(title)
						update updatePost(id) with(title)
					}
				}
			`,
			ExpectedFiles: []*codegenerator.GeneratedFile{
				{
					Path: "index.js",
					Contents: `
						import { handleCustomFunction } from '@teamkeel/functions-runtime';
						import { createServer } from "http";
						import url from "url";
						// Imports to custom functions
						import createPost from '../functions/createPost';
						import updatePost from '../functions/updatePost';
						const startRuntimeServer = ({ functions, api }) => {
							const listener = async (req, res) => {
								if (req.method === "POST") {
									const buffers = [];
									for await (const chunk of req) {
										buffers.push(chunk);
									}
									const data = Buffer.concat(buffers).toString();
									// jsonrpc-2.0 compliant payload
									const json = JSON.parse(data);
									const rpcResponse = await handleCustomFunction(json, config);
									
									res.statusCode = 200;
									res.setHeader('Content-Type', 'application/json');
									res.write(
										JSON.stringify(rpcResponse)
									);
								}
								res.end();
							};
							const server = createServer(listener);
							const port = (process.env.PORT && parseInt(process.env.PORT, 10)) || 3001;
							server.listen(port);
						};
						startRuntimeServer({
							functions: {
								createPost,
								updatePost,
							},
							api: {
								models: {
									post: new PostApi(),
									identity: new IdentityApi(),
								}
							}
						});
					`,
				},
			},
		},
	}

	runCases(t, cases, func(cg *codegenerator.Generator) ([]*codegenerator.GeneratedFile, error) {
		return cg.GenerateDevelopmentHandler()
	})
}

func runCases(t *testing.T, cases []TestCase, codegenFn func(cg *codegenerator.Generator) ([]*codegenerator.GeneratedFile, error)) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			builder := schema.Builder{}

			sch, err := builder.MakeFromString(tc.Schema)

			require.NoError(t, err)

			tmpDir, err := os.MkdirTemp("", tc.Name)

			require.NoError(t, err)

			cg := codegenerator.NewGenerator(sch, tmpDir)

			generatedFiles, err := codegenFn(cg)

			require.NoError(t, err)

			compareFiles(t, tc, generatedFiles)

			o, err := typecheck(t, generatedFiles)

			if err != nil {
				fmt.Print(string(o))
				t.Fail()
			}
		})
	}
}

// Removes inconsequential differences between actual and expected strings:
// - Normalises tab / 2 space differences
// - Normalises indentation between actual vs expected
func normaliseString(str string) string {
	lines := strings.Split(str, "\n")

	lines = lo.Filter(lines, func(line string, idx int) bool {
		if line == "" {
			return idx > 0 && idx < len(lines)-1
		}

		return true
	})

	if len(lines) == 0 {
		return ""
	}

	firstLine := lines[0]

	firstLineChars := strings.Split(firstLine, "")

	indentDepth := 0

	for _, char := range firstLineChars {
		if char != " " && char != "\t" {
			break
		}

		indentDepth += 1
	}

	newLines := []string{}

	for _, line := range lines {
		lineLength := len(line)

		if lineLength <= indentDepth {
			// blank line in body of lines
			continue
		} else {
			newLines = append(newLines, line[indentDepth:])
		}
	}

	tabReplacer := strings.NewReplacer(
		"\t", "  ",
	)

	newStr := strings.Join(newLines, "\n")

	newStr = tabReplacer.Replace(newStr)

	return newStr
}

func compareFiles(t *testing.T, tc TestCase, generatedFiles []*codegenerator.GeneratedFile) {
actual:
	for _, actualFile := range generatedFiles {
		for _, expectedFile := range tc.ExpectedFiles {
			if expectedFile.Contents == NO_CONTENT {
				continue actual
			}

			if actualFile.Path == expectedFile.Path {
				assert.Contains(t, normaliseString(actualFile.Contents), normaliseString(expectedFile.Contents))

				actual := normaliseString(actualFile.Contents)
				expected := normaliseString(expectedFile.Contents)

				if expected == "" {
					// was not asserted

					continue actual
				}

				if !strings.Contains(actual, expected) {
					// Attempting to perform a unified diff between a partial substring and a much larger actual string creates
					// some problems with the diff output - comprehension of what has changed within the substring is particularly tricky with the unified
					// diff output
					// Therefore, the call to matchActual will try to match the expected string against the actual, returning only the relevant
					// portion of the actual string for diff display.
					diff := diffmatchpatch.New()
					match := matchActual(actual, expected)

					// expected goes first in the diff order below so that things missing in actual
					// are shown as deletions rather than additions
					diffs := diff.DiffMain(expected, match, false)

					fmt.Printf("Test case '%s' failed.\n%s\nContextual Diff (%s):\n%s\n%s\n\nActual (%s):\n%s\n%s\n\nExpected (%s):\n%s\n%s\n",
						t.Name(),
						DIVIDER,
						expectedFile.Path,
						DIVIDER,
						diff.DiffPrettyText(diffs),
						actualFile.Path,
						DIVIDER,
						actual,
						expectedFile.Path,
						DIVIDER,
						expected,
					)
					t.Fail()
				}

				continue actual
			}
		}

		assert.Fail(t, fmt.Sprintf("no matching expected file for actual file %s", actualFile.Path))
	}

	// we need to do the inverse of the above, so that we don't miss out on the scenario
	// where there are files in the actual that are not in the expected array
expected:
	for _, expectedFile := range tc.ExpectedFiles {
		if expectedFile.Contents == NO_CONTENT {
			continue
		}

		for _, actualFile := range generatedFiles {
			if actualFile.Path == expectedFile.Path {
				continue expected
			}
		}

		assert.Fail(t, fmt.Sprintf("no matching actual file for expected file %s", expectedFile.Path))
	}
}

//go:embed tsconfig.json
var sampleTsConfig string

// After we have asserted that the actual and expected values match, we want to typecheck the outputted d.ts
// files using tsc to make sure that it is valid typescript!
func typecheck(t *testing.T, generatedFiles []*codegenerator.GeneratedFile) (output string, err error) {
	tmpDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)

	f, err := os.Create(filepath.Join(tmpDir, "tsconfig.json"))

	if err != nil {
		return "", err
	}

	f.WriteString(sampleTsConfig)

	hasTypeScriptInputs := lo.SomeBy(generatedFiles, func(f *codegenerator.SourceCode) bool {
		ext := filepath.Ext(f.Path)

		switch ext {
		case ".ts", ".d.ts":
			return true
		default:
			return false
		}
	})

	// dont typecheck if there are no typescript files to typecheck
	if !hasTypeScriptInputs {
		return "", nil
	}

	for _, file := range generatedFiles {
		f, err := os.Create(filepath.Join(tmpDir, file.Path))

		assert.NoError(t, err)

		_, err = f.WriteString(file.Contents)

		assert.NoError(t, err)
	}

	defer f.Close()

	// todo: remove skipLibCheck flag once we have the @teamkeel/functions-runtime package in place.
	cmd := exec.Command("npx", "-p", "typescript", "tsc", "--noEmit", "--pretty", "--skipLibCheck", "--incremental", "--project", filepath.Base(f.Name()))
	cmd.Dir = tmpDir

	b, e := cmd.CombinedOutput()

	str := string(b)

	// tsc outputs ansi escape codes in its typechecking output
	// which don't render correctly in the vscode test output channel
	// (most likely not handled in the vscode-go plugin)
	str = stripansi.Strip(str)

	if e != nil {
		err = e
	}

	return str, err
}

// matchActual attempts to match the smaller expected string against a subsection of the larger
// actual result - line by line comparison is performed to find a partial match in the actual output
// The return value is a subsection of the actual input string
func matchActual(actual, expected string) string {
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	match := false
	matchStart := 0
	matchEnd := 0

	differ := diffmatchpatch.New()

	// Loop over each expected line, and try to find a match in the actual lines. If there is a match between actual and expected lines in any iteration,
	// then break execution and take a slice of the actual lines based on how far we iterated through both loops
expected:
	for eIdx, el := range expectedLines {
		for aIdx, al := range actualLines {
			diffs := differ.DiffMain(al, el, false)
			levenshteinDistance := differ.DiffLevenshtein(diffs)

			// the current levenshtein distance threshold may actually produce some
			// false positives for lines that are similar enough but aren't actually
			// the same code (with differences inside of it), so we can tweak the threshold
			// when we run into these problems
			if al == el || levenshteinDistance < LEVENSHTEIN_THRESHOLD {
				match = true
				matchStart = aIdx - eIdx

				// the end location is the length of expected lines minus the number of
				// expected lines we iterated through first to find a match, plus the number
				// of lines we iterated through actual to find the match
				matchEnd = len(expectedLines) - eIdx + aIdx

				break expected
			}
		}
	}

	if match {
		if matchStart < 0 {
			matchStart = 0
		}
		matchingLines := strings.Join(actualLines[matchStart:min(matchEnd, len(actualLines))], "\n")
		return matchingLines
	}

	return strings.Join(actualLines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
