package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/casing"
	"github.com/teamkeel/keel/codegen"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema/parser"
	"golang.org/x/exp/slices"
)

type generateOptions struct {
	developmentServer  bool
	prismaBinaryTarget string
}

// WithDevelopmentServer enables or disables the generation of the development
// server entry point. By default this is disabled.
func WithDevelopmentServer(b bool) func(o *generateOptions) {
	return func(o *generateOptions) {
		o.developmentServer = b
	}
}

// WithPrismaBinaryTarget adds target to the binaryTargets list in the Prisma
// client generation config
func WithPrismaBinaryTarget(target string) func(o *generateOptions) {
	return func(o *generateOptions) {
		o.prismaBinaryTarget = target
	}
}

// Generate generates and returns a list of objects that represent files to be written
// to a project. Calling .Write() on the result will cause those files be written to disk.
// This function should not interact with the file system so it can be used in a backend
// context.
func Generate(ctx context.Context, schema *proto.Schema, opts ...func(o *generateOptions)) (codegen.GeneratedFiles, error) {
	options := &generateOptions{}
	for _, o := range opts {
		o(options)
	}

	files := generateSdkPackage(schema)
	files = append(files, generateTestingPackage(schema)...)
	files = append(files, generateTestingSetup()...)
	files = append(files, generatePrismaSchema(schema, options.prismaBinaryTarget)...)

	if options.developmentServer {
		files = append(files, generateDevelopmentServer(schema)...)
	}

	return files, nil
}

var toPrismaTypes = map[proto.Type]string{
	proto.Type_TYPE_ID:        "String",
	proto.Type_TYPE_STRING:    "String",
	proto.Type_TYPE_INT:       "Int",
	proto.Type_TYPE_BOOL:      "Boolean",
	proto.Type_TYPE_TIMESTAMP: "DateTime",
	proto.Type_TYPE_DATE:      "DateTime",
	proto.Type_TYPE_DATETIME:  "DateTime",
	proto.Type_TYPE_PASSWORD:  "String",
	proto.Type_TYPE_SECRET:    "String",
}

// generatePrismaSchema takes our proto.Schema and generates a valid Prisma schema from it.
func generatePrismaSchema(schema *proto.Schema, binaryTarget string) codegen.GeneratedFiles {
	s := &codegen.Writer{}

	binaryTargets := []string{`"native"`}
	if binaryTarget != "" {
		binaryTargets = append(binaryTargets, fmt.Sprintf(`"%s"`, binaryTarget))
	}

	// Note on the binaryTargets - rhel-openssl-1.0.x is needed for AWS Lambda x86
	// If we were to move to ARM based Lambda we'd need to update this
	s.Writeln(fmt.Sprintf(`
datasource db {
    provider = "postgresql"
    url      = env("KEEL_DB_CONN") 
} 

generator client {
    provider = "prisma-client-js"
    previewFeatures = ["jsonProtocol", "tracing"]
    binaryTargets = [%s]
}
`, strings.Join(binaryTargets, ", ")))

	for _, m := range schema.Models {
		s.Writef("model %s {", m.Name)
		s.Writeln("")
		s.Indent()

		for _, f := range m.Fields {
			var prismaType string
			var mapping string
			switch f.Type.Type {
			case proto.Type_TYPE_ENUM:
				prismaType = f.Type.EnumName.Value
			case proto.Type_TYPE_MODEL:
				prismaType = f.Type.ModelName.Value
			default:
				var ok bool
				prismaType, ok = toPrismaTypes[f.Type.Type]
				if !ok {
					// https://www.prisma.io/docs/concepts/components/prisma-schema/data-model#unsupported-types
					prismaType = "Unsupported"
				}
				mapping = fmt.Sprintf(`@map("%s")`, casing.ToSnake(f.Name))
			}
			s.Write(f.Name)
			s.Write(" ")
			s.Write(prismaType)
			if f.Optional {
				s.Write("?")
			}
			if f.Type.Repeated {
				s.Write("[]")
			}
			s.Write(" ")
			s.Write(mapping)

			if f.PrimaryKey {
				s.Writef(` @id`)
			}
			if f.Type.Type != proto.Type_TYPE_MODEL && f.Unique {
				s.Write(" @unique")
			}

			if f.Type.Type == proto.Type_TYPE_MODEL {
				relatedModel, _, relName := getPrismaRelationInfo(schema, m, f)

				s.Write(" @relation(")
				s.Writef(`"%s"`, relName)

				if f.ForeignKeyFieldName != nil {
					s.Write(", fields: [")
					s.Write(f.ForeignKeyFieldName.Value)
					s.Write("], references: [")
					s.Write(proto.PrimaryKeyFieldName(relatedModel))
					s.Write("]")
				}

				s.Write(")")

			}

			s.Writeln("")
		}

		// Prisma enforces relationship bi-directionality, whereas our schema language doesn't enforce specifying both sides of a relationship in some cases
		// so therefore, after adding all of the known fields for a model into the corresponding Prisma model, we now need to go and add find any relationships between this model and another model where a link field isn't specified on *this* side of the relationship but one
		// is on the other side.
		for _, otherModel := range schema.Models {
			for _, otherField := range otherModel.Fields {
				if otherField.Type.Type == proto.Type_TYPE_MODEL && otherField.Type.ModelName.Value == m.Name && otherField.InverseFieldName == nil {
					// format we're generating: {generatedFieldName} {otherModel.Name} @relation("{generatedRelationshipName}")
					_, fieldName, relName := getPrismaRelationInfo(schema, otherModel, otherField)
					fieldType := otherModel.Name

					if !otherField.Unique {
						// if the field isn't unique, then it means this is the has-many side
						fieldType += "[]"
					} else {
						// in prisma, one to one relationships must set the side of the relationship
						// that doesn't have the foreign key to optional
						// docs: https://www.prisma.io/docs/concepts/components/prisma-schema/relations/one-to-one-relations#required-and-optional-1-1-relation-fields
						fieldType += "?"
					}

					s.Writeln("")
					s.Writef("%s %s @relation(\"%s\")", fieldName, fieldType, relName)
				}
			}
		}
		s.Writeln("")
		s.Writef(`@@map("%s")`, casing.ToSnake(m.Name))
		s.Writeln("")
		s.Dedent()
		s.Writeln("}")
	}

	for _, e := range schema.Enums {
		s.Writef("enum %s {", e.Name)
		s.Writeln("")
		s.Indent()
		for _, v := range e.Values {
			s.Writeln(v.Name)
		}
		s.Dedent()
		s.Writeln("}")
	}

	return codegen.GeneratedFiles{
		{
			Path:     ".build/schema.prisma",
			Contents: s.String(),
		},
	}
}

func getPrismaRelationInfo(schema *proto.Schema, m *proto.Model, f *proto.Field) (*proto.Model, string, string) {
	nameParts := []string{m.Name, f.Name}
	relatedModel := proto.FindModel(schema.Models, f.Type.ModelName.Value)
	nameParts = append(nameParts, relatedModel.Name)
	fieldName := ""

	if f.InverseFieldName != nil {
		fieldName = f.InverseFieldName.Value
	} else {
		fieldName = fmt.Sprintf("%s_By_%s", casing.ToLowerCamel(m.Name), casing.ToCamel(f.Name))

	}
	nameParts = append(nameParts, fieldName)
	slices.Sort(nameParts)
	return relatedModel, fieldName, strings.Join(nameParts, "")
}

func generateSdkPackage(schema *proto.Schema) codegen.GeneratedFiles {
	sdk := &codegen.Writer{}
	sdk.Writeln(`const { sql } = require("kysely")`)
	sdk.Writeln(`const runtime = require("@teamkeel/functions-runtime")`)
	sdk.Writeln("")

	sdkTypes := &codegen.Writer{}
	sdkTypes.Writeln(`import { Kysely, Generated } from "kysely"`)
	sdkTypes.Writeln(`import * as runtime from "@teamkeel/functions-runtime"`)
	sdkTypes.Writeln(`import { Headers } from 'node-fetch'`)
	sdkTypes.Writeln("")

	writePermissions(sdk, schema)

	writeMessages(sdkTypes, schema, false)

	for _, enum := range schema.Enums {
		writeEnum(sdkTypes, enum)
		writeEnumWhereCondition(sdkTypes, enum)
		writeEnumObject(sdk, enum)
	}

	for _, model := range schema.Models {
		writeTableInterface(sdkTypes, model)
		writeModelInterface(sdkTypes, model)
		writeCreateValuesInterface(sdkTypes, model)
		writeWhereConditionsInterface(sdkTypes, model)
		writeUniqueConditionsInterface(sdkTypes, model)
		writeModelAPIDeclaration(sdkTypes, model)
		writeModelQueryBuilderDeclaration(sdkTypes, model)
		writeModelDefaultValuesFunction(sdk, model)

		for _, op := range model.Operations {
			// We only care about custom functions for the SDK
			if op.Implementation != proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM {
				continue
			}

			writeCustomFunctionWrapperType(sdkTypes, model, op)

			sdk.Writef("module.exports.%s = (fn) => fn;", casing.ToCamel(op.Name))
			sdk.Writeln("")
		}
	}

	writeTableConfig(sdk, schema.Models)

	writeAPIFactory(sdk, schema)

	writeDatabaseInterface(sdkTypes, schema)
	writeAPIDeclarations(sdkTypes, schema)

	sdk.Writeln("module.exports.getDatabase = runtime.getDatabase;")

	return []*codegen.GeneratedFile{
		{
			Path:     "node_modules/@teamkeel/sdk/index.js",
			Contents: sdk.String(),
		},
		{
			Path:     "node_modules/@teamkeel/sdk/index.d.ts",
			Contents: sdkTypes.String(),
		},
		{
			Path:     "node_modules/@teamkeel/sdk/package.json",
			Contents: `{"name": "@teamkeel/sdk"}`,
		},
	}
}

func writeTableInterface(w *codegen.Writer, model *proto.Model) {
	w.Writef("export interface %sTable {\n", model.Name)
	w.Indent()
	for _, field := range model.Fields {
		if field.Type.Type == proto.Type_TYPE_MODEL {
			continue
		}
		w.Write(casing.ToSnake(field.Name))
		w.Write(": ")
		t := toTypeScriptType(field.Type, false)
		if field.DefaultValue != nil {
			t = fmt.Sprintf("Generated<%s>", t)
		}
		w.Write(t)
		if field.Optional {
			w.Write(" | null")
		}
		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")
}

func writeModelInterface(w *codegen.Writer, model *proto.Model) {
	w.Writef("export interface %s {\n", model.Name)
	w.Indent()
	for _, field := range model.Fields {
		if field.Type.Type == proto.Type_TYPE_MODEL {
			continue
		}
		w.Write(field.Name)
		w.Write(": ")
		t := toTypeScriptType(field.Type, false)
		w.Write(t)
		if field.Optional {
			w.Write(" | null")
		}
		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")
}

func writeCreateValuesInterface(w *codegen.Writer, model *proto.Model) {
	w.Writef("export interface %sCreateValues {\n", model.Name)
	w.Indent()
	for _, field := range model.Fields {
		// For now you can't create related models when creating a record
		if field.Type.Type == proto.Type_TYPE_MODEL {
			continue
		}
		w.Write(field.Name)
		if field.Optional || field.DefaultValue != nil {
			w.Write("?")
		}
		w.Write(": ")
		t := toTypeScriptType(field.Type, false)
		w.Write(t)
		if field.Optional {
			w.Write(" | null")
		}
		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")
}

func writeWhereConditionsInterface(w *codegen.Writer, model *proto.Model) {
	w.Writef("export interface %sWhereConditions {\n", model.Name)
	w.Indent()
	for _, field := range model.Fields {
		w.Write(field.Name)
		w.Write("?")
		w.Write(": ")
		if field.Type.Type == proto.Type_TYPE_MODEL {
			// Embed related models where conditions
			w.Writef("%sWhereConditions | null;", field.Type.ModelName.Value)
		} else {
			w.Write(toTypeScriptType(field.Type, false))
			w.Write(" | ")
			w.Write(toWhereConditionType(field))
			w.Write(" | null;")
		}

		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")
}

func writeMessages(w *codegen.Writer, schema *proto.Schema, isTestingPackage bool) {
	for _, msg := range schema.Messages {
		if msg.Name == parser.MessageFieldTypeAny {
			continue
		}
		writeMessage(w, schema, msg, isTestingPackage)
	}
}

func writeMessage(w *codegen.Writer, schema *proto.Schema, message *proto.Message, isTestingPackage bool) {
	w.Writef("export interface %s {\n", message.Name)
	w.Indent()

	for _, field := range message.Fields {
		w.Write(field.Name)

		if field.Optional {
			w.Write("?")
		}

		w.Write(": ")

		w.Write(toTypeScriptType(field.Type, isTestingPackage))

		if field.Type.Repeated {
			w.Write("[]")
		}

		if field.Nullable {
			w.Write(" | null")
		}

		w.Writeln(";")
	}

	w.Dedent()

	w.Writeln("}")
}

func writeUniqueConditionsInterface(w *codegen.Writer, model *proto.Model) {
	w.Writef("export type %sUniqueConditions = ", model.Name)
	w.Indent()
	for _, f := range model.Fields {
		var tsType string

		switch {
		case f.Unique || f.PrimaryKey:
			tsType = toTypeScriptType(f.Type, false)
		case proto.IsHasMany(f):
			// If a model "has one" of another model then you can
			// do a lookup on any of that models unique fields
			tsType = fmt.Sprintf("%sUniqueConditions", f.Type.ModelName.Value)
		default:
			// TODO: support f.UniqueWith for compound unique constraints
			continue
		}

		w.Writeln("")
		w.Writef("| {%s: %s}", f.Name, tsType)
	}
	w.Writeln(";")
	w.Dedent()
}

func writeModelAPIDeclaration(w *codegen.Writer, model *proto.Model) {
	w.Writef("export type %sAPI = {\n", model.Name)
	w.Indent()
	w.Writef("create(values: %sCreateValues): Promise<%s>;\n", model.Name, model.Name)
	w.Writef("update(where: %sUniqueConditions, values: Partial<%s>): Promise<%s>;\n", model.Name, model.Name, model.Name)
	w.Writef("delete(where: %sUniqueConditions): Promise<string>;\n", model.Name)
	w.Writef("findOne(where: %sUniqueConditions): Promise<%s | null>;\n", model.Name, model.Name)
	w.Writef("findMany(where: %sWhereConditions): Promise<%s[]>;\n", model.Name, model.Name)
	w.Writef("where(where: %sWhereConditions): %sQueryBuilder;\n", model.Name, model.Name)
	w.Dedent()
	w.Writeln("}")
}

func writeModelQueryBuilderDeclaration(w *codegen.Writer, model *proto.Model) {
	w.Writef("export type %sQueryBuilder = {\n", model.Name)
	w.Indent()
	w.Writef("where(where: %sWhereConditions): %sQueryBuilder;\n", model.Name, model.Name)
	w.Writef("orWhere(where: %sWhereConditions): %sQueryBuilder;\n", model.Name, model.Name)
	w.Writef("findMany(): Promise<%s[]>;\n", model.Name)
	w.Dedent()
	w.Writeln("}")
}

func writeEnumObject(w *codegen.Writer, enum *proto.Enum) {
	w.Writef("module.exports.%s = {\n", enum.Name)
	w.Indent()
	for _, v := range enum.Values {
		w.Write(v.Name)
		w.Write(": ")
		w.Writef(`"%s"`, v.Name)
		w.Writeln(",")
	}
	w.Dedent()
	w.Writeln("};")
}

func writeEnum(w *codegen.Writer, enum *proto.Enum) {
	w.Writef("export enum %s {\n", enum.Name)
	w.Indent()
	for _, v := range enum.Values {
		w.Write(v.Name)
		w.Write(" = ")
		w.Writef(`"%s"`, v.Name)
		w.Writeln(",")
	}
	w.Dedent()
	w.Writeln("}")
}

func writeEnumWhereCondition(w *codegen.Writer, enum *proto.Enum) {
	w.Writef("export interface %sWhereCondition {\n", enum.Name)
	w.Indent()
	w.Write("equals?: ")
	w.Write(enum.Name)
	w.Writeln(" | null;")
	w.Write("oneOf?: ")
	w.Write(enum.Name)
	w.Write("[]")
	w.Writeln(" | null;")
	w.Dedent()
	w.Writeln("}")
}

func writeDatabaseInterface(w *codegen.Writer, schema *proto.Schema) {
	w.Writeln("interface database {")
	w.Indent()
	for _, model := range schema.Models {
		w.Writef("%s: %sTable;", casing.ToSnake(model.Name), model.Name)
		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")
	w.Writeln("export declare function getDatabase(): Kysely<database>;")
}

func writeAPIDeclarations(w *codegen.Writer, schema *proto.Schema) {
	w.Writeln("export type ModelsAPI = {")
	w.Indent()
	for _, model := range schema.Models {
		w.Write(casing.ToLowerCamel(model.Name))
		w.Write(": ")
		w.Writef(`%sAPI`, model.Name)
		w.Writeln(";")
	}
	w.Dedent()
	w.Writeln("}")
	w.Writeln("export declare const models: ModelsAPI;")
	w.Writeln("export declare const permissions: runtime.Permissions;")

	w.Writeln("type Environment = {")

	w.Indent()

	for _, variable := range schema.EnvironmentVariables {
		w.Writef("%s: string;\n", variable.Name)
	}

	w.Dedent()
	w.Writeln("}")
	w.Writeln("type Secrets = {")

	w.Indent()

	for _, secret := range schema.Secrets {
		w.Writef("%s: string;\n", secret.Name)
	}

	w.Dedent()
	w.Writeln("}")
	w.Writeln("")

	w.Writeln("export interface ContextAPI extends runtime.ContextAPI {")
	w.Indent()
	w.Writeln("secrets: Secrets;")
	w.Writeln("env: Environment;")
	w.Writeln("identity?: Identity;")
	w.Writeln("now(): Date;")
	w.Dedent()
	w.Writeln("}")
}

func writeAPIFactory(w *codegen.Writer, schema *proto.Schema) {
	w.Writeln("function createContextAPI({ responseHeaders, meta }) {")
	w.Indent()
	w.Writeln("const headers = new runtime.RequestHeaders(meta.headers);")
	w.Writeln("const response = { headers: responseHeaders }")
	w.Writeln("const now = () => { return new Date(); };")
	w.Writeln("const { identity } = meta;")
	w.Writeln("const isAuthenticated = identity != null;")
	w.Writeln("const env = {")
	w.Indent()

	for _, variable := range schema.EnvironmentVariables {
		// fetch the value of the env var from the process.env (will pull the value based on the current environment)
		// outputs "key: process.env["key"] || []"
		w.Writef("%s: process.env[\"%s\"] || \"\",\n", variable.Name, variable.Name)
	}

	w.Dedent()
	w.Writeln("};")
	w.Writeln("const secrets = {")
	w.Indent()

	for _, secret := range schema.Secrets {
		w.Writef("%s: meta.secrets.%s || \"\",\n", secret.Name, secret.Name)
	}

	w.Dedent()
	w.Writeln("};")
	w.Writeln("return { headers, response, identity, env, now, secrets, isAuthenticated };")
	w.Dedent()
	w.Writeln("};")

	w.Writeln("function createModelAPI() {")
	w.Indent()
	w.Writeln("return {")
	w.Indent()
	for _, model := range schema.Models {
		w.Write(casing.ToLowerCamel(model.Name))
		w.Write(": ")
		w.Writef(`new runtime.ModelAPI("%s", %sDefaultValues, tableConfigMap)`, casing.ToSnake(model.Name), casing.ToLowerCamel(model.Name))
		w.Writeln(",")
	}
	w.Dedent()
	w.Writeln("};")
	w.Dedent()
	w.Writeln("};")

	w.Writeln("function createPermissionApi() {")
	w.Indent()
	w.Writeln("return new runtime.Permissions();")
	w.Dedent()
	w.Writeln("};")

	w.Writeln(`module.exports.models = createModelAPI();`)
	w.Writeln(`module.exports.permissions = createPermissionApi();`)
	w.Writeln("module.exports.createContextAPI = createContextAPI;")
}

func writeTableConfig(w *codegen.Writer, models []*proto.Model) {
	w.Write("const tableConfigMap = ")

	// In case the words map and string over and over aren't clear enough
	// for you see the packages/functions-runtime/src/ModelAPI.js file for
	// docs on how this object is expected to be structured
	tableConfigMap := map[string]map[string]map[string]string{}

	for _, model := range models {
		for _, field := range model.Fields {
			if field.Type.Type != proto.Type_TYPE_MODEL {
				continue
			}

			relationshipConfig := map[string]string{
				"referencesTable": casing.ToSnake(field.Type.ModelName.Value),
				"foreignKey":      casing.ToSnake(proto.GetForignKeyFieldName(models, field)),
			}

			switch {
			case proto.IsHasOne(field):
				relationshipConfig["relationshipType"] = "hasOne"
			case proto.IsHasMany(field):
				relationshipConfig["relationshipType"] = "hasMany"
			case proto.IsBelongsTo(field):
				relationshipConfig["relationshipType"] = "belongsTo"
			}

			tableConfig, ok := tableConfigMap[casing.ToSnake(model.Name)]
			if !ok {
				tableConfig = map[string]map[string]string{}
				tableConfigMap[casing.ToSnake(model.Name)] = tableConfig
			}

			tableConfig[field.Name] = relationshipConfig
		}
	}

	b, _ := json.MarshalIndent(tableConfigMap, "", "    ")
	w.Write(string(b))
	w.Writeln(";")
}

func writeModelDefaultValuesFunction(w *codegen.Writer, model *proto.Model) {
	w.Writef("function %sDefaultValues() {", casing.ToLowerCamel(model.Name))
	w.Writeln("")
	w.Indent()
	w.Writeln("const r = {};")
	for _, field := range model.Fields {
		if field.DefaultValue == nil {
			continue
		}
		if field.DefaultValue.UseZeroValue {
			w.Writef("r.%s = ", field.Name)
			switch field.Type.Type {
			case proto.Type_TYPE_ID:
				w.Write("runtime.ksuid()")
			case proto.Type_TYPE_STRING:
				w.Write(`""`)
			case proto.Type_TYPE_BOOL:
				w.Write(`false`)
			case proto.Type_TYPE_INT:
				w.Write(`0`)
			case proto.Type_TYPE_DATETIME, proto.Type_TYPE_DATE, proto.Type_TYPE_TIMESTAMP:
				w.Write("new Date()")
			}
			w.Writeln(";")
			continue
		}
		// TODO: support expressions
	}
	w.Writeln("return r;")
	w.Dedent()
	w.Writeln("}")
}

func writeCustomFunctionWrapperType(w *codegen.Writer, model *proto.Model, op *proto.Operation) {
	w.Writef("export declare function %s", casing.ToCamel(op.Name))

	inputType := op.InputMessageName
	if inputType == parser.MessageFieldTypeAny {
		inputType = "any"
	}

	w.Writef("(fn: (ctx: ContextAPI, inputs: %s) => ", inputType)
	w.Write(toCustomFunctionReturnType(model, op, false))
	w.Write("): ")
	w.Write(toCustomFunctionReturnType(model, op, false))
	w.Writeln(";")
}

func toCustomFunctionReturnType(model *proto.Model, op *proto.Operation, isTestingPackage bool) string {
	returnType := "Promise<"
	sdkPrefix := ""
	if isTestingPackage {
		sdkPrefix = "sdk."
	}
	switch op.Type {
	case proto.OperationType_OPERATION_TYPE_CREATE:
		returnType += sdkPrefix + model.Name
	case proto.OperationType_OPERATION_TYPE_UPDATE:
		returnType += sdkPrefix + model.Name
	case proto.OperationType_OPERATION_TYPE_GET:
		returnType += sdkPrefix + model.Name + " | null"
	case proto.OperationType_OPERATION_TYPE_LIST:
		returnType += sdkPrefix + model.Name + "[]"
	case proto.OperationType_OPERATION_TYPE_DELETE:
		returnType += "string"
	case proto.OperationType_OPERATION_TYPE_READ, proto.OperationType_OPERATION_TYPE_WRITE:
		isAny := op.ResponseMessageName == parser.MessageFieldTypeAny

		if isAny {
			returnType += "any"
		} else {
			returnType += op.ResponseMessageName
		}
	}
	returnType += ">"
	return returnType
}

func toActionReturnType(model *proto.Model, op *proto.Operation) string {
	returnType := "Promise<"
	sdkPrefix := "sdk."

	switch op.Type {
	case proto.OperationType_OPERATION_TYPE_CREATE:
		returnType += sdkPrefix + model.Name
	case proto.OperationType_OPERATION_TYPE_UPDATE:
		returnType += sdkPrefix + model.Name
	case proto.OperationType_OPERATION_TYPE_GET:
		returnType += sdkPrefix + model.Name + " | null"
	case proto.OperationType_OPERATION_TYPE_LIST:
		returnType += "{results: " + sdkPrefix + model.Name + "[], pageInfo: runtime.PageInfo}"
	case proto.OperationType_OPERATION_TYPE_DELETE:
		// todo: create ID type
		returnType += "string"
	case proto.OperationType_OPERATION_TYPE_READ, proto.OperationType_OPERATION_TYPE_WRITE:
		returnType += op.ResponseMessageName
	}

	returnType += ">"
	return returnType
}

func generateDevelopmentServer(schema *proto.Schema) codegen.GeneratedFiles {
	w := &codegen.Writer{}
	w.Writeln(`import { handleRequest, tracing } from '@teamkeel/functions-runtime';`)
	w.Writeln(`import { createContextAPI, permissionFns } from '@teamkeel/sdk';`)
	w.Writeln(`import { createServer } from "http";`)

	functions := []*proto.Operation{}
	for _, model := range schema.Models {
		for _, op := range model.Operations {
			if op.Implementation != proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM {
				continue
			}
			functions = append(functions, op)
			// namespace import to avoid naming clashes
			w.Writef(`import function_%s from "../functions/%s.ts"`, op.Name, op.Name)
			w.Writeln(";")
		}
	}

	w.Writeln("const functions = {")
	w.Indent()
	for _, fn := range functions {
		w.Writef("%s: function_%s,", fn.Name, fn.Name)
		w.Writeln("")
	}
	w.Dedent()
	w.Writeln("}")

	w.Writeln("const actionTypes = {")
	w.Indent()

	for _, fn := range functions {
		w.Writef("%s: \"%s\",\n", fn.Name, fn.Type.String())
	}

	w.Dedent()
	w.Writeln("}")

	w.Writeln(`
const listener = async (req, res) => {
	const u = new URL(req.url, "http://" + req.headers.host);
	if (req.method === "GET" && u.pathname === "/_health") {
		res.statusCode = 200;
		res.end();
		return;
	}

	if (req.method === "POST") {
		const buffers = [];
		for await (const chunk of req) {
			buffers.push(chunk);
		}
		const data = Buffer.concat(buffers).toString();
		const json = JSON.parse(data);

		const rpcResponse = await handleRequest(json, {
			functions,
			createContextAPI,
			actionTypes,
			permissionFns,
		});

		res.statusCode = 200;
		res.setHeader('Content-Type', 'application/json');
		res.write(JSON.stringify(rpcResponse));
		res.end();
		return;
	}

	res.statusCode = 400;
	res.end();
};

tracing.init();

const server = createServer(listener);
const port = (process.env.PORT && parseInt(process.env.PORT, 10)) || 3001;
server.listen(port);`)

	return []*codegen.GeneratedFile{
		{
			Path:     ".build/server.js",
			Contents: w.String(),
		},
	}
}

func generateTestingPackage(schema *proto.Schema) codegen.GeneratedFiles {
	js := &codegen.Writer{}
	types := &codegen.Writer{}

	// The testing package uses ES modules as it only used in the context of running tests
	// with Vitest
	js.Writeln(`import sdk from "@teamkeel/sdk"`)
	js.Writeln("const { getDatabase, models } = sdk;")
	js.Writeln(`import { ActionExecutor, sql } from "@teamkeel/testing-runtime";`)
	js.Writeln("")
	js.Writeln("export { models };")
	js.Writeln("export const actions = new ActionExecutor({});")
	js.Writeln("export async function resetDatabase() {")
	js.Indent()
	js.Writeln("const db = getDatabase();")
	js.Write("await sql`TRUNCATE TABLE ")
	tableNames := []string{}
	for _, model := range schema.Models {
		tableNames = append(tableNames, casing.ToSnake(model.Name))
	}
	js.Writef("%s CASCADE", strings.Join(tableNames, ","))
	js.Writeln("`.execute(db);")
	js.Dedent()
	js.Writeln("}")

	writeTestingTypes(types, schema)

	return codegen.GeneratedFiles{
		{
			Path:     "node_modules/@teamkeel/testing/index.mjs",
			Contents: js.String(),
		},
		{
			Path:     "node_modules/@teamkeel/testing/index.d.ts",
			Contents: types.String(),
		},
		{
			Path:     "node_modules/@teamkeel/testing/package.json",
			Contents: `{"name": "@teamkeel/testing", "type": "module", "exports": "./index.mjs"}`,
		},
	}
}

func generateTestingSetup() codegen.GeneratedFiles {
	return codegen.GeneratedFiles{
		{
			Path: ".build/vitest.config.mjs",
			Contents: `
import { defineConfig } from "vitest/config";

export default defineConfig({
	test: {
		setupFiles: [__dirname + "/vitest.setup"],
	},
});
			`,
		},
		{
			Path: ".build/vitest.setup.mjs",
			Contents: `
import { expect } from "vitest";
import { toHaveError, toHaveAuthorizationError } from "@teamkeel/testing-runtime";

expect.extend({
	toHaveError,
	toHaveAuthorizationError,
});
			`,
		},
	}
}

func writeTestingTypes(w *codegen.Writer, schema *proto.Schema) {
	w.Writeln(`import * as sdk from "@teamkeel/sdk";`)
	w.Writeln(`import * as runtime from "@teamkeel/functions-runtime";`)

	// We need to import the testing-runtime package to get
	// the types for the extended vitest matchers e.g. expect(v).toHaveAuthorizationError()
	w.Writeln(`import "@teamkeel/testing-runtime";`)
	w.Writeln("")

	// For the testing package we need input and response types for all actions
	writeMessages(w, schema, true)

	w.Writeln("declare class ActionExecutor {")
	w.Indent()
	w.Writeln("withIdentity(identity: sdk.Identity): ActionExecutor;")
	w.Writeln("withAuthToken(token: string): ActionExecutor;")
	for _, model := range schema.Models {
		for _, op := range model.Operations {
			msg := proto.FindMessage(schema.Messages, op.InputMessageName)

			w.Writef("%s(i", op.Name)

			// Check that all of the top level fields in the matching message are optional
			// If so, then we can make it so you don't even need to specify the key
			// example, this allows for:
			// await actions.listActivePublishersWithActivePosts();
			// instead of:
			// const { results: publishers } =
			// await actions.listActivePublishersWithActivePosts({ where: {} });
			if lo.EveryBy(msg.Fields, func(f *proto.MessageField) bool {
				return f.Optional
			}) {
				w.Write("?")
			}

			w.Writef(`: %s): %s`, op.InputMessageName, toActionReturnType(model, op))
			w.Writeln(";")
		}
	}
	w.Dedent()
	w.Writeln("}")
	w.Writeln("export declare const actions: ActionExecutor;")
	w.Writeln("export declare const models: sdk.ModelsAPI;")
	w.Writeln("export declare function resetDatabase(): Promise<void>;")
}

func toTypeScriptType(t *proto.TypeInfo, isTestingPackage bool) (ret string) {
	switch t.Type {
	case proto.Type_TYPE_ID:
		ret = "string"
	case proto.Type_TYPE_STRING:
		ret = "string"
	case proto.Type_TYPE_BOOL:
		ret = "boolean"
	case proto.Type_TYPE_INT:
		ret = "number"
	case proto.Type_TYPE_DATE, proto.Type_TYPE_DATETIME, proto.Type_TYPE_TIMESTAMP:
		ret = "Date"
	case proto.Type_TYPE_ENUM:
		ret = t.EnumName.Value
	case proto.Type_TYPE_MESSAGE:
		ret = t.MessageName.Value
	case proto.Type_TYPE_MODEL:
		// models are imported from the sdk
		if isTestingPackage {
			ret = fmt.Sprintf("sdk.%s", t.ModelName.Value)
		} else {
			ret = t.ModelName.Value
		}
	default:
		ret = "any"
	}

	return ret
}

func toWhereConditionType(f *proto.Field) string {
	switch f.Type.Type {
	case proto.Type_TYPE_ID:
		return "runtime.IDWhereCondition"
	case proto.Type_TYPE_STRING:
		return "runtime.StringWhereCondition"
	case proto.Type_TYPE_BOOL:
		return "runtime.BooleanWhereCondition"
	case proto.Type_TYPE_INT:
		return "runtime.NumberWhereCondition"
	case proto.Type_TYPE_DATE, proto.Type_TYPE_DATETIME, proto.Type_TYPE_TIMESTAMP:
		return "runtime.DateWhereCondition"
	case proto.Type_TYPE_ENUM:
		return fmt.Sprintf("%sWhereCondition", f.Type.EnumName.Value)
	default:
		return "any"
	}
}
