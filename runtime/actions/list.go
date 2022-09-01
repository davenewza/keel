package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"gorm.io/gorm"
)

// List implements a Keel List Action.
// In quick overview this means generating a SQL query
// based on the List operation's Inputs and Where clause,
// running that query, and returning the results.
func List(
	ctx context.Context,
	operation *proto.Operation,
	schema *proto.Schema,
	inputs interface{}) (interface{}, error) {
	listInput, err := buildListInput(operation, inputs)
	if err != nil {
		return nil, err
	}
	db, err := runtimectx.GetDB(ctx)
	if err != nil {
		return nil, err
	}

	model := proto.FindModel(schema.Models, operation.ModelName)

	tableName := strcase.ToSnake(model.Name)

	// Initialise a query on the table = to which we'll add Where clauses.
	tx := db.Table(tableName)

	// Add the WHERE clauses derived from the inputs.
	tx, err = addListInputFilters(operation, listInput, tx)
	if err != nil {
		return nil, err
	}

	// todo
	// Add the WHERE clauses derived from EXPLICIT inputs (i.e. the operation's where clauses).
	// tx, err = addWhereFilters(operation, schema, args, tx)
	// if err != nil {
	// 	return nil, err
	// }

	// Todo: should we validate the type of the values?, or let postgres object to them later?

	// Execute the SQL query.
	result := []map[string]any{}
	tx = tx.Find(&result)
	if tx.Error != nil {
		return nil, tx.Error
	}
	res := toLowerCamelMaps(result)

	return res, nil
}

// addListInputFilters adds Where clauses to the given gorm.DB corresponding to the
// given ListInput.
func addListInputFilters(op *proto.Operation, listInput *ListInput, tx *gorm.DB) (*gorm.DB, error) {
	// We'll look at each of the fields specified as inputs by the operation in the schema,
	// and then try to find these referenced by the where filters in the given ListInput.
	for _, schemaInput := range op.Inputs {
		if schemaInput.Behaviour != proto.InputBehaviour_INPUT_BEHAVIOUR_IMPLICIT {
			return nil, errors.New("not yet supported: explicit inputs for list actions")
		}
		expectedFieldName := schemaInput.Target[0]
		var matchingWhere *Where
		for _, where := range listInput.Wheres {
			if where.Name == expectedFieldName {
				matchingWhere = where
				break
			}
		}
		if matchingWhere == nil {
			return nil, fmt.Errorf("operation expects an input named: <%s>, but none is present on the request", expectedFieldName)
		}
		tx = addWhere(tx, expectedFieldName, matchingWhere)
	}
	return tx, nil
}

// addWhere updates the given gorm.DB tx with a where clause that represents the given
// query.
func addWhere(tx *gorm.DB, columnName string, where *Where) *gorm.DB {
	w := fmt.Sprintf("%s = ?", strcase.ToSnake(columnName))
	return tx.Where(w, where.Operand)
}

// buildListInput consumes the dictionary that carries the LIST operation input values on the
// incoming request, and composes a corresponding actions.ListInput object that is good
// to pass to the generic actions.List() function.
func buildListInput(operation *proto.Operation, requestInputArgs any) (*ListInput, error) {
	page := Page{}
	wheres := []*Where{}

	argsMap, ok := requestInputArgs.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot cast this: %+v to map[string]any", requestInputArgs)
	}
	whereInputs, ok := argsMap["where"]
	if !ok {
		return nil, fmt.Errorf("arguments map does not contain a where key: %v", argsMap)
	}
	whereInputsAsMap, ok := whereInputs.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot cast this: %v to a map[string]any", whereInputs)
	}

	for argName, argValue := range whereInputsAsMap {
		argValueAsMap, ok := argValue.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot cast this: %v to a map[string]any", argValue)
		}
		for operator, operand := range argValueAsMap {
			_ = operator
			where := &Where{
				Name:     argName,
				Operator: OperatorEquals, // todo this should be f(operand),
				Operand:  operand,
			}
			wheres = append(wheres, where)
		}
	}
	inp := &ListInput{
		Page:   page,
		Wheres: wheres,
	}
	return inp, nil
}