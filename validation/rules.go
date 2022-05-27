package validation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/iancoleman/strcase"
	"github.com/teamkeel/keel/expressions"
	"github.com/teamkeel/keel/parser"
)

var (
	ReservedNames  = []string{"id", "createdAt", "updatedAt"}
	ReservedModels = []string{"query"}
)

// A Validator knows how to validate a parsed Keel schema.
//
// Conceptually we are validating a single schema.
// But the Validator supports it being "delivered" as a collection
// of *parser.Schema objects - to match up with a user's schema likely
// being written across N files.
//
// We use a []Input to model the inputs - so that the original file names are
// available for error reporting. (TODO although that is not implemented yet).
type Validator struct {
	inputs []Input
}

func NewValidator(inputs []Input) *Validator {
	return &Validator{
		inputs: inputs,
	}
}

func (v *Validator) RunAllValidators() error {
	validatorFuncs := []func([]Input) []error{
		noReservedFieldNames,
		noReservedModelNames,
		modelsUpperCamel,
		fieldsOpsFuncsLowerCamel,
		fieldNamesMustBeUniqueInAModel,
		operationsUniqueGlobally,
		operationFunctionInputs,
		operationUniqueFieldInput,
		supportedFieldTypes,
		supportedAttributeTypes,
		modelsGloballyUnique,
	}
	var errors []*ValidationError

	for _, vf := range validatorFuncs {
		err := vf(v.inputs)

		for _, e := range err {
			if verrs, ok := e.(*ValidationError); ok {
				errors = append(errors, verrs)
			}
		}
	}

	if len(errors) > 0 {
		errors := ValidationErrors{Errors: errors}
		return errors
	}

	return nil
}

// Models are UpperCamel
func modelsUpperCamel(inputs []Input) []error {
	var errors []error
	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, decl := range schema.Declarations {
			if decl.Model == nil {
				continue
			}
			// todo - these MustCompile regex would be better at module scope, to
			// make the MustCompile panic a load-time thing rather than a runtime thing.
			reg := regexp.MustCompile("([A-Z][a-z0-9]+)+")

			if reg.FindString(decl.Model.NameToken.Name) != decl.Model.NameToken.Name {
				suggested := strcase.ToCamel(strings.ToLower(decl.Model.NameToken.Name))

				errors = append(
					errors,
					validationError(
						ErrorUpperCamel,
						TemplateLiterals{
							Literals: map[string]string{
								"Model":     decl.Model.NameToken.Name,
								"Suggested": suggested,
							},
						},
						decl.Model.NameToken.Pos,
						decl.Model.NameToken.EndPos,
					),
				)
			}
		}
	}

	return errors
}

//Fields/operations are lowerCamel
func fieldsOpsFuncsLowerCamel(inputs []Input) []error {
	var errors []error
	for _, input := range inputs {
		schema := input.ParsedSchema

		for _, decl := range schema.Declarations {
			if decl.Model == nil {
				continue
			}
			for _, model := range decl.Model.Sections {
				for _, field := range model.Fields {
					if field.BuiltIn {
						continue
					}
					if strcase.ToLowerCamel(field.NameToken.Name) != field.NameToken.Name {
						errors = append(
							errors,
							validationError(ErrorFieldNameLowerCamel,
								TemplateLiterals{
									Literals: map[string]string{
										"Name":      field.NameToken.Name,
										"Suggested": strcase.ToLowerCamel(strings.ToLower(field.NameToken.Name)),
									},
								},
								field.NameToken.Pos,
								field.NameToken.EndPos,
							),
						)
					}
				}
				for _, operation := range model.Operations {
					if strcase.ToLowerCamel(operation.NameToken.Name) != operation.NameToken.Name {
						errors = append(
							errors,
							validationError(ErrorOperationNameLowerCamel,
								TemplateLiterals{
									Literals: map[string]string{
										"Name":      operation.NameToken.Name,
										"Suggested": strcase.ToLowerCamel(strings.ToLower(operation.NameToken.Name)),
									},
								},
								operation.NameToken.Pos,
								operation.NameToken.EndPos,
							),
						)
					}
				}

				for _, function := range model.Functions {
					if strcase.ToLowerCamel(function.NameToken.Name) != function.NameToken.Name {
						errors = append(
							errors,
							validationError(ErrorFunctionNameLowerCamel,
								TemplateLiterals{
									Literals: map[string]string{
										"Name":      function.NameToken.Name,
										"Suggested": strcase.ToLowerCamel(strings.ToLower(function.NameToken.Name)),
									},
								},
								function.NameToken.Pos,
								function.NameToken.EndPos,
							),
						)
					}
				}
			}
		}
	}

	return errors
}

//Field names must be unique in a model
func fieldNamesMustBeUniqueInAModel(inputs []Input) []error {
	var errors []error
	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, model := range schema.Declarations {
			if model.Model == nil {
				continue
			}
			for _, sections := range model.Model.Sections {
				fieldNames := map[string]bool{}
				for _, name := range sections.Fields {
					if _, ok := fieldNames[name.NameToken.Name]; ok {
						errors = append(
							errors,
							validationError(ErrorFieldNamesUniqueInModel,
								TemplateLiterals{
									Literals: map[string]string{
										"Name": name.NameToken.Name,
										"Line": fmt.Sprint(name.Pos.Line),
									},
								},
								name.NameToken.Pos,
								name.NameToken.EndPos,
							),
						)
					}
					fieldNames[name.NameToken.Name] = true
				}
			}
		}
	}
	return errors
}

type GlobalOperations struct {
	Name   string
	Model  string
	Pos    lexer.Position
	EndPos lexer.Position
}

func uniqueOperationsGlobally(inputs []Input) []GlobalOperations {
	var globalOperations []GlobalOperations
	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, declaration := range schema.Declarations {
			if declaration.Model == nil {
				continue
			}
			for _, sec := range declaration.Model.Sections {
				for _, operation := range sec.Operations {
					globalOperations = append(globalOperations, GlobalOperations{
						Name:   operation.NameToken.Name,
						Model:  declaration.Model.NameToken.Name,
						Pos:    operation.NameToken.Pos,
						EndPos: operation.NameToken.EndPos,
					})
				}
			}
		}
	}
	return globalOperations
}

//Operations must be globally unique
func operationsUniqueGlobally(inputs []Input) []error {
	var errors []error
	var operationNames []string

	globalOperations := uniqueOperationsGlobally(inputs)

	for _, name := range globalOperations {
		operationNames = append(operationNames, name.Name)
	}
	duplicates := findDuplicates(operationNames)

	if len(duplicates) == 0 {
		return nil
	}

	var duplicationOperations []GlobalOperations

	for _, operation := range globalOperations {
		for _, duplicate := range duplicates {
			if operation.Name == duplicate {
				duplicationOperations = append(duplicationOperations, operation)
			}
		}
	}

	seenOperations := map[string]bool{}

	for _, nameError := range duplicationOperations {
		key := fmt.Sprintf("%s-%s", nameError.Model, nameError.Name)

		if _, ok := seenOperations[key]; ok {
			errors = append(
				errors,
				validationError(ErrorOperationsUniqueGlobally,
					TemplateLiterals{
						Literals: map[string]string{
							"Model": nameError.Model,
							"Name":  nameError.Name,
							"Line":  fmt.Sprint(nameError.Pos.Line),
						},
					},
					nameError.Pos,
					nameError.EndPos,
				),
			)

			break
		}

		seenOperations[key] = true
	}

	return errors
}

type operationFunctionInputFields struct {
	Fields []*parser.ActionArg
	Pos    lexer.Position
}

//Inputs of operations/functions must be model fields
func operationFunctionInputs(inputs []Input) []error {
	var errors []error

	operationFields := make(map[string]*operationFunctionInputFields, 0)
	functionFields := make(map[string]*operationFunctionInputFields, 0)

	for _, input := range inputs {
		schema := input.ParsedSchema

		for _, declaration := range schema.Declarations {
			if declaration.Model == nil {
				continue
			}
			for _, modelSection := range declaration.Model.Sections {
				for _, operation := range modelSection.Operations {
					if len(operation.Arguments) == 0 {
						continue
					}
					operationFields[operation.NameToken.Name] = &operationFunctionInputFields{
						Fields: operation.Arguments,
						Pos:    operation.Pos,
					}
				}
			}

		}

		operationFields = findInvalidOpsFunctionInputs(inputs, operationFields)
	}
	for _, input := range inputs {
		schema := input.ParsedSchema

		for _, declaration := range schema.Declarations {
			if declaration.Model == nil {
				continue
			}
			for _, modelSection := range declaration.Model.Sections {
				for _, function := range modelSection.Functions {
					if len(function.Arguments) == 0 {
						continue
					}
					functionFields[function.NameToken.Name] = &operationFunctionInputFields{
						Fields: function.Arguments,
						Pos:    function.Pos,
					}
				}
			}

		}

		functionFields = findInvalidOpsFunctionInputs(inputs, functionFields)
	}

	errors = append(errors, buildErrorInvalidInputs(operationFields)...)
	errors = append(errors, buildErrorInvalidInputs(functionFields)...)

	return errors
}

//No reserved field names (id, createdAt, updatedAt)
func noReservedFieldNames(inputs []Input) []error {
	var errors []error
	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, name := range ReservedNames {
			for _, dec := range schema.Declarations {
				if dec.Model == nil {
					continue
				}
				for _, modelSection := range dec.Model.Sections {
					for _, field := range modelSection.Fields {
						if field.BuiltIn {
							continue
						}

						if strings.EqualFold(name, field.NameToken.Name) {
							errors = append(
								errors,
								validationError(ErrorReservedFieldName,
									TemplateLiterals{
										Literals: map[string]string{
											"Name":       field.NameToken.Name,
											"Suggestion": fmt.Sprintf("%ser", field.NameToken.Name),
										},
									},
									field.NameToken.Pos,
									field.NameToken.EndPos,
								),
							)
						}
					}
				}
			}
		}
	}

	return errors
}

// Check for reserved model names
func noReservedModelNames(inputs []Input) []error {
	var errors []error

	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, name := range ReservedModels {
			for _, dec := range schema.Declarations {
				if dec.Model == nil {
					continue
				}

				if strings.EqualFold(name, dec.Model.NameToken.Name) {
					errors = append(
						errors,
						validationError(ErrorReservedModelName,
							TemplateLiterals{
								Literals: map[string]string{
									"Name":       dec.Model.NameToken.Name,
									"Suggestion": fmt.Sprintf("%ser", dec.Model.NameToken.Name),
								},
							},
							dec.Model.NameToken.Pos,
							dec.Model.NameToken.EndPos,
						),
					)
				}
			}
		}
	}

	return errors
}

//GET operation must take a unique field as an input (or a unique combinations of inputs)
func operationUniqueFieldInput(inputs []Input) []error {
	var errors []error
	var fields []*parser.ModelField

	for _, input := range inputs {
		schema := input.ParsedSchema

		for _, dec := range schema.Declarations {
			if dec.Model == nil {
				continue
			}

			for _, section := range dec.Model.Sections {
				fields = append(fields, section.Fields...)
			}
		}
	}

	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, dec := range schema.Declarations {
			if dec.Model == nil {
				continue
			}

			for _, section := range dec.Model.Sections {
				if len(section.Operations) == 0 {
					continue
				}
				nonFieldAttrs := make(map[string]bool, 0)
				for _, operation := range section.Operations {
					nonFieldAttrs[operation.NameToken.Name] = false

					if operation.Type != parser.ActionTypeGet {
						continue
					}

					isValid := false

					for _, field := range fields {
						if len(operation.Arguments) != 1 && len(operation.Attributes) > 0 {
							validAttrs := checkAttributeExpressions(operation.Attributes, dec.Model.NameToken.Name, field)
							if validAttrs {
								nonFieldAttrs[operation.NameToken.Name] = true
								isValid = true
							}
						}

						if !nonFieldAttrs[operation.NameToken.Name] && len(operation.Arguments) != 1 {
							continue
						}

						if !nonFieldAttrs[operation.NameToken.Name] {
							isValid = checkFuncArgsUnique(operation, fields)
						}
					}

					if !isValid {
						errors = append(
							errors,
							validationError(ErrorOperationInputFieldNotUnique,
								TemplateLiterals{
									Literals: map[string]string{
										"Name": operation.NameToken.Name,
									},
								},
								operation.NameToken.Pos,
								operation.NameToken.EndPos,
							),
						)
					}
				}
			}
		}
	}

	return errors
}

func checkAttributeExpressions(input []*parser.Attribute, model string, field *parser.ModelField) bool {
	var isValid bool

	for _, attr := range input {
		for _, attrArg := range attr.Arguments {
			if len(field.Attributes) == 0 {
				continue
			}
			for _, at := range field.Attributes {
				if at.NameToken.Name != "unique" {
					continue
				}
				ok := expressions.IsAssignment(attrArg.Expression)
				if !ok {
					continue
				}
				if len(attrArg.Expression.Or) == 0 {
					continue
				}

				condition, err := expressions.ToAssignmentCondition(attrArg.Expression)
				if err != nil {
					continue
				}

				lhsOk := checkAssignmentFields(condition.LHS, model, field)
				if lhsOk {
					isValid = true
				}
				rhsOk := checkAssignmentFields(condition.RHS, model, field)
				if rhsOk {
					isValid = true
				}
			}
		}
	}

	return isValid
}

func checkAssignmentFields(indents *expressions.Value, model string, field *parser.ModelField) bool {
	if indents.Ident[0] != strings.ToLower(model) {
		return false
	}
	return indents.Ident[1] == field.NameToken.Name
}

func checkFuncArgsUnique(function *parser.ModelAction, fields []*parser.ModelField) bool {
	isValid := false
	arg := function.Arguments[0]

	for _, field := range fields {
		if field.NameToken.Name != arg.NameToken.Name {
			continue
		}

		for _, attr := range field.Attributes {
			if attr.NameToken.Name == "unique" {
				isValid = true
			}
			if attr.NameToken.Name == "primaryKey" {
				isValid = true
			}
		}
	}

	return isValid
}

//Supported field types
func supportedFieldTypes(inputs []Input) []error {
	var errors []error

	var fieldTypes = map[string]bool{"Text": true, "Date": true, "Timestamp": true, "Image": true, "Boolean": true, "Enum": true, "Identity": true, parser.FieldTypeID: true}

	for _, input := range inputs {
		schema := input.ParsedSchema

		// Append all model names to the supported types definition
		for _, dec := range schema.Declarations {
			if dec.Model != nil {
				fieldTypes[dec.Model.NameToken.Name] = true
			}
		}

		for _, dec := range schema.Declarations {
			if dec.Model == nil {
				continue
			}

			for _, section := range dec.Model.Sections {
				for _, field := range section.Fields {
					if _, ok := fieldTypes[field.Type]; !ok {
						availableTypes := []string{}

						for fieldType := range fieldTypes {
							if len(fieldType) > 0 {
								availableTypes = append(availableTypes, fieldType)
							}
						}

						// todo feed hint suggestions into validation error somehow.
						sort.Strings(availableTypes)

						hint := NewCorrectionHint(availableTypes, field.Type)

						suggestions := strings.Join(hint.Results, ", ")

						errors = append(
							errors,
							validationError(ErrorUnsupportedFieldType,
								TemplateLiterals{
									Literals: map[string]string{
										"Name":        field.NameToken.Name,
										"Type":        field.Type,
										"Suggestions": suggestions,
									},
								},
								field.NameToken.Pos,
								field.NameToken.EndPos,
							),
						)
					}
				}
			}
		}
	}

	return errors
}

func findModels(inputs []Input) []*parser.Model {
	models := []*parser.Model{}
	for _, input := range inputs {
		for _, decl := range input.ParsedSchema.Declarations {
			if decl.Model != nil {
				models = append(models, decl.Model)
			}
		}
	}
	return models
}

//Models are globally unique
func modelsGloballyUnique(inputs []Input) []error {
	var errors []error
	seenModelNames := map[string]bool{}

	for _, model := range findModels(inputs) {
		if _, ok := seenModelNames[model.NameToken.Name]; ok {
			errors = append(
				errors,
				validationError(ErrorUniqueModelsGlobally,
					TemplateLiterals{
						Literals: map[string]string{
							"Name": model.NameToken.Name,
						},
					},
					model.NameToken.Pos,
					model.NameToken.EndPos,
				),
			)

			continue
		}
		seenModelNames[model.NameToken.Name] = true
	}

	return errors
}

func supportedAttributeTypes(inputs []Input) []error {
	var errors []error

	for _, input := range inputs {
		schema := input.ParsedSchema

		for _, dec := range schema.Declarations {
			if dec.Model != nil {
				for _, section := range dec.Model.Sections {
					if section.Attribute != nil {
						errors = append(errors, checkAttributes([]*parser.Attribute{section.Attribute}, "model", dec.Model.NameToken.Name)...)
					}

					if section.Operations != nil {
						for _, op := range section.Operations {
							errors = append(errors, checkAttributes(op.Attributes, "operation", op.NameToken.Name)...)
						}
					}

					if section.Functions != nil {
						for _, function := range section.Functions {
							errors = append(errors, checkAttributes(function.Attributes, "function", function.NameToken.Name)...)
						}
					}

					if section.Fields != nil {
						for _, field := range section.Fields {
							errors = append(errors, checkAttributes(field.Attributes, "field", field.NameToken.Name)...)
						}
					}
				}
			}

			// Validate attributes defined within api sections
			if dec.API != nil {
				for _, section := range dec.API.Sections {
					if section.Attribute != nil {
						errors = append(errors, checkAttributes([]*parser.Attribute{section.Attribute}, "api", dec.API.NameToken.Name)...)
					}
				}
			}
		}
	}

	return errors
}

func checkAttributes(attributes []*parser.Attribute, definedOn string, parentName string) []error {
	var supportedAttributes = map[string][]string{
		parser.KeywordModel:     {parser.AttributePermission},
		parser.KeywordApi:       {parser.AttributeGraphQL},
		parser.KeywordField:     {parser.AttributeUnique, parser.AttributeOptional},
		parser.KeywordOperation: {parser.AttributeSet, parser.AttributeWhere, parser.AttributePermission},
		parser.KeywordFunction:  {parser.AttributePermission},
	}

	var builtIns = map[string][]string{
		parser.KeywordModel:     {},
		parser.KeywordApi:       {},
		parser.KeywordOperation: {},
		parser.KeywordFunction:  {},
		parser.KeywordField:     {parser.AttributePrimaryKey},
	}

	errors := make([]error, 0)

	for _, attr := range attributes {
		if contains(builtIns[definedOn], attr.NameToken.Name) {
			continue
		}

		if !contains(supportedAttributes[definedOn], attr.NameToken.Name) {
			hintOptions := supportedAttributes[definedOn]

			for i, hint := range hintOptions {
				hintOptions[i] = fmt.Sprintf("@%s", hint)
			}

			hint := NewCorrectionHint(hintOptions, attr.NameToken.Name)
			suggestions := strings.Join(hint.Results, ", ")

			errors = append(
				errors,
				validationError(ErrorUnsupportedAttributeType,
					TemplateLiterals{
						Literals: map[string]string{
							"Name":        fmt.Sprintf("@%s", attr.NameToken.EndPos),
							"ParentName":  parentName,
							"DefinedOn":   definedOn,
							"Suggestions": suggestions,
						},
					},
					attr.NameToken.Pos,
					attr.NameToken.EndPos,
				),
			)
		}
	}

	return errors
}

func findDuplicates(s []string) []string {
	inResult := make(map[string]bool)
	var result []string

	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
		} else {
			result = append(result, str)
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}

func buildErrorInvalidInputs(fields map[string]*operationFunctionInputFields) []error {
	var errors []error
	if len(fields) > 0 {
		for functionName, functionInput := range fields {
			for _, field := range functionInput.Fields {
				errors = append(
					errors,
					validationError(ErrorInputsNotFields,
						TemplateLiterals{
							Literals: map[string]string{
								"Model": functionName,
								"Field": field.NameToken.Name,
								"Line":  fmt.Sprint(field.Pos.Line),
							},
						},
						field.NameToken.Pos,
						field.NameToken.EndPos,
					),
				)
			}
		}
	}
	return errors
}

func findInvalidOpsFunctionInputs(inputs []Input, operationInput map[string]*operationFunctionInputFields) map[string]*operationFunctionInputFields {
	for _, input := range inputs {
		schema := input.ParsedSchema
		for _, input := range schema.Declarations {
			if input.Model == nil {
				continue
			}
			for _, modelName := range input.Model.Sections {
				for _, field := range modelName.Fields {
					for operationName, operationField := range operationInput {
						for _, operationFieldName := range operationField.Fields {
							if operationFieldName.NameToken.Name == field.NameToken.Name {
								delete(operationInput, operationName)
							}
						}
					}
				}
			}
		}
	}
	return operationInput
}
