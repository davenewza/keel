package actions

import (
	"context"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/samber/lo"
	"github.com/segmentio/ksuid"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"github.com/teamkeel/keel/schema/parser"
	"golang.org/x/exp/slices"
)

// interpretExpressionGivenArgs examines the given expression, in order to work out how to construct a gorm WHERE clause.
//
// The ONLY form we support at the moment in this infant version is this: "person.name == name"
//
// It returns a column and a value that is good to be used like this:
//
//	tx.Where(fmt.Sprintf("%s = ?", column), value)
func interpretExpressionGivenArgs(
	expr *parser.Expression,
	operation *proto.Operation,
	schema *proto.Schema,
	args map[string]any,
) (column string, value any, err error) {

	// Make sure the expression is in the form we can handle.

	conditions := expr.Conditions()
	if len(conditions) != 1 {
		return "", nil, fmt.Errorf("cannot yet handle multiple conditions, have: %d", len(conditions))
	}
	condition := conditions[0]
	cType := condition.Type()
	if cType != parser.LogicalCondition {
		return "", nil, fmt.Errorf("cannot yet handle condition types other than LogicalCondition, have: %s", cType)
	}

	if condition.Operator.ToString() != parser.OperatorEquals {
		return "", nil, fmt.Errorf(
			"cannot yet handle operators other than OperatorEquals, have: %s",
			condition.Operator.ToString())
	}

	if condition.LHS.Type() != parser.TypeIdent {
		return "", nil, fmt.Errorf("cannot handle LHS of type other than TypeIdent, have: %s", condition.LHS.Type())
	}
	if condition.RHS.Type() != parser.TypeIdent {
		return "", nil, fmt.Errorf("cannot handle RHS of type other than TypeIdent, have: %s", condition.LHS.Type())
	}

	lhs := condition.LHS
	if len(lhs.Ident.Fragments) != 2 {
		return "", nil, fmt.Errorf("cannot handle LHS identifier unless it has 2 fragments, have: %d", len(lhs.Ident.Fragments))
	}

	rhs := condition.RHS
	if len(rhs.Ident.Fragments) != 1 {
		return "", nil, fmt.Errorf("cannot handle RHS identifier unless it has 1 fragment, have: %d", len(rhs.Ident.Fragments))
	}

	// Make sure the first fragment in the LHS is the name of the model of which this operation is part.
	// e.g. "person" in the example above.
	modelTarget := strcase.ToCamel(lhs.Ident.Fragments[0].Fragment)
	if modelTarget != operation.ModelName {
		return "", nil, fmt.Errorf("can only handle the first LHS fragment referencing the Operation's model, have: %s", modelTarget)
	}

	// Make sure the second fragment in the LHS is the name of a field of the model of which this operation is part.
	// e.g. "name" in the example above.
	fieldName := lhs.Ident.Fragments[1].Fragment
	if !proto.ModelHasField(schema, modelTarget, fieldName) {
		return "", nil, fmt.Errorf("this model: %s, does not have a field of name: %s", modelTarget, fieldName)
	}

	// Make sure the single fragment in the RHS matches up with an expected Input for this operation.
	inputName := rhs.Ident.Fragments[0].Fragment
	if !proto.OperationHasInput(operation, inputName) {
		return "", nil, fmt.Errorf("operation does not define an input called: %s", inputName)
	}

	// Make sure the "where" part of the args on the specified input has been provided in the given Args
	arg, ok := args[inputName]
	if !ok {
		return "", nil, fmt.Errorf("request does not have provide argument of name: %s", inputName)
	}

	// Now we have all the data we need to return
	return strcase.ToSnake(fieldName), arg, nil
}

// TODO - need to DRY up and rationalise the functions above and below!!!

// interpretExpressionField examines the given expression, in order to work out how to construct a gorm WHERE clause.
//
// The ONLY form we support at the moment in this infant version is this: "person.name == <an-input-name>"
//
// It returns the field name being assigned to.
//
//	tx.Where(fmt.Sprintf("%s = ?", column), value)
func interpretExpressionField(
	expr *parser.Expression,
	operation *proto.Operation,
	schema *proto.Schema,
) (column string, err error) {

	// Make sure the expression is in the form we can handle.

	conditions := expr.Conditions()
	if len(conditions) != 1 {
		return "", fmt.Errorf("cannot yet handle multiple conditions, have: %d", len(conditions))
	}
	condition := conditions[0]
	cType := condition.Type()
	if cType != parser.LogicalCondition {
		return "", fmt.Errorf("cannot yet handle condition types other than LogicalCondition, have: %s", cType)
	}

	if condition.Operator.ToString() != parser.OperatorEquals {
		return "", fmt.Errorf(
			"cannot yet handle operators other than OperatorEquals, have: %s",
			condition.Operator.ToString())
	}

	if condition.LHS.Type() != parser.TypeIdent {
		return "", fmt.Errorf("cannot handle LHS of type other than TypeIdent, have: %s", condition.LHS.Type())
	}
	if condition.RHS.Type() != parser.TypeIdent {
		return "", fmt.Errorf("cannot handle RHS of type other than TypeIdent, have: %s", condition.LHS.Type())
	}

	lhs := condition.LHS
	if len(lhs.Ident.Fragments) != 2 {
		return "", fmt.Errorf("cannot handle LHS identifier unless it has 2 fragments, have: %d", len(lhs.Ident.Fragments))
	}

	rhs := condition.RHS
	if len(rhs.Ident.Fragments) != 1 {
		return "", fmt.Errorf("cannot handle RHS identifier unless it has 1 fragment, have: %d", len(rhs.Ident.Fragments))
	}

	// Make sure the first fragment in the LHS is the name of the model of which this operation is part.
	// e.g. "person" in the example above.
	modelTarget := strcase.ToCamel(lhs.Ident.Fragments[0].Fragment)
	if modelTarget != operation.ModelName {
		return "", fmt.Errorf("can only handle the first LHS fragment referencing the Operation's model, have: %s", modelTarget)
	}

	// Make sure the second fragment in the LHS is the name of a field of the model of which this operation is part.
	// e.g. "name" in the example above.
	fieldName := lhs.Ident.Fragments[1].Fragment
	if !proto.ModelHasField(schema, modelTarget, fieldName) {
		return "", fmt.Errorf("this model: %s, does not have a field of name: %s", modelTarget, fieldName)
	}

	// Now we have all the data we need to return
	return fieldName, nil
}

// EvaluatePermissions will evaluate all the permission conditions defined on models and operations
func EvaluatePermissions(
	ctx context.Context,
	op *proto.Operation,
	schema *proto.Schema,
	data map[string]any,
) (authorized bool, err error) {
	permissions := []*proto.PermissionRule{}

	model := proto.FindModel(schema.Models, op.ModelName)

	modelPermissions := lo.Filter(model.Permissions, func(modelPermission *proto.PermissionRule, _ int) bool {
		return slices.Contains(modelPermission.OperationsTypes, op.Type)
	})

	permissions = append(permissions, modelPermissions...)

	if op.Permissions != nil {
		permissions = append(permissions, op.Permissions...)
	}

	// todo: remove this once we make permissions a requirement for any access
	// https://linear.app/keel/issue/RUN-135/permissions-required-for-access-at-all
	if len(permissions) == 0 {
		return true, nil
	}

	for _, permission := range permissions {
		if permission.Expression != nil {
			expression, err := parser.ParseExpression(permission.Expression.Source)
			if err != nil {
				return false, err
			}

			authorized, err := evaluateExpression(ctx, expression, op, schema, data)
			if err != nil {
				return false, err
			}
			if authorized {
				return true, nil
			}
		}
	}

	return false, nil
}

// evaluateExpression evaluates a given conditional expression
func evaluateExpression(
	context context.Context,
	expression *parser.Expression,
	operation *proto.Operation,
	schema *proto.Schema,
	data map[string]any,
) (result bool, err error) {

	conditions := expression.Conditions()
	if len(conditions) != 1 {
		return false, fmt.Errorf("cannot yet handle multiple conditions, have: %d", len(conditions))
	}
	condition := conditions[0]

	if condition.Type() == parser.ValueCondition {
		valueType, _ := GetOperandType(condition.LHS, operation, schema)
		if valueType != proto.Type_TYPE_BOOL {
			return false, fmt.Errorf("value operand must be of type bool, not %s", condition.Type())
		}

		value, err := evaluateOperandValue(context, condition.LHS, operation, schema, data, valueType)
		if err != nil {
			return false, err
		}

		return value.(bool), nil
	}

	if condition.Type() != parser.LogicalCondition {
		return false, fmt.Errorf("can only handle condition type of LogicalCondition, have: %s", condition.Type())
	}

	// Determine the native protobuf type underlying the expression comparison
	var operandType proto.Type
	lhsType, _ := GetOperandType(condition.LHS, operation, schema)
	rhsType, _ := GetOperandType(condition.LHS, operation, schema)
	switch {
	case lhsType != proto.Type_TYPE_UNKNOWN && (lhsType == rhsType || rhsType == proto.Type_TYPE_UNKNOWN):
		operandType = lhsType
	case rhsType != proto.Type_TYPE_UNKNOWN && (lhsType == rhsType || lhsType == proto.Type_TYPE_UNKNOWN):
		operandType = rhsType
	default:
		return false, fmt.Errorf("lhs: %s, and rhs: %s, are not of the same native type", lhsType, rhsType)
	}

	// Evaluate the values on each side of the expression
	lhsValue, err := evaluateOperandValue(context, condition.LHS, operation, schema, data, operandType)
	if err != nil {
		return false, err
	}
	rhsValue, err := evaluateOperandValue(context, condition.RHS, operation, schema, data, operandType)
	if err != nil {
		return false, err
	}

	// The LHS and RHS types must be equal unless the RHS is a null literal
	if lhsType != rhsType && rhsValue != nil {
		return false, fmt.Errorf("lhs type: %s, and rhs type: %s, are not the same", lhsType, rhsType)
	}

	// Evaluate the condition
	return evaluateOperandCondition(lhsValue, rhsValue, operandType, condition.Operator)
}

// GetOperandType determines the underlying type to compare with for an operand
func GetOperandType(
	operand *parser.Operand,
	operation *proto.Operation,
	schema *proto.Schema,
) (proto.Type, error) {

	if operand.Ident == nil {
		switch {
		case operand.String != nil:
			return proto.Type_TYPE_STRING, nil
		case operand.Number != nil:
			return proto.Type_TYPE_INT, nil
		case operand.True || operand.False:
			return proto.Type_TYPE_BOOL, nil
		case operand.Null:
			return proto.Type_TYPE_UNKNOWN, nil
		default:
			return proto.Type_TYPE_UNKNOWN, fmt.Errorf("cannot handle operand type")
		}
	}

	target := operand.Ident.Fragments[0].Fragment
	switch {
	case strcase.ToCamel(target) == operation.ModelName:
		// todo: evaluate at the database-level where applicable
		// https://linear.app/keel/issue/RUN-129/expressions-to-evaluate-at-database-level-where-applicable

		modelTarget := strcase.ToCamel(target)
		fieldName := operand.Ident.Fragments[1].Fragment

		if !proto.ModelHasField(schema, strcase.ToCamel(modelTarget), fieldName) {
			return proto.Type_TYPE_UNKNOWN, fmt.Errorf("this model: %s, does not have a field of name: %s", modelTarget, fieldName)
		}

		operandType := proto.FindField(schema.Models, strcase.ToCamel(modelTarget), fieldName).Type.Type
		return operandType, nil
	case operand.Ident.IsContext():
		fieldName := operand.Ident.Fragments[1].Fragment
		return runtimectx.ContextFieldTypes[fieldName], nil // todo: if not found
	default:
		return proto.Type_TYPE_UNKNOWN, fmt.Errorf("cannot handle operand target %s", target)
	}
}

// evaluateOperandValue evaluates the value to compare with for an operand
func evaluateOperandValue(
	context context.Context,
	operand *parser.Operand,
	operation *proto.Operation,
	schema *proto.Schema,
	data map[string]any,
	operandType proto.Type,
) (any, error) {

	isLiteral, _ := operand.IsLiteralType()

	switch {
	case isLiteral:
		return toNative(operand, operandType)
	case operand.Ident != nil && proto.EnumExists(schema.Enums, operand.Ident.Fragments[0].Fragment):
		return operand.Ident.Fragments[1].Fragment, nil
	case operand.Ident != nil && len(operand.Ident.Fragments) == 1 && data[operand.Ident.Fragments[0].Fragment] != nil:
		inputValue, _ := data[operand.Ident.Fragments[0].Fragment]
		return inputValue, nil
	case operand.Ident != nil && strcase.ToCamel(operand.Ident.Fragments[0].Fragment) == operation.ModelName:
		target := operand.Ident.Fragments[0].Fragment
		modelTarget := strcase.ToCamel(target)
		fieldName := operand.Ident.Fragments[1].Fragment

		if !proto.ModelHasField(schema, strcase.ToCamel(modelTarget), fieldName) {
			return nil, fmt.Errorf("this model: %s, does not have a field of name: %s", modelTarget, fieldName)
		}

		field := proto.FindField(schema.Models, strcase.ToCamel(modelTarget), fieldName)
		operandType := field.Type.Type
		isOptional := field.Optional
		fieldValue := data[fieldName]

		// If the value of the optional field then return nil for later comparison
		if fieldValue == nil {
			if isOptional {
				return nil, nil
			} else {
				return nil, fmt.Errorf("required field is nil: %s", fieldName)
			}
		}

		switch operandType {
		case proto.Type_TYPE_STRING, proto.Type_TYPE_BOOL:
			return fieldValue, nil
		case proto.Type_TYPE_INT:
			// todo: unify these to a single type at the source?
			switch v := fieldValue.(type) {
			case int:
				// Sourced from GraphQL input parameters.
				return int64(fieldValue.(int)), nil
			case float64:
				// Sourced from integration test framework.
				return int64(fieldValue.(float64)), nil
			case int32:
				// Sourced from database.
				return int64(fieldValue.(int32)), nil // todo: https://linear.app/keel/issue/RUN-98/number-type-as-int32-or-int64
			case int64:
				// Sourced from a default set value on a field.
				return fieldValue, nil
			default:
				return nil, fmt.Errorf("cannot yet parse %s to int64", v)
			}
		case proto.Type_TYPE_ENUM:
			return fieldValue, nil
		case proto.Type_TYPE_IDENTITY:
			switch v := fieldValue.(type) {
			case *ksuid.KSUID:
				// Sourced from GraphQL input parameters.
				return *fieldValue.(*ksuid.KSUID), nil
			case string:
				// Sourced from database.
				value, err := ksuid.Parse(fieldValue.(string))
				if err != nil {
					return nil, fmt.Errorf("cannot parse %s to ksuid", fieldValue)
				}
				return value, nil
			default:
				return nil, fmt.Errorf("cannot yet parse %s to ksuid.KSUID", v)
			}
		default:
			return nil, fmt.Errorf("cannot yet compare operand of type %s", operandType)
		}
	case operand.Ident != nil && operand.Ident.IsContextIdentityField():
		isAuthenticated, err := runtimectx.IsAuthenticated(context)
		if err != nil {
			return nil, err
		}

		if !isAuthenticated {
			return nil, nil
		}

		ksuid, err := runtimectx.GetIdentity(context)
		if err != nil {
			return nil, err
		}
		return *ksuid, nil
	case operand.Ident != nil && operand.Ident.IsContextIsAuthenticatedField():
		isAuthenticated, err := runtimectx.IsAuthenticated(context)
		if err != nil {
			return nil, err
		}
		return isAuthenticated, nil
	case operand.Ident != nil && operand.Ident.IsContextNowField():
		return nil, fmt.Errorf("cannot yet handle ctx field now")
	case operand.Type() == parser.TypeArray:
		return nil, fmt.Errorf("cannot yet handle operand of type non-literal array")
	default:
		return nil, fmt.Errorf("cannot handle operand of unknown type")

	}
}

// evaluateOperandCondition evaluates the condition by comparing the lhs and rhs operands using the given operator
func evaluateOperandCondition(
	lhs any,
	rhs any,
	operandType proto.Type,
	operator *parser.Operator,
) (bool, error) {
	// Evaluate when either operand or both are nil
	if lhs == nil && rhs == nil {
		return true && (operator.Symbol != parser.OperatorNotEquals), nil
	} else if lhs == nil || rhs == nil {
		return false || (operator.Symbol == parser.OperatorNotEquals), nil
	}

	// Evaluate with non-nil operands
	switch operandType {
	case proto.Type_TYPE_STRING:
		return compareString(lhs.(string), rhs.(string), operator)
	case proto.Type_TYPE_INT:
		return compareInt(lhs.(int64), rhs.(int64), operator)
	case proto.Type_TYPE_BOOL:
		return compareBool(lhs.(bool), rhs.(bool), operator)
	case proto.Type_TYPE_ENUM:
		return compareEnum(lhs.(string), rhs.(string), operator)
	case proto.Type_TYPE_IDENTITY:
		return compareIdentity(lhs.(ksuid.KSUID), rhs.(ksuid.KSUID), operator)
	default:
		return false, fmt.Errorf("cannot yet handle comparision of type: %s", operandType)
	}
}

func compareString(
	lhs string,
	rhs string,
	operator *parser.Operator,
) (bool, error) {
	switch operator.Symbol {
	case parser.OperatorEquals:
		return lhs == rhs, nil
	case parser.OperatorNotEquals:
		return lhs != rhs, nil
	default:
		return false, fmt.Errorf("operator: %s, not supported for type: %s", operator.Symbol, proto.Type_TYPE_STRING)
	}
}

func compareInt(
	lhs int64,
	rhs int64,
	operator *parser.Operator,
) (bool, error) {
	switch operator.Symbol {
	case parser.OperatorEquals:
		return lhs == rhs, nil
	case parser.OperatorNotEquals:
		return lhs != rhs, nil
	case parser.OperatorGreaterThan:
		return lhs > rhs, nil
	case parser.OperatorGreaterThanOrEqualTo:
		return lhs >= rhs, nil
	case parser.OperatorLessThan:
		return lhs < rhs, nil
	case parser.OperatorLessThanOrEqualTo:
		return lhs <= rhs, nil
	default:
		return false, fmt.Errorf("operator: %s, not supported for type: %s", operator.Symbol, proto.Type_TYPE_INT)
	}
}

func compareBool(
	lhs bool,
	rhs bool,
	operator *parser.Operator,
) (bool, error) {
	switch operator.Symbol {
	case parser.OperatorEquals:
		return lhs == rhs, nil
	case parser.OperatorNotEquals:
		return lhs != rhs, nil
	default:
		return false, fmt.Errorf("operator: %s, not supported for type: %s", operator.Symbol, proto.Type_TYPE_BOOL)
	}
}

func compareEnum(
	lhs string,
	rhs string,
	operator *parser.Operator,
) (bool, error) {
	switch operator.Symbol {
	case parser.OperatorEquals:
		return lhs == rhs, nil
	case parser.OperatorNotEquals:
		return lhs != rhs, nil
	default:
		return false, fmt.Errorf("operator: %s, not supported for type: %s", operator.Symbol, proto.Type_TYPE_STRING)
	}
}

func compareIdentity(
	lhs ksuid.KSUID,
	rhs ksuid.KSUID,
	operator *parser.Operator,
) (bool, error) {
	switch operator.Symbol {
	case parser.OperatorEquals:
		return lhs == rhs, nil
	case parser.OperatorNotEquals:
		return lhs != rhs, nil
	default:
		return false, fmt.Errorf("operator: %s, not supported for type: %s", operator.Symbol, proto.Type_TYPE_ID)
	}
}
