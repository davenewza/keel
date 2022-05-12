package schema

import (

	"github.com/teamkeel/keel/expressions"
	"github.com/teamkeel/keel/parser"
	"github.com/teamkeel/keel/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// makeProtoModels derives and returns a proto.Schema from the given (known to be valid) set of parsed AST.
func (scm *Schema) makeProtoModels(parserSchemas []*parser.Schema) *proto.Schema {
	protoSchema := &proto.Schema{}

	for _, parserSchema := range parserSchemas {
		for _, decl := range parserSchema.Declarations {
			switch {
			case decl.Model != nil:
				protoModel := scm.makeModel(decl)
				protoSchema.Models = append(protoSchema.Models, protoModel)
			case decl.Role != nil:
				// todo implement Role
			case decl.API != nil:
				// todo API not yet supported in proto
			default:
				panic("Case not recognized")
			}
		}
	}
	return protoSchema
}

func (scm *Schema) makeModel(decl *parser.Declaration) *proto.Model {
	parserModel := decl.Model
	protoModel := &proto.Model{
		Name: parserModel.Name,
	}
	for _, section := range parserModel.Sections {
		switch {

		case section.Fields != nil:
			protoModel.Fields = scm.makeFields(section.Fields, protoModel.Name)

		case section.Functions != nil:
			protoModel.Operations = scm.makeOperations(section.Functions, protoModel.Name, proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM)

		case section.Operations != nil:
			protoModel.Operations = scm.makeOperations(section.Operations, protoModel.Name, proto.OperationImplementation_OPERATION_IMPLEMENTATION_AUTO)

		case section.Attribute != nil:
			scm.applyModelAttribute(parserModel, protoModel, section.Attribute)
		default:
			panic("unrecognized case")
		}
	}

	return protoModel
}

func (scm *Schema) makeFields(parserFields []*parser.ModelField, modelName string) []*proto.Field {
	protoFields := []*proto.Field{}
	for _, parserField := range parserFields {
		protoField := scm.makeField(parserField, modelName)
		protoFields = append(protoFields, protoField)
	}
	return protoFields
}

func (scm *Schema) makeField(parserField *parser.ModelField, modelName string) *proto.Field {
	protoField := &proto.Field{
		ModelName: modelName,
		Name:      parserField.Name,
	}

	// We establish the field type when possible using the 1:1 mapping between parser enums
	// and proto enums. However, when the parsed field type is not one of the built in types, we
	// infer that it must refer to one of the Models defined in the schema, and is therefore of type
	// relationship.
	switch parserField.Type {
	case parser.FieldTypeBoolean:
		protoField.Type = proto.FieldType_FIELD_TYPE_BOOL
	case parser.FieldTypeText:
		protoField.Type = proto.FieldType_FIELD_TYPE_STRING
	case parser.FieldTypeCurrency:
		protoField.Type = proto.FieldType_FIELD_TYPE_CURRENCY
	case parser.FieldTypeDate:
		protoField.Type = proto.FieldType_FIELD_TYPE_DATE
	case parser.FieldTypeDatetime:
		protoField.Type = proto.FieldType_FIELD_TYPE_DATETIME
	case parser.FieldTypeEnum:
		protoField.Type = proto.FieldType_FIELD_TYPE_ENUM
	case parser.FieldTypeID:
		protoField.Type = proto.FieldType_FIELD_TYPE_ID
	case parser.FieldTypeImage:
		protoField.Type = proto.FieldType_FIELD_TYPE_IMAGE
	case parser.FieldTypeNumber:
		protoField.Type = proto.FieldType_FIELD_TYPE_INT
	case parser.FieldTypeIdentity:
		protoField.Type = proto.FieldType_FIELD_TYPE_IDENTITY
	default:
		protoField.Type = proto.FieldType_FIELD_TYPE_RELATIONSHIP
	}
	scm.applyFieldAttributes(parserField, protoField)
	return protoField
}

func (scm *Schema) makeOperations(parserFunctions []*parser.ModelAction, modelName string, impl proto.OperationImplementation) []*proto.Operation {
	protoOps := []*proto.Operation{}
	for _, parserFunc := range parserFunctions {
		protoOp := scm.makeOp(parserFunc, modelName, impl)
		protoOps = append(protoOps, protoOp)
	}
	return protoOps
}

func (scm *Schema) makeOp(parserFunction *parser.ModelAction, modelName string, impl proto.OperationImplementation) *proto.Operation {
	protoOp := &proto.Operation{
		ModelName:      modelName,
		Name:           parserFunction.Name,
		Implementation: impl,
		Type:           scm.mapToOperationType(parserFunction.Type),
	}
	protoOp.Inputs = scm.makeArguments(parserFunction, modelName)
	scm.applyFunctionAttributes(parserFunction, protoOp, modelName)

	return protoOp
}

func (scm *Schema) makeArguments(parserFunction *parser.ModelAction, modelName string) []*proto.OperationInput {
	// Currently, we only support arguments of the form <modelname>.
	operationInputs := []*proto.OperationInput{}
	for _, parserArg := range parserFunction.Arguments {
		operationInput := proto.OperationInput{
			Name:      parserArg.Name,
			Type:      proto.OperationInputType_OPERATION_INPUT_TYPE_FIELD,
			ModelName: wrapperspb.String(parserArg.Name),
			FieldName: wrapperspb.String(parserArg.Name),
		}
		operationInputs = append(operationInputs, &operationInput)
	}
	return operationInputs
}

func (scm *Schema) applyModelAttribute(parserModel *parser.Model, protoModel *proto.Model, attribute *parser.Attribute) {
	switch attribute.Name {
	case parser.AttributePermission:
		perm := scm.permissionAttributeToProtoPermission(attribute)
		perm.ModelName = protoModel.Name
		protoModel.Permissions = append(protoModel.Permissions, perm)
	}
}

func (scm *Schema) applyFunctionAttributes(parserFunction *parser.ModelAction, protoOperation *proto.Operation, modelName string) {
	for _, attribute := range parserFunction.Attributes {
		switch attribute.Name {
		case parser.AttributePermission:
			perm := scm.permissionAttributeToProtoPermission(attribute)
			perm.ModelName = modelName
			perm.OperationName = wrapperspb.String(protoOperation.Name)
			protoOperation.Permissions = append(protoOperation.Permissions, perm)
		case parser.AttributeWhere:
			// todo hope to remove error return from ToString
			expr, _ := expressions.ToString(attribute.Arguments[0].Expression)
			where := &proto.Expression{Source: expr}
			protoOperation.WhereExpressions = append(protoOperation.WhereExpressions, where)
		case parser.AttributeSet:
			// todo hope to remove error return from ToString
			expr, _ := expressions.ToString(attribute.Arguments[0].Expression)
			set := &proto.Expression{Source: expr}
			protoOperation.SetExpressions = append(protoOperation.SetExpressions, set)
		}
	}
}

func (scm *Schema) applyFieldAttributes(parserField *parser.ModelField, protoField *proto.Field) {
	for _, fieldAttribute := range parserField.Attributes {
		switch fieldAttribute.Name {
		case parser.AttributeUnique:
			protoField.Unique = true
		case parser.AttributeOptional:
			protoField.Optional = true
		}
	}
}

func (scm *Schema) permissionAttributeToProtoPermission(attr *parser.Attribute) *proto.PermissionRule {
	pr := &proto.PermissionRule{}
	for _, arg := range attr.Arguments {
		switch arg.Name {
		// todo use parser constants for "expression" etc below
		case "expression":
			expr, _ := expressions.ToString(arg.Expression)
			pr.Expression = &proto.Expression{Source: expr}
		case "role":
			value, _ := expressions.ToValue(arg.Expression)
			pr.RoleName = value.Ident[0]
		case "actions":
			value, _ := expressions.ToValue(arg.Expression)
			for _, v := range value.Array.Values {
				pr.OperationsTypes = append(pr.OperationsTypes, scm.mapToOperationType(v.Ident[0]))
			}
		}
	}
	return pr
}

func (scm *Schema) mapToOperationType(parsedOperation string) proto.OperationType {
	switch parsedOperation {
	case parser.ActionTypeCreate:
		return proto.OperationType_OPERATION_TYPE_CREATE
	case parser.ActionTypeUpdate:
		return proto.OperationType_OPERATION_TYPE_UPDATE
	case parser.ActionTypeGet:
		return proto.OperationType_OPERATION_TYPE_GET
	case parser.ActionTypeList:
		return proto.OperationType_OPERATION_TYPE_LIST
	default:
		return proto.OperationType_OPERATION_TYPE_UNKNOWN
	}
}
