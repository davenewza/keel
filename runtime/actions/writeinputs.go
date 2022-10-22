package actions

import (
	"github.com/iancoleman/strcase"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema/parser"
)

// DefaultCaptureSetValues updates the writeValues field in the given scope, with
// field/values that should be set for each of the given operation's Set expressions.
func DefaultCaptureSetValues(scope *Scope, args RequestArguments) error {
	ctx := scope.context

	for _, setExpression := range scope.operation.SetExpressions {
		expression, err := parser.ParseExpression(setExpression.Source)
		if err != nil {
			return err
		}

		assignment, err := expression.ToAssignmentCondition()
		if err != nil {
			return err
		}

		lhsOperandType, err := GetOperandType(assignment.LHS, scope.operation, scope.schema)
		if err != nil {
			return err
		}

		fieldName := assignment.LHS.Ident.Fragments[1].Fragment

		// todo: examine whole snake casing thing here
		scope.writeValues[strcase.ToSnake(fieldName)], err = evaluateOperandValue(ctx, assignment.RHS, scope.operation, scope.schema, args, lhsOperandType)
		if err != nil {
			return err
		}
	}
	return nil
}

// captureImplicitWriteInputValues updates the writeValues field in the
// given scope object with key/values that represent the implicit write-mode
// inputs carried by the given request.
func DefaultCaptureImplicitWriteInputValues(inputs []*proto.OperationInput, args RequestArguments, scope *Scope) error {
	for _, input := range inputs {
		if input.Behaviour != proto.InputBehaviour_INPUT_BEHAVIOUR_IMPLICIT {
			continue
		}

		if input.Mode != proto.InputMode_INPUT_MODE_WRITE {
			continue
		}

		fieldName := input.Target[0]
		value, ok := args[fieldName]

		if !ok {
			continue
		}

		// todo: examine whole snake casing thing here
		scope.writeValues[strcase.ToSnake(fieldName)] = value
	}
	return nil
}