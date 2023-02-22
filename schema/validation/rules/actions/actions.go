package actions

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/samber/lo"
	"github.com/teamkeel/keel/formatting"
	"github.com/teamkeel/keel/schema/parser"
	"github.com/teamkeel/keel/schema/query"
	"github.com/teamkeel/keel/schema/validation/errorhandling"
)

var (
	reservedActionNames = []string{
		parser.ImplicitAuthenticateOperationName,
	}
	validActionTypes = []string{
		parser.ActionTypeGet,
		parser.ActionTypeCreate,
		parser.ActionTypeUpdate,
		parser.ActionTypeList,
		parser.ActionTypeDelete,
	}
)

func ActionNamingRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, action := range query.ModelActions(model) {
			if strcase.ToLowerCamel(action.Name.Value) != action.Name.Value {
				errs.Append(errorhandling.ErrorActionNameLowerCamel,
					map[string]string{
						"Name":      action.Name.Value,
						"Suggested": strcase.ToLowerCamel(strings.ToLower(action.Name.Value)),
					},
					action.Name,
				)
			}
		}
	}

	return
}

// validate only read+write can be used with returns
// validate returns has to be specified with read+write
func ActionTypesRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, function := range query.ModelFunctions(model) {
			hasReturns := len(function.Returns) > 0
			validFunctionActionTypes := validActionTypes

			if hasReturns {
				validFunctionActionTypes = []string{parser.ActionTypeRead, parser.ActionTypeWrite}

				if function.Type.Value != parser.ActionTypeRead && function.Type.Value != parser.ActionTypeWrite {
					errs.AppendError(
						errorhandling.NewValidationErrorWithDetails(
							errorhandling.TypeError,
							errorhandling.ErrorDetails{
								Message: "The 'returns' keyword can only be used with 'read' or 'write' actions",
							},
							function.Type,
						),
					)
					continue
				}
			}

			if !hasReturns && (function.Type.Value == parser.ActionTypeRead || function.Type.Value == parser.ActionTypeWrite) {
				errs.AppendError(errorhandling.NewValidationErrorWithDetails(
					errorhandling.TypeError,
					errorhandling.ErrorDetails{
						Message: "The 'returns' keyword must be specified when using a 'read' or 'write' action type",
						Hint:    "Try to append 'returns(MyMessageType)'",
					},
					function.Node,
				))

				continue
			}

			// handles case where there is an unknown action type specified for a normal custom function
			if !lo.Contains(validFunctionActionTypes, function.Type.Value) {
				errs.AppendError(
					errorhandling.NewValidationErrorWithDetails(
						errorhandling.TypeError,
						errorhandling.ErrorDetails{
							Message: fmt.Sprintf("%s is not a valid action type. Valid types are %s", function.Type.Value, formatting.HumanizeList(validFunctionActionTypes, formatting.DelimiterOr)),
							Hint:    fmt.Sprintf("Valid types are %s", formatting.HumanizeList(validFunctionActionTypes, formatting.DelimiterOr)),
						},
						function.Type,
					),
				)
			}
		}

		for _, operation := range query.ModelOperations(model) {
			hasReturns := len(operation.Returns) > 0

			if hasReturns {
				errs.AppendError(errorhandling.NewValidationErrorWithDetails(
					errorhandling.TypeError,
					errorhandling.ErrorDetails{
						Message: "The 'returns' keyword is not valid in an operation",
						Hint:    fmt.Sprintf("Did you mean to create '%s' as a function?", operation.Name.Value),
					},
					operation.Returns[0].Node,
				))
			}

			if !lo.Contains(validActionTypes, operation.Type.Value) {
				errs.AppendError(
					errorhandling.NewValidationErrorWithDetails(
						errorhandling.TypeError,
						errorhandling.ErrorDetails{
							Message: fmt.Sprintf("%s is not a valid action type. Valid types are %s", operation.Type.Value, formatting.HumanizeList(validActionTypes, formatting.DelimiterOr)),
							Hint:    fmt.Sprintf("Valid types are %s", formatting.HumanizeList(validActionTypes, formatting.DelimiterOr)),
						},
						operation.Type,
					),
				)
			}
		}
	}

	return
}

func UniqueOperationNamesRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	operationNames := map[string]bool{}

	for _, model := range query.Models(asts) {
		for _, action := range query.ModelActions(model) {
			if _, ok := operationNames[action.Name.Value]; ok {
				errs.Append(errorhandling.ErrorOperationsUniqueGlobally,
					map[string]string{
						"Model": model.Name.Value,
						"Name":  action.Name.Value,
						"Line":  fmt.Sprint(action.Pos.Line),
					},
					action.Name,
				)
			}
			operationNames[action.Name.Value] = true
		}
	}

	return
}

// ReservedActionNameRule ensures that all actions (operations or functions) do not
// use a reserved name
func ReservedActionNameRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, op := range query.ModelActions(model) {
			if lo.Contains(reservedActionNames, op.Name.Value) {
				errs.Append(errorhandling.ErrorReservedActionName,
					map[string]string{
						"Name":       op.Name.Value,
						"Suggestion": fmt.Sprintf("perform%s", strcase.ToCamel(op.Name.Value)),
					},
					op.Name,
				)
			}
		}
	}

	return errs
}

// CreateOperationRequiredFieldsRule validates that a create operation is specified in such a way
// that all the fields that must be set, are covered by either inputs or set expressions.
func CreateOperationRequiredFieldsRule(
	asts []*parser.AST) (errs errorhandling.ValidationErrors) {

	for _, model := range query.Models(asts) {
		requiredFieldsWithAliases := requiredCreateFields(model)
		createActions := query.ModelCreateOperations(model)
		for _, createAction := range createActions {
			for _, fld := range requiredFieldsWithAliases {
				satisfiedByWithInput := requiredFieldInWithClause(fld.AllowedInputNames, createAction)
				satisfiedBySetExpr := satisfiedBySetExpr(fld.AllowedSetExprNames, model.Name.Value, createAction)

				// If the missing field has aliases, we use an error format dedicated to that case.
				if !satisfiedByWithInput && !satisfiedBySetExpr {
					switch {
					case len(fld.AllowedInputNames) > 1:
						errs.Append(errorhandling.ErrorCreateOperationMissingInputAliases,
							map[string]string{
								"WithNames": formatting.HumanizeList(fld.AllowedInputNames, ""),
								"SetNames":  formatting.HumanizeList(fld.AllowedSetExprNames, ""),
							},
							createAction.Name,
						)
					default:
						// The more general case.
						errs.Append(errorhandling.ErrorCreateOperationMissingInput,
							map[string]string{
								"FieldName": fld.AllowedInputNames[0],
							},
							createAction.Name,
						)
					}
				}
			}
		}
	}
	return errs
}

// RequiredField specifies a field name that is "required", and gives you a set of
// alternative names by which the field may be referred to in either an operation input,
// or in an operation assignment expression.
// It is capable of modelling the following aliases for a foreign key field, such as
//
//	[Author, authorId, author.id]
//
// Or for more general fields with no additional alias names:
//
//	[Age]
type requiredField struct {
	AllowedInputNames   []string
	AllowedSetExprNames []string
}

// setExpressions returns all the non-nil expressions from all
// the @set attributes on the given action.
func setExpressions(action *parser.ActionNode) []*parser.Expression {
	setters := lo.Filter(action.Attributes, func(a *parser.AttributeNode, _ int) bool {
		return a.Name.Value == parser.AttributeSet
	})
	expressions := []*parser.Expression{}
	for _, setAttr := range setters {
		if len(setAttr.Arguments) == 0 {
			continue
		}
		if setAttr.Arguments[0].Expression != nil {
			expressions = append(expressions, setAttr.Arguments[0].Expression)
		}
	}
	return expressions
}

// requiredCreateFields works out which of the fields on the given model,
// must be specified for any create action on that model to be valid.
func requiredCreateFields(model *parser.ModelNode) []*requiredField {
	req := []*requiredField{}

	for _, f := range query.ModelFields(model) {
		if f.Optional {
			continue
		}
		if f.Repeated {
			continue
		}
		if query.FieldHasAttribute(f, parser.AttributeDefault) {
			continue
		}
		// The model fields associated with foreign keys fields are not required
		// because instead the auto-generated "authorId" field is - which is caught below.
		if f.FkInfo != nil && f.FkInfo.OwningField == f {
			continue
		}

		// We conclude this field IS required.

		if f.FkInfo != nil && f.FkInfo.ForeignKeyField == f {
			// A required FK field can be satisfied by alternative (alias) names like
			// 		"author", "authorId", "author.id"
			dottedForm := strings.Join([]string{
				f.FkInfo.OwningField.Name.Value,
				f.FkInfo.ReferredToModelPrimaryKey.Name.Value}, ".")
			requiredField := requiredField{
				AllowedInputNames:   []string{f.Name.Value, dottedForm},
				AllowedSetExprNames: []string{f.Name.Value, f.FkInfo.OwningField.Name.Value},
			}
			req = append(req, &requiredField)
		} else {
			// The general case
			req = append(req, &requiredField{
				AllowedInputNames:   []string{f.Name.Value},
				AllowedSetExprNames: []string{f.Name.Value},
			})
		}
	}
	return req
}

// requiredFieldInWithClause returns true if any of the given names/aliases are
// present the the given action's "With" inputs.
func requiredFieldInWithClause(fieldAliases []string, action *parser.ActionNode) bool {
	for _, altFieldName := range fieldAliases {
		for _, input := range action.With {
			if input.Label == nil && input.Type.ToString() == altFieldName {
				return true
			}
		}
	}
	return false
}

// satisfiedBySetExpr returns true if any of the given names/aliases are
// present in any of the given action's @set expressions as the LHS of an assignment.
// E.g
// @set(mymodel.age =
// @set(mymodel.authorId =
func satisfiedBySetExpr(fieldAliases []string, modelName string, action *parser.ActionNode) bool {
	setExpressions := setExpressions(action)
	for _, expr := range setExpressions {
		assignment, err := expr.ToAssignmentCondition()
		if err != nil {
			continue
		}
		lhs := assignment.LHS

		if len(lhs.Ident.Fragments) != 2 {
			continue
		}
		modelName, fieldName := lhs.Ident.Fragments[0].Fragment, lhs.Ident.Fragments[1].Fragment
		if modelName != strcase.ToLowerCamel(modelName) {
			continue
		}

		for _, altFieldName := range fieldAliases {
			if fieldName == altFieldName {
				return true
			}
		}
	}
	return false
}

// UpdateOperationUniqueConstraintRule checks that all update operations
// are filtering on unique fields only
func UpdateOperationUniqueConstraintRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {

	for _, model := range query.Models(asts) {
		// Note - this is applied only to Operations, i.e. not Function.
		for _, action := range query.ModelOperations(model) {
			if action.Type.Value != parser.ActionTypeUpdate {
				continue
			}
			errs.Concat(requireUniqueLookup(asts, action, model))
		}
	}

	return
}

func ListActionModelInputsRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, action := range query.ModelActions(model) {
			if action.Type.Value != parser.ActionTypeList {
				continue
			}

			for _, input := range action.Inputs {
				resolvedType := query.ResolveInputType(asts, input, model)
				if resolvedType == "" {
					continue
				}

				m := query.Model(asts, resolvedType)
				if m == nil {
					continue
				}

				// error - cannot use a model field as an input to a list action
				errs.Append(errorhandling.ErrorModelNotAllowedAsInput,
					map[string]string{
						"Input":      input.Type.ToString(),
						"ActionType": action.Type.Value,
						"ModelName":  m.Name.Value,
					},
					input.Type,
				)
			}
		}
	}

	return
}

// GetOperationUniqueConstraintRule checks that all get operations
// are filtering on unique fields only
func GetOperationUniqueConstraintRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {

	for _, model := range query.Models(asts) {
		// Note - this is applied only to Operations, i.e. not Functions.
		for _, action := range query.ModelOperations(model) {
			if action.Type.Value != parser.ActionTypeGet {
				continue
			}

			errs.Concat(requireUniqueLookup(asts, action, model))
		}
	}

	return
}

// DeleteOperationUniqueConstraintRule checks that all get operations
// are filtering on unique fields only
func DeleteOperationUniqueConstraintRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {

	for _, model := range query.Models(asts) {
		// Note - this is applied only to Operations, i.e. not Functions.
		for _, action := range query.ModelOperations(model) {
			if action.Type.Value != parser.ActionTypeDelete {
				continue
			}

			errs.Concat(requireUniqueLookup(asts, action, model))
		}
	}

	return
}

func requireUniqueLookup(asts []*parser.AST, action *parser.ActionNode, model *parser.ModelNode) (errs errorhandling.ValidationErrors) {

	hasUniqueLookup := false

	// check for inputs that refer to non-unique fields
	for _, arg := range action.Inputs {
		isUnique, err := validateInputIsUnique(asts, action, arg, model)
		if err != nil {
			errs.AppendError(err)
		}
		if isUnique {
			hasUniqueLookup = true
		}
	}

	// check for @where attributes that filter on non-unique fields
	// only when the inputs are non-unique
	if !hasUniqueLookup {
		for _, attr := range action.Attributes {

			if attr.Name.Value != parser.AttributeWhere {
				continue
			}

			if len(attr.Arguments) == 0 {
				continue
			}

			if attr.Arguments[0].Expression == nil {
				continue
			}

			conds := attr.Arguments[0].Expression.Conditions()

			for _, condition := range conds {
				// If it's not a logical condition it will be caught by the
				// @where attribute validation
				if condition.Type() != parser.LogicalCondition {
					continue
				}

				operator := condition.Operator.Symbol

				// Only "==" and "in" are direct comparison operators, anything else
				// doesn't make sense for a unique lookup e.g. age > 5
				if operator != parser.OperatorEquals && operator != parser.OperatorIn {
					errs.Append(errorhandling.ErrorNonDirectComparisonOperatorUsed,
						map[string]string{
							"Operator":      operator,
							"OperationType": action.Type.Value,
						},
						condition.Operator,
					)
					continue
				}

				// we always check the LHS
				operands := []*parser.Operand{condition.LHS}

				// if it's an equal operator we can check both sides
				if operator == parser.OperatorEquals {
					operands = append(operands, condition.RHS)
				}

				for _, op := range operands {
					if op.Ident == nil || len(op.Ident.Fragments) != 2 {
						continue
					}

					modelName, fieldName := op.Ident.Fragments[0].Fragment, op.Ident.Fragments[1].Fragment

					if modelName != strcase.ToLowerCamel(model.Name.Value) {
						continue
					}

					field := query.ModelField(model, fieldName)
					if field == nil {
						continue
					}

					// we've found a @where that is filtering on a unique
					// field using a direct comparison operator
					if query.FieldIsUnique(field) {
						hasUniqueLookup = true
						continue
					}

					// @where attribute that has a condition on a non-unique field
					// this is an error
					errs.Append(errorhandling.ErrorOperationWhereNotUnique,
						map[string]string{
							"Ident":         op.Ident.ToString(),
							"OperationType": action.Type.Value,
						},
						op.Ident,
					)
				}
			}
		}
	}

	// If a unique lookup was found, then drop all errors found for any
	// non-unique lookups found
	if hasUniqueLookup {
		errs = errorhandling.ValidationErrors{}
	}

	// If we did not find a unique field make sure there is an error on the
	// action. This might happen if the action is defined with no inputs or
	// @where clauses e.g. `get getMyThing()`
	if !hasUniqueLookup && len(errs.Errors) == 0 {
		errs.Append(errorhandling.ErrorOperationMissingUniqueInput,
			map[string]string{
				"Name": action.Name.Value,
			},
			action.Name,
		)
	}

	return
}

func ValidActionInputTypesRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, action := range query.ModelOperations(model) {
			isFunction := false
			for _, input := range action.Inputs {
				errs.AppendError(validateInputType(isFunction, asts, input, model, action))
			}

			for _, input := range action.With {
				errs.AppendError(validateInputType(isFunction, asts, input, model, action))
			}
		}

		for _, action := range query.ModelFunctions(model) {
			isFunction := true
			for _, input := range action.Inputs {
				errs.AppendError(validateInputType(isFunction, asts, input, model, action))
			}

			for _, input := range action.With {
				errs.AppendError(validateInputType(isFunction, asts, input, model, action))
			}
		}
	}

	return
}

func ValidOperationInputUsagesRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, operation := range query.ModelOperations(model) {
			isFunction := false
			for _, input := range operation.Inputs {
				errs.AppendError(validateInputUsage(isFunction, asts, input, model, operation))
			}

			for _, input := range operation.With {
				errs.AppendError(validateInputUsage(isFunction, asts, input, model, operation))
			}
		}
	}

	return errs
}

// Validate that inputs with labels are used somewhere (within an expression in any of the child attributes)
func validateInputUsage(
	isFunction bool,
	asts []*parser.AST,
	input *parser.ActionInputNode,
	model *parser.ModelNode,
	action *parser.ActionNode,
) *errorhandling.ValidationError {
	if input.Label == nil {
		return nil
	}

	isUsed := false

	fieldsAssignedWithExplicitInput := []string{}

	for _, attr := range action.Attributes {
		if !lo.Contains([]string{parser.AttributeWhere, parser.AttributeSet, parser.AttributePermission, parser.AttributeValidate}, attr.Name.Value) {
			continue
		}

		if len(attr.Arguments) == 0 {
			continue
		}

		expr := attr.Arguments[0].Expression
		if expr == nil {
			continue
		}

		for _, cond := range expr.Conditions() {
			for _, operand := range []*parser.Operand{cond.LHS, cond.RHS} {
				if operand.Ident != nil && operand.ToString() == input.Label.Value {
					// we've found a usage of the input
					isUsed = true

					if cond.LHS != nil && cond.LHS != operand && cond.LHS.Ident != nil {
						fieldsAssignedWithExplicitInput = append(fieldsAssignedWithExplicitInput, cond.LHS.Ident.LastFragment())
						continue
					}

					if cond.RHS != nil && cond.RHS != operand && cond.RHS.Ident != nil {
						fieldsAssignedWithExplicitInput = append(fieldsAssignedWithExplicitInput, cond.RHS.Ident.LastFragment())
					}
				}
			}
		}
	}

	if isUsed {
		// check explicit input doesn't clash with an implicit input already defined in the inputs list
		allInputs := append(action.Inputs, action.With...)
		implicitInputs := lo.Filter(allInputs, func(input *parser.ActionInputNode, _ int) bool {
			// inputs without a label are deemed to be implicit
			return input.Label == nil
		})

		for _, input := range implicitInputs {
			if lo.Contains(fieldsAssignedWithExplicitInput, input.Name()) {
				return errorhandling.NewValidationError(
					errorhandling.ErrorClashingImplicitInput,
					errorhandling.TemplateLiterals{
						Literals: map[string]string{
							"ImplicitInputName": input.Name(),
						},
					},
					input,
				)
			}

		}

		return nil
	}

	// No usages of the input - report error
	return errorhandling.NewValidationError(
		errorhandling.ErrorUnusedInput,
		errorhandling.TemplateLiterals{
			Literals: map[string]string{
				"InputName": input.Label.Value,
			},
		},
		input.Label,
	)
}

// validateInputType validates that an input's type is acceptable, according to the following three rules:
// 1. input.Label is nil and input.Type matches name of a field on the current model (or nested field)
// 2. input.Label is not nil and input.Type.Fragments[0] is a valid built-in type or enum
// 3. isFunction is true, input.Label is nil, input.Type.Fragments[0] is a message name, and action.Inputs has a length of 1
func validateInputType(
	isFunction bool,
	asts []*parser.AST,
	input *parser.ActionInputNode,
	model *parser.ModelNode,
	action *parser.ActionNode) *errorhandling.ValidationError {
	resolvedType := query.ResolveInputType(asts, input, model)
	msg := query.Message(asts, input.Type.ToString())

	// validate that labels are lower camel cased to avoid ambiguity with message types
	if input.Label != nil && strcase.ToLowerCamel(input.Label.Value) != input.Label.Value {
		return errorhandling.NewValidationErrorWithDetails(
			errorhandling.ActionInputError,
			errorhandling.ErrorDetails{
				Message: fmt.Sprintf("Input label '%s' must be in lower camel case", input.Label.Value),
				Hint:    fmt.Sprintf("Try '%s' instead", strcase.ToLowerCamel(input.Label.Value)),
			},
			input.Label,
		)
	}
	// if not explicitly labelled then we don't need to check for the input being used
	// as inputs using short-hand syntax are implicitly use
	// For functions the input doesn't need to be used
	if resolvedType != "" && (isFunction || input.Label == nil) {
		return nil
	}

	if msg != nil && !isFunction {
		// not allowed a message in an operation (return)
		return errorhandling.NewValidationErrorWithDetails(
			errorhandling.ActionInputError,
			errorhandling.ErrorDetails{
				Message: fmt.Sprintf("You cannot use message %s in an operation", msg.Name.Value),
				Hint:    "Messages can only be used in functions",
			},
			input.Node,
		)
	}

	if msg != nil {
		if action.Type.Value != parser.ActionTypeRead && action.Type.Value != parser.ActionTypeWrite {
			return errorhandling.NewValidationErrorWithDetails(
				errorhandling.ActionInputError,
				errorhandling.ErrorDetails{
					Message: "You can only use messages in a read/write function",
					Hint:    "Messages can only be used in read/write functions",
				},
				action.Node,
			)
		}

		if len(action.Inputs) > 0 && action.Inputs[0] == input {
			// all good - message being used as input to function
			return nil
		}

		// todo: move the below to separate rule

		// not allowed to have more than one message (return)
		// example of error is "ThisIsNot" message here
		// create createFoo(ThisIsValid, ThisIsNot)

		return errorhandling.NewValidationErrorWithDetails(
			errorhandling.ActionInputError,
			errorhandling.ErrorDetails{
				Message: "You cannot have multiple message-based inputs",
			},
			action.Inputs[1],
		)
	}

	// If type cannot be resolved report error
	if resolvedType == "" {
		types := []string{}
		for _, field := range query.ModelFields(model) {
			types = append(types, field.Name.Value)
		}

		types = append(types, query.MessageNames(asts)...)

		// todo:
		// if there is no label, suggest model field names
		// if there is no label and only first input and isFunction, suggest message types
		// if there is a label, then suggest built ins

		hint := errorhandling.NewCorrectionHint(types, input.Type.ToString())

		return errorhandling.NewValidationError(
			errorhandling.ErrorInvalidActionInput,
			errorhandling.TemplateLiterals{
				Literals: map[string]string{
					"Input":     input.Type.ToString(),
					"Suggested": hint.ToString(),
				},
			},
			input.Type,
		)
	}

	return nil
}

// CreateOperationNoReadInputsRule validates that create actions don't accept
// any read-only inputs
func CreateOperationNoReadInputsRule(asts []*parser.AST) (errs errorhandling.ValidationErrors) {
	for _, model := range query.Models(asts) {
		for _, action := range query.ModelActions(model) {
			if action.Type.Value != parser.ActionTypeCreate {
				continue
			}

			if len(action.Inputs) == 0 {
				continue
			}

			for _, i := range action.Inputs {
				var name string
				if i.Label != nil {
					name = i.Label.Value
				} else {
					name = i.Type.ToString()
				}
				errs.Append(errorhandling.ErrorCreateOperationNoInputs,
					map[string]string{
						"Input": name,
					},
					i,
				)
			}
		}
	}

	return
}

func validateInputIsUnique(asts []*parser.AST, action *parser.ActionNode, input *parser.ActionInputNode, model *parser.ModelNode) (isUnique bool, err *errorhandling.ValidationError) {
	// handle built-in type e.g. not referencing a field name
	// for example `get getMyThing(name: Text)`
	if parser.IsBuiltInFieldType(input.Type.ToString()) {
		return false, nil
	}

	var field *parser.FieldNode

	for _, fragment := range input.Type.Fragments {
		if model == nil {
			return false, nil
		}
		field = query.ModelField(model, fragment.Fragment)
		if field == nil {
			return false, nil
		}
		if !query.FieldIsUnique(field) {
			// input refers to a non-unique field - this is an error
			return false, errorhandling.NewValidationError(errorhandling.ErrorOperationInputNotUnique,
				errorhandling.TemplateLiterals{
					Literals: map[string]string{
						"Input":         fragment.Fragment,
						"OperationType": action.Type.Value,
					},
				},
				fragment,
			)
		}
		model = query.Model(asts, field.Type)
	}

	// If we have a model at the end of this it means that the input
	// is referring to the "bare" model and not a specific field of that
	// model. This is an error for unique inputs.
	if model != nil {
		// input refers to a non-unique field - this is an error
		return false, errorhandling.NewValidationError(errorhandling.ErrorModelNotAllowedAsInput,
			errorhandling.TemplateLiterals{
				Literals: map[string]string{
					"ActionType": action.Type.Value,
					"Input":      input.Type.ToString(),
					"ModelName":  model.Name.Value,
				},
			},
			input,
		)
	}

	return true, nil
}
