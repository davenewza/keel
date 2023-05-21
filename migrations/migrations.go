package migrations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	ChangeTypeAdded    = "ADDED"
	ChangeTypeRemoved  = "REMOVED"
	ChangeTypeModified = "MODIFIED"
)

var ErrNoStoredSchema = errors.New("no schema stored in keel_schema table")
var ErrMultipleStoredSchemas = errors.New("more than one schema found in keel_schema table")

type DatabaseChange struct {
	// The model this change applies to
	Model string

	// The field this change applies to (might be empty)
	Field string

	// The type of change
	Type string
}

type Migrations struct {
	Schema *proto.Schema

	// Describes the changes that will be applied to the database
	// if SQL is run
	Changes []*DatabaseChange

	// The SQL to run to execute the database schema changes
	SQL string
}

// HasModelFieldChanges returns true if the migrations contain model field changes to be applied
func (m *Migrations) HasModelFieldChanges() bool {
	return m.SQL != ""
}

// Apply executes the migrations against the database
func (m *Migrations) Apply(ctx context.Context, database db.Database) error {

	sql := strings.Builder{}

	sql.WriteString(m.SQL)
	sql.WriteString("\n")

	sql.WriteString("CREATE TABLE IF NOT EXISTS keel_schema ( schema TEXT NOT NULL );\n")
	sql.WriteString("DELETE FROM keel_schema;\n")

	b, err := protojson.Marshal(m.Schema)
	if err != nil {
		return err
	}

	// Cannot use parameters as then you get an error:
	//   ERROR: cannot insert multiple commands into a prepared statement (SQLSTATE 42601)
	escapedJSON := db.QuoteLiteral(string(b))
	insertStmt := fmt.Sprintf("INSERT INTO keel_schema (schema) VALUES (%s);", escapedJSON)
	sql.WriteString(insertStmt)

	// Enable extensions
	sql.WriteString("CREATE EXTENSION IF NOT EXISTS pg_stat_statements;\n")

	_, err = database.ExecuteStatement(ctx, sql.String())

	return err
}

// Create inspects the database using gorm.DB connection
// and creates the required schema migrations that will result in
// the database reflecting the provided proto.Schema
func New(newSchema *proto.Schema, currSchema *proto.Schema) *Migrations {

	if currSchema == nil {
		currSchema = &proto.Schema{}
	}

	statements := []string{}

	changes := []*DatabaseChange{}

	currModels := proto.ModelNames(currSchema)
	newModels := proto.ModelNames(newSchema)
	modelsInCommon := lo.Intersect(newModels, currModels)

	// Models added or removed.
	modelsAdded, modelsRemoved := lo.Difference(newModels, currModels)

	for _, modelName := range modelsAdded {
		model := proto.FindModel(newSchema.Models, modelName)
		statements = append(statements, createTableStmt(model))
		changes = append(changes, &DatabaseChange{
			Model: modelName,
			Type:  ChangeTypeAdded,
		})
	}
	// the calls to createTableStmt() in the loop above ensures that all models have been
	// created with all their fields, so now we apply foreign key constraints for each of the NEW
	// models' Id fields. We cannot do this earlier because the FK sql depends on every all the
	// tables now having been defined.
	for _, modelName := range modelsAdded {
		model := proto.FindModel(newSchema.Models, modelName)
		statements = append(statements, fkConstraintsForModel(model, newSchema)...)
	}

	for _, modelName := range modelsRemoved {
		statements = append(statements, dropTableStmt(modelName))
		changes = append(changes, &DatabaseChange{
			Model: modelName,
			Type:  ChangeTypeRemoved,
		})
	}

	// Fields added or removed.
	// Note these are fields that exist on models that exist in both the old and
	// new schema.
	for _, modelName := range modelsInCommon {
		model := proto.FindModel(newSchema.Models, modelName)
		currFieldNames := proto.FieldNames(proto.FindModel(currSchema.Models, modelName))
		newFieldNames := proto.FieldNames(proto.FindModel(newSchema.Models, modelName))
		fieldsAdded, fieldsRemoved := lo.Difference(newFieldNames, currFieldNames)

		for _, fieldName := range fieldsAdded {

			field := proto.FindField(newSchema.Models, modelName, fieldName)

			// This type of field exists only in proto land - and has no corresponding
			// column in the database.
			if field.Type.Type == proto.Type_TYPE_MODEL {
				continue
			}

			statements = append(statements, addColumnStmt(modelName, field))
			changes = append(changes, &DatabaseChange{
				Model: modelName,
				Field: fieldName,
				Type:  ChangeTypeAdded,
			})

			// When the field added is a foreign key field, we add a corresponding foreign key constraint.
			if field.ForeignKeyInfo != nil {
				statements = append(statements, fkConstraint(field, model, newSchema))
			}
		}

		for _, fieldName := range fieldsRemoved {
			field := proto.FindField(currSchema.Models, modelName, fieldName)

			// Fields of type Model do not show up in the database in of themselves, so we skip them.
			// We autogenerated sibling foreign key fields for them in the proto Schema, which will
			// pass through this loop in their own right.
			if field.Type.Type == proto.Type_TYPE_MODEL {
				continue
			}

			statements = append(statements, dropColumnStmt(modelName, field))
			changes = append(changes, &DatabaseChange{
				Model: modelName,
				Field: fieldName,
				Type:  ChangeTypeRemoved,
			})
		}

		fieldsInCommon := lo.Intersect(newFieldNames, currFieldNames)
		for _, fieldName := range fieldsInCommon {
			newField := proto.FindField(newSchema.Models, modelName, fieldName)
			currField := proto.FindField(currSchema.Models, modelName, fieldName)

			alterSQL := alterColumnStmt(modelName, newField, currField)
			if alterSQL == "" {
				continue
			}

			statements = append(statements, alterSQL)
			changes = append(changes, &DatabaseChange{
				Model: modelName,
				Field: fieldName,
				Type:  ChangeTypeModified,
			})
		}
	}

	return &Migrations{
		Schema:  newSchema,
		Changes: changes,
		SQL:     strings.TrimSpace(strings.Join(statements, "\n")),
	}
}

func keelSchemaTableExists(ctx context.Context, database db.Database) (bool, error) {

	// to_regclass docs - https://www.postgresql.org/docs/current/functions-info.html#FUNCTIONS-INFO-CATALOG-TABLE
	// translates a textual relation name to its OID ... this function will
	// return NULL rather than throwing an error if the name is not found.
	result, err := database.ExecuteQuery(ctx, "SELECT to_regclass('keel_schema') AS name")
	if err != nil {
		return false, err
	}

	return result.Rows[0]["name"] != nil, nil
}

func GetCurrentSchema(ctx context.Context, database db.Database) (*proto.Schema, error) {
	exists, err := keelSchemaTableExists(ctx, database)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	result, err := database.ExecuteQuery(ctx, "SELECT schema FROM keel_schema")
	if err != nil {
		return nil, err
	}

	if len(result.Rows) == 0 {
		return nil, ErrNoStoredSchema
	} else if len(result.Rows) > 1 {
		return nil, ErrMultipleStoredSchemas
	}

	schema, ok := result.Rows[0]["schema"].(string)
	if !ok {
		return nil, errors.New("schema could not be converted to string")
	}

	var protoSchema proto.Schema
	err = protojson.Unmarshal([]byte(schema), &protoSchema)
	if err != nil {
		return nil, err
	}

	return &protoSchema, nil
}

// fkConstraintsForModel generates foreign key constraint statements for each of fields marked as
// being foreign keys in the given model.
// present in the given model.
func fkConstraintsForModel(model *proto.Model, schema *proto.Schema) (fkStatements []string) {
	fkFields := proto.ForeignKeyFields(model)
	for _, field := range fkFields {
		stmt := fkConstraint(field, model, schema)
		fkStatements = append(fkStatements, stmt)
	}
	return fkStatements
}

// fkConstraint generates a foreign key constraint statement for the given foreign key field.
func fkConstraint(field *proto.Field, thisModel *proto.Model, schema *proto.Schema) (fkStatement string) {
	fki := field.ForeignKeyInfo
	onDelete := lo.Ternary(field.Optional, "SET NULL", "CASCADE")
	stmt := addForeignKeyConstraintStmt(
		Identifier(thisModel.Name),
		Identifier(field.Name),
		Identifier(fki.RelatedModelName),
		Identifier(fki.RelatedModelField),
		onDelete,
	)
	return stmt
}
