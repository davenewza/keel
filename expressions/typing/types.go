package typing

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/teamkeel/keel/schema/parser"
	"github.com/teamkeel/keel/schema/query"
)

type Ident []string

var (
	ID        = cel.OpaqueType(parser.FieldTypeID)
	Text      = cel.OpaqueType(parser.FieldTypeText)
	Markdown  = cel.OpaqueType(parser.FieldTypeMarkdown)
	Number    = cel.OpaqueType(parser.FieldTypeNumber)
	Decimal   = cel.OpaqueType(parser.FieldTypeDecimal)
	Boolean   = cel.OpaqueType(parser.FieldTypeBoolean)
	Timestamp = cel.OpaqueType(parser.FieldTypeTimestamp)
	Date      = cel.OpaqueType(parser.FieldTypeDate)
	Duration  = cel.OpaqueType(parser.FieldTypeDuration)
)
var (
	IDArray        = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeID))
	TextArray      = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeText))
	MarkdownArray  = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeMarkdown))
	NumberArray    = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeNumber))
	DecimalArray   = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeDecimal))
	BooleanArray   = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeBoolean))
	TimestampArray = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeTimestamp))
	DateArray      = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeDate))
	DurationArray  = cel.OpaqueType(fmt.Sprintf("%s[]", parser.FieldTypeDuration))
)

var (
	FunctionSum    = "SUM"
	FunctionCount  = "COUNT"
	FunctionAvg    = "AVG"
	FunctionMedian = "MEDIAN"
	FunctionMin    = "MIN"
	FunctionMax    = "MAX"
)

var (
	Role = cel.OpaqueType("_Role")
)

func MapType(schema []*parser.AST, typeName string, isRepeated bool) (*types.Type, error) {
	// For single operand conditions
	if typeName == parser.FieldTypeBoolean && !isRepeated {
		return types.BoolType, nil
	}

	switch typeName {
	case parser.FieldTypeID,
		parser.FieldTypeText,
		parser.FieldTypeMarkdown,
		parser.FieldTypeNumber,
		parser.FieldTypeBoolean,
		parser.FieldTypeDecimal,
		parser.FieldTypeTimestamp,
		parser.FieldTypeDate,
		parser.FieldTypeFile,
		parser.FieldTypeVector,
		parser.FieldTypeSecret,
		parser.FieldTypePassword,
		parser.FieldTypeDuration:
		if isRepeated {
			return cel.OpaqueType(fmt.Sprintf("%s[]", typeName)), nil
		} else {
			return cel.OpaqueType(typeName), nil
		}

	case Role.String(), "_ActionType", "_FieldName":
		if isRepeated {
			typeName = typeName + "[]"
		}
		return types.NewOpaqueType(typeName), nil
	}

	switch {
	case query.Enum(schema, typeName) != nil:
		if isRepeated {
			typeName = typeName + "[]"
		}
		return types.NewOpaqueType(typeName), nil
	case query.Model(schema, typeName) != nil:
		if isRepeated {
			typeName = typeName + "[]"
		}
		return types.NewObjectType(typeName), nil
	}

	return nil, fmt.Errorf("unknown type '%s'", typeName)
}
