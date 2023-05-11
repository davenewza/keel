package errorhandling

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/teamkeel/keel/schema/node"

	"gopkg.in/yaml.v3"
)

// error codes
const (
	ErrorUpperCamel                                 = "E001"
	ErrorActionNameLowerCamel                       = "E002"
	ErrorFieldNamesUniqueInModel                    = "E003"
	ErrorOperationsUniqueGlobally                   = "E004"
	ErrorInvalidActionInput                         = "E005"
	ErrorReservedFieldName                          = "E006"
	ErrorOperationMissingUniqueInput                = "E008"
	ErrorUnsupportedFieldType                       = "E009"
	ErrorUniqueModelsGlobally                       = "E010"
	ErrorUnsupportedAttributeType                   = "E011"
	ErrorFieldNameLowerCamel                        = "E012"
	ErrorInvalidAttributeArgument                   = "E013"
	ErrorAttributeRequiresNamedArguments            = "E014"
	ErrorAttributeMissingRequiredArgument           = "E015"
	ErrorInvalidValue                               = "E016"
	ErrorUniqueAPIGlobally                          = "E017"
	ErrorUniqueRoleGlobally                         = "E018"
	ErrorUniqueEnumGlobally                         = "E019"
	ErrorUnresolvableExpression                     = "E020"
	ErrorForbiddenExpressionOperation               = "E022"
	ErrorForbiddenValueCondition                    = "E023"
	ErrorIncorrectArguments                         = "E024"
	ErrorInvalidSyntax                              = "E025"
	ErrorExpressionTypeMismatch                     = "E026"
	ErrorForbiddenOperator                          = "E027"
	ErrorNonBooleanValueCondition                   = "E028"
	ErrorExpressionArrayMismatchingOperator         = "E030"
	ErrorExpressionForbiddenArrayLHS                = "E031"
	ErrorExpressionMixedTypesInArrayLiteral         = "E032"
	ErrorCreateOperationNoInputs                    = "E033"
	ErrorCreateOperationMissingInput                = "E034"
	ErrorOperationInputNotUnique                    = "E035"
	ErrorOperationWhereNotUnique                    = "E036"
	ErrorNonDirectComparisonOperatorUsed            = "E037"
	ErrorUnusedInput                                = "E038"
	ErrorInvalidOneToOneRelationship                = "E039"
	ErrorInvalidActionType                          = "E040"
	ErrorModelNotAllowedAsInput                     = "E041"
	ErrorReservedActionName                         = "E042"
	ErrorClashingImplicitInput                      = "E043"
	ErrorMissingRelationshipField                   = "E044"
	ErrorAmbiguousRelationship                      = "E045"
	ErrorCreateOperationMissingInputAliases         = "E046"
	ErrorModelNotFound                              = "E047"
	ErrorExpressionFieldTypeMismatch                = "E048"
	ErrorExpressionMultipleConditions               = "E049"
	ErrorDefaultExpressionNeeded                    = "E050"
	ErrorDefaultExpressionOperatorPresent           = "E051"
	ErrorRelationAttrOnWrongFieldType               = "E052"
	ErrorRelationAttrOnNonRepeatedField             = "E053"
	ErrorRelationAttributShouldBeIdentifier         = "E054"
	ErrorRelationAttributeUnrecognizedField         = "E055"
	ErrorRelationAttributeRelatedFieldWrongType     = "E056"
	ErrorRelationAttributeRelatedFieldIsNotRepeated = "E057"
	ErrorRelationAttributeRelatedFieldIsDuplicated  = "E058"
	ErrorCreateOperationAmbiguousRelationship       = "E059"
)

type ErrorDetails struct {
	Message string `json:"message" yaml:"message"`
	Hint    string `json:"hint" yaml:"hint"`
}

type TemplateLiterals struct {
	Literals map[string]string
}

type ValidationError struct {
	*ErrorDetails

	Code   string   `json:"code" regexp:"\\d+"`
	Pos    LexerPos `json:"pos,omitempty"`
	EndPos LexerPos `json:"endPos,omitempty"`
}

type LexerPos struct {
	Filename string `json:"filename"`
	Offset   int    `json:"offset"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s - on line: %v", e.Message, e.Pos.Line)
}

func (e *ValidationError) Unwrap() error { return e }

type ValidationErrors struct {
	Errors []*ValidationError `json:"errors"`
}

func (v *ValidationErrors) Append(code string, data map[string]string, node node.ParserNode) {
	v.Errors = append(v.Errors, NewValidationError(code,
		TemplateLiterals{
			Literals: data,
		},
		node,
	))
}

func (v *ValidationErrors) AppendError(e *ValidationError) {
	if e != nil {
		v.Errors = append(v.Errors, e)
	}
}

func (v *ValidationErrors) Concat(verrs ValidationErrors) {
	v.Errors = append(v.Errors, verrs.Errors...)
}

func (v ValidationErrors) Error() string {
	str := ""

	for _, err := range v.Errors {
		str += fmt.Sprintf("%s: %s\n", err.Code, err.Message)
	}

	return str
}

func (e ValidationErrors) Unwrap() error { return e }

type ErrorType string

const (
	NamingError              ErrorType = "NamingError"
	DuplicateDefinitionError ErrorType = "DuplicateDefinitionError"
	TypeError                ErrorType = "TypeError"
	UndefinedError           ErrorType = "UndefinedError"
	ActionInputError         ErrorType = "ActionInputError"
	AttributeArgumentError   ErrorType = "AttributeArgumentError"
	AttributeNotAllowedError ErrorType = "AttributeNotAllowedError"
	RelationshipError        ErrorType = "RelationshipError"
)

func NewValidationErrorWithDetails(t ErrorType, details ErrorDetails, position node.ParserNode) *ValidationError {
	start, end := position.GetPositionRange()

	return &ValidationError{
		Code:         string(t),
		ErrorDetails: &details,
		Pos: LexerPos{
			Filename: start.Filename,
			Offset:   start.Offset,
			Line:     start.Line,
			Column:   start.Column,
		},
		EndPos: LexerPos{
			Filename: end.Filename,
			Offset:   end.Offset,
			Line:     end.Line,
			Column:   end.Column,
		},
	}
}

func NewValidationError(code string, data TemplateLiterals, position node.ParserNode) *ValidationError {
	start, end := position.GetPositionRange()

	return &ValidationError{
		Code: code,
		// todo global locale setting
		ErrorDetails: buildErrorDetailsFromYaml(code, "en", data),
		Pos: LexerPos{
			Filename: start.Filename,
			Offset:   start.Offset,
			Line:     start.Line,
			Column:   start.Column,
		},
		EndPos: LexerPos{
			Filename: end.Filename,
			Offset:   end.Offset,
			Line:     end.Line,
			Column:   end.Column,
		},
	}
}

//go:embed errors.yml
var errorsYaml []byte

var errorDetailsByCode map[string]map[string]*ErrorDetails

func init() {
	err := yaml.Unmarshal(errorsYaml, &errorDetailsByCode)

	if err != nil {
		panic(err)
	}
}

func renderTemplate(name string, tmpl string, data map[string]string) string {
	template, err := template.New(name).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	err = template.Execute(&buf, data)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

// Takes an error code like E001, finds the relevant copy in the errors.yml file and interpolates the literals into the yaml template.
func buildErrorDetailsFromYaml(code string, locale string, literals TemplateLiterals) *ErrorDetails {
	ed, ok := errorDetailsByCode[locale][code]
	if !ok {
		panic(fmt.Sprintf("no error details for error code: %s", code))
	}

	return &ErrorDetails{
		Message: renderTemplate(fmt.Sprintf("%s-%s", code, "message"), ed.Message, literals.Literals),
		Hint:    renderTemplate(fmt.Sprintf("%s-%s", code, "hint"), ed.Hint, literals.Literals),
	}
}
