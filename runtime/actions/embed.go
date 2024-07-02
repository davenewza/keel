package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/teamkeel/keel/casing"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema/parser"
)

func resolveEmbeddedData(ctx context.Context, schema *proto.Schema, sourceModel *proto.Model, sourceID string, fragments []string) (any, error) {
	if len(fragments) == 0 {
		return nil, errors.New("invalid embed resolver")
	}
	embedTargetField := fragments[0]

	field := proto.FindField(schema.Models, sourceModel.GetName(), casing.ToLowerCamel(embedTargetField))
	if field == nil {
		return nil, fmt.Errorf("embed target field (%s) does not exist in model %s", embedTargetField, sourceModel.GetName())
	}

	if !proto.IsTypeModel(field) {
		return nil, fmt.Errorf("field (%s) is not a embeddable model field", embedTargetField)

	}

	relatedModelName := field.Type.ModelName.Value
	relatedModel := proto.FindModel(schema.Models, relatedModelName)
	foreignKeyField := proto.GetForignKeyFieldName(schema.Models, field)

	dbQuery := NewQuery(relatedModel)
	// we apply the where clause which will filter based on the joins set up depending on the relationship type
	err := dbQuery.Where(&QueryOperand{
		table:  casing.ToSnake(sourceModel.GetName()),
		column: casing.ToSnake(parser.FieldNameId),
	}, Equals, Value(sourceID))

	switch {
	case proto.IsBelongsTo(field):
		dbQuery.Join(
			sourceModel.Name,
			&QueryOperand{
				table:  casing.ToSnake(sourceModel.Name),
				column: casing.ToSnake(foreignKeyField),
			},
			&QueryOperand{
				table:  casing.ToSnake(relatedModelName),
				column: casing.ToSnake(parser.FieldNameId),
			})
		if err != nil {
			return nil, fmt.Errorf("applying sql where: %w", err)
		}

		stmt := dbQuery.SelectStatement()
		result, err := stmt.ExecuteToSingle(ctx)
		if err != nil {
			return nil, fmt.Errorf("executing query to single: %w", err)
		}

		// recurse and resolve child embeds
		if len(fragments) > 1 {
			if childId, ok := result[parser.FieldNameId].(string); ok {
				childEmbed, err := resolveEmbeddedData(ctx, schema, relatedModel, childId, fragments[1:])
				if err != nil {
					return nil, fmt.Errorf("retrieving child embed: %w", err)
				}
				result[fragments[1]] = childEmbed
			}
		}

		return result, nil
	case proto.IsHasMany(field):
		dbQuery.Join(
			sourceModel.Name,
			&QueryOperand{
				table:  casing.ToSnake(sourceModel.GetName()),
				column: casing.ToSnake(parser.FieldNameId),
			},
			&QueryOperand{
				table:  casing.ToSnake(relatedModelName),
				column: casing.ToSnake(foreignKeyField),
			})
		stmt := dbQuery.SelectStatement()
		result, _, err := stmt.ExecuteToMany(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("executing query to many: %w", err)
		}

		// recurse and resolve child embeds for each of our results
		if len(fragments) > 1 {
			for i := range result {
				childId, ok := result[i][parser.FieldNameId].(string)
				if !ok {
					// we skip if we don't have a child embed id
					continue
				}
				childEmbed, err := resolveEmbeddedData(ctx, schema, relatedModel, childId, fragments[1:])
				if err != nil {
					return nil, fmt.Errorf("retrieving child embed: %w", err)
				}
				result[i][fragments[1]] = childEmbed
			}
		}

		return result, nil
	case proto.IsHasOne(field):
		dbQuery.Join(
			sourceModel.Name,
			&QueryOperand{
				table:  casing.ToSnake(sourceModel.GetName()),
				column: casing.ToSnake(parser.FieldNameId),
			},
			&QueryOperand{
				table:  casing.ToSnake(relatedModelName),
				column: casing.ToSnake(foreignKeyField),
			})

		stmt := dbQuery.SelectStatement()
		result, err := stmt.ExecuteToSingle(ctx)
		if err != nil {
			return nil, fmt.Errorf("executing query to single: %w", err)
		}

		// recurse and resolve child embeds
		if len(fragments) > 1 {
			if childId, ok := result[parser.FieldNameId].(string); ok {
				childEmbed, err := resolveEmbeddedData(ctx, schema, relatedModel, childId, fragments[1:])
				if err != nil {
					return nil, fmt.Errorf("retrieving child embed: %w", err)
				}
				result[fragments[1]] = childEmbed
			}
		}

		return result, nil
	}

	return nil, errors.New("unsupported embed type")
}