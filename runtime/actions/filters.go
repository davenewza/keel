package actions

import (
	"fmt"

	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema/parser"
)

// Applies all implicit input filters to the query.
func (query *QueryBuilder) applyImplicitFilters(scope *Scope, args map[string]any) error {
	message := proto.FindWhereInputMessage(scope.schema, scope.operation.Name)
	if message == nil {
		return nil
	}

	for _, input := range message.Fields {
		if !input.IsModelField() {
			// Skip if this is an explicit input (probably used in a @where)
			continue
		}

		fieldName := input.Name
		value, ok := args[fieldName]

		if !ok {
			return fmt.Errorf("this expected input: %s, is missing from this provided args map: %+v", fieldName, args)
		}

		err := query.whereByImplicitFilter(scope, input.Target, fieldName, Equals, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// Applies all exlicit where attribute filters to the query.
func (query *QueryBuilder) applyExplicitFilters(scope *Scope, args map[string]any) error {
	for _, where := range scope.operation.WhereExpressions {
		expression, err := parser.ParseExpression(where.Source)
		if err != nil {
			return err
		}

		// Resolve the database statement for this expression
		err = query.whereByExpression(scope, expression, args)
		if err != nil {
			return err
		}
	}

	return nil
}
