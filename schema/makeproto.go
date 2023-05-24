package schema

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/casing"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/schema/parser"
	"github.com/teamkeel/keel/schema/query"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// makeProtoModels derives and returns a proto.Schema from the given (known to be valid) set of parsed AST.
func (scm *Builder) makeProtoModels() *proto.Schema {
	scm.proto = &proto.Schema{}

	// makeAnyType adds a global 'Any' type to the messages registry which is useful for those who want untyped inputs and responses for arbitrary functions
	scm.makeAnyType()

	// Add any messages defined declaratively in the schema to the registry of message types
	for _, ast := range scm.asts {
		for _, d := range ast.Declarations {
			if d.Message != nil {
				scm.makeMessage(d)
			}
		}
	}

	for _, parserSchema := range scm.asts {
		for _, decl := range parserSchema.Declarations {
			switch {
			case decl.Model != nil:
				scm.makeModel(decl)
			case decl.Role != nil:
				scm.makeRole(decl)
			case decl.API != nil:
				scm.makeAPI(decl)
			case decl.Enum != nil:
				scm.makeEnum(decl)
			case decl.Message != nil:
				// noop
			default:
				panic("Case not recognized")
			}
		}

		for _, envVar := range parserSchema.EnvironmentVariables {
			scm.proto.EnvironmentVariables = append(scm.proto.EnvironmentVariables, &proto.EnvironmentVariable{
				Name: envVar,
			})
		}

		for _, secret := range parserSchema.Secrets {
			scm.proto.Secrets = append(scm.proto.Secrets, &proto.Secret{
				Name: secret,
			})
		}
	}

	return scm.proto
}

func makeListQueryInputMessage(typeInfo *proto.TypeInfo, nullable bool) (*proto.Message, error) {
	var message *proto.Message
	switch typeInfo.Type {
	case proto.Type_TYPE_ID:
		message = &proto.Message{Name: makeInputMessageName("IDQuery"), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "oneOf",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					Repeated: true,
				},
			},
			{
				Name:     "notEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
		}}
	case proto.Type_TYPE_STRING:
		message = &proto.Message{Name: makeInputMessageName("StringQuery"), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "notEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "startsWith",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "endsWith",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "contains",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "oneOf",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					Repeated: true,
				},
			},
		}}
	case proto.Type_TYPE_INT:
		message = &proto.Message{Name: makeInputMessageName("IntQuery"), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "notEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "lessThan",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "lessThanOrEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "greaterThan",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "greaterThanOrEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "oneOf",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					Repeated: true,
				},
			},
		}}
	case proto.Type_TYPE_BOOL:
		message = &proto.Message{Name: makeInputMessageName("BooleanQuery"), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
		}}
	case proto.Type_TYPE_DATE:
		message = &proto.Message{Name: makeInputMessageName("DateQuery"), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "before",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "onOrBefore",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "after",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "onOrAfter",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
		}}
	case proto.Type_TYPE_DATETIME, proto.Type_TYPE_TIMESTAMP:
		message = &proto.Message{Name: makeInputMessageName("TimestampQuery"), Fields: []*proto.MessageField{
			{
				Name:     "before",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
			{
				Name:     "after",
				Optional: true,
				Type: &proto.TypeInfo{
					Type: typeInfo.Type,
				},
			},
		}}
	case proto.Type_TYPE_ENUM:
		message = &proto.Message{Name: makeInputMessageName(fmt.Sprintf("%sQuery", typeInfo.EnumName.Value)), Fields: []*proto.MessageField{
			{
				Name:     "equals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					EnumName: typeInfo.EnumName,
				},
			},
			{
				Name:     "notEquals",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					EnumName: typeInfo.EnumName,
				},
			},
			{
				Name:     "oneOf",
				Optional: true,
				Type: &proto.TypeInfo{
					Type:     typeInfo.Type,
					EnumName: typeInfo.EnumName,
					Repeated: true,
				},
			},
		}}
	default:
		return nil, fmt.Errorf("unsupported query type %s", typeInfo.Type.String())
	}

	// If the field is optional, then we need to return a Nullable query type,
	// which means we add the { isNull: true } query option.
	if nullable {
		nullableField := proto.MessageField{
			Name:     "isNull",
			Optional: true,
			Type: &proto.TypeInfo{
				Type: proto.Type_TYPE_BOOL,
			},
		}

		message.Name = fmt.Sprintf("Nullable%s", message.Name)
		message.Fields = append(message.Fields, &nullableField)
	}

	return message, nil
}

// Creates a proto.Message from a slice of action inputs.
func (scm *Builder) makeMessageFromActionInputNodes(name string, inputs []*parser.ActionInputNode, model *parser.ModelNode, action *parser.ActionNode, impl proto.OperationImplementation) *proto.Message {
	fields := []*proto.MessageField{}
	for _, input := range inputs {
		typeInfo, target, targetsOptionalField := scm.inferParserInputType(model, action, input, impl)

		fields = append(fields, &proto.MessageField{
			Name:        input.Name(),
			Type:        typeInfo,
			Target:      target,
			Optional:    input.Optional || targetsOptionalField,
			MessageName: name,
		})
	}

	return &proto.Message{
		Name:   name,
		Fields: fields,
	}
}

// Creates the message structure from an implicit input. For relationships, this will create a nested hierarchy of messages.
func (scm *Builder) makeMessageHierarchyFromImplicitInput(rootMessage *proto.Message, input *parser.ActionInputNode, model *parser.ModelNode, action *parser.ActionNode, impl proto.OperationImplementation) {
	target := lo.Map(input.Type.Fragments, func(ident *parser.IdentFragment, _ int) string {
		return ident.Fragment
	})

	currMessage := rootMessage
	currModel := model.Name.Value

	for currIndex, fragment := range target {
		if currIndex < len(target)-1 {
			// If this is not the last target fragment, then we know the current fragment is referring to a related model field.
			// Therefore, we must create a new message for this related model and add it to the current message as a field (if this hasn't already been done with a previous input).

			// Message name of nested message appended with the target framements. E.g. CreateSaleItemsInput
			relatedModelMessageName := makeInputMessageName(action.Name.Value, target[0:currIndex+1]...)

			// Does the field already exist from a previous input?
			fieldAlreadyCreated := false
			for _, f := range currMessage.Fields {
				if f.Name == fragment {
					fieldAlreadyCreated = true
				}
			}

			// Get field on the model.
			field := query.ModelField(query.Model(scm.asts, currModel), fragment)
			if field == nil {
				panic(fmt.Sprintf("cannot find field %s on model %s", fragment, currModel))
			}

			if !fieldAlreadyCreated {
				// Add the related model message as a field to the current message with typeInfo of Type_TYPE_MESSAGE.
				currMessage.Fields = append(currMessage.Fields, &proto.MessageField{
					Name: fragment,
					Type: &proto.TypeInfo{
						Type: proto.Type_TYPE_MESSAGE,
						// Repeated with be true in a 1:M relationship.
						Repeated: field.Repeated,
						MessageName: &wrapperspb.StringValue{
							Value: relatedModelMessageName,
						},
					},
					Optional:    field.Optional,
					MessageName: currMessage.Name,
				})

				currMessage = &proto.Message{
					Name:   relatedModelMessageName,
					Fields: []*proto.MessageField{},
				}
				scm.proto.Messages = append(scm.proto.Messages, currMessage)
			} else {
				for _, m := range scm.proto.Messages {
					if m.Name == relatedModelMessageName {
						currMessage = m
					}
				}
			}

			currModel = field.Type.Value
		} else {
			typeInfo, target, targetsOptionalField := scm.inferParserInputType(model, action, input, impl)

			// If this is the last or only target, then we add the field to the current message using the typeInfo.
			currMessage.Fields = append(currMessage.Fields, &proto.MessageField{
				Name:        fragment,
				Type:        typeInfo,
				Target:      target,
				Optional:    input.Optional || targetsOptionalField,
				MessageName: currMessage.Name,
			})
		}
	}
}

// Adds a set of proto.Messages to top level Messages registry for all inputs of an Action
func (scm *Builder) makeActionInputMessages(model *parser.ModelNode, action *parser.ActionNode, impl proto.OperationImplementation) {
	switch action.Type.Value {
	case parser.ActionTypeCreate:
		rootMessage := &proto.Message{
			Name:   makeInputMessageName(action.Name.Value),
			Fields: []*proto.MessageField{},
		}
		scm.proto.Messages = append(scm.proto.Messages, rootMessage)

		for _, input := range action.With {
			if input.Label == nil {
				// If its an implicit input, then create a nested object input structure.
				scm.makeMessageHierarchyFromImplicitInput(rootMessage, input, model, action, impl)
			} else {
				// This is an explicit input, so the first and only fragment will reference the type used.
				typeInfo := scm.explicitInputToTypeInfo(input)

				rootMessage.Fields = append(rootMessage.Fields, &proto.MessageField{
					Name:        input.Label.Value,
					Type:        typeInfo,
					Optional:    input.Optional,
					MessageName: rootMessage.Name,
				})
			}
		}
	case parser.ActionTypeGet, parser.ActionTypeDelete, parser.ActionTypeRead, parser.ActionTypeWrite:
		// Create message and add it to the proto schema
		messageName := makeInputMessageName(action.Name.Value)
		message := scm.makeMessageFromActionInputNodes(messageName, action.Inputs, model, action, impl)
		scm.proto.Messages = append(scm.proto.Messages, message)
	case parser.ActionTypeUpdate:
		// Create where message and add it to the proto schema
		whereMessageName := makeWhereMessageName(action.Name.Value)
		whereMessage := scm.makeMessageFromActionInputNodes(whereMessageName, action.Inputs, model, action, impl)
		scm.proto.Messages = append(scm.proto.Messages, whereMessage)

		// Create values message and add it to the proto schema
		valuesMessage := &proto.Message{
			Name:   makeValuesMessageName(action.Name.Value),
			Fields: []*proto.MessageField{},
		}
		scm.proto.Messages = append(scm.proto.Messages, valuesMessage)
		for _, input := range action.With {
			if input.Label == nil {
				// If its an implicit input, then create a nested object input structure.
				scm.makeMessageHierarchyFromImplicitInput(valuesMessage, input, model, action, impl)
			} else {
				// This is an explicit input, so the first and only fragment will reference the type used.
				typeInfo := scm.explicitInputToTypeInfo(input)

				valuesMessage.Fields = append(valuesMessage.Fields, &proto.MessageField{
					Name:        input.Label.Value,
					Type:        typeInfo,
					Optional:    input.Optional,
					MessageName: valuesMessage.Name,
				})
			}
		}

		// Create root operation message with "where" and "values" fields.
		scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
			Name: makeInputMessageName(action.Name.Value),
			Fields: []*proto.MessageField{
				{
					Name: "where",
					Optional: len(whereMessage.Fields) == 0 || lo.EveryBy(whereMessage.Fields, func(f *proto.MessageField) bool {
						return f.Optional
					}),
					MessageName: makeInputMessageName(action.Name.Value),
					Type: &proto.TypeInfo{
						Type:        proto.Type_TYPE_MESSAGE,
						MessageName: wrapperspb.String(makeWhereMessageName(action.Name.Value)),
					},
				},
				{
					Name: "values",
					Optional: len(valuesMessage.Fields) == 0 || lo.EveryBy(valuesMessage.Fields, func(f *proto.MessageField) bool {
						return f.Optional
					}),
					MessageName: makeInputMessageName(action.Name.Value),
					Type: &proto.TypeInfo{
						Type:        proto.Type_TYPE_MESSAGE,
						MessageName: wrapperspb.String(makeValuesMessageName(action.Name.Value)),
					},
				},
			},
		})
	case parser.ActionTypeList:
		wheres := []*proto.MessageField{}
		for _, input := range action.Inputs {
			typeInfo, target, targetsOptionalField := scm.inferParserInputType(model, action, input, impl)

			if target != nil {
				queryMessage, err := makeListQueryInputMessage(typeInfo, targetsOptionalField)

				if err != nil {
					continue
				}

				if !lo.SomeBy(scm.proto.Messages, func(m *proto.Message) bool { return m.Name == queryMessage.Name }) {
					scm.proto.Messages = append(scm.proto.Messages, queryMessage)
				}

				wheres = append(wheres, &proto.MessageField{
					Name: input.Name(),
					Type: &proto.TypeInfo{
						Type:        proto.Type_TYPE_MESSAGE,
						MessageName: wrapperspb.String(queryMessage.Name)},
					Target:      target,
					Optional:    input.Optional || targetsOptionalField,
					MessageName: makeWhereMessageName(action.Name.Value),
				})
			} else {
				wheres = append(wheres, &proto.MessageField{
					Name:        input.Name(),
					Type:        typeInfo,
					Optional:    input.Optional || targetsOptionalField,
					MessageName: makeWhereMessageName(action.Name.Value),
				})
			}
		}

		scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
			Name:   makeWhereMessageName(action.Name.Value),
			Fields: wheres,
		})

		scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
			Name: makeInputMessageName(action.Name.Value),
			Fields: []*proto.MessageField{
				{
					Name: "where",
					Optional: len(wheres) == 0 || lo.EveryBy(wheres, func(f *proto.MessageField) bool {
						return f.Optional
					}),
					MessageName: makeInputMessageName(action.Name.Value),
					Type: &proto.TypeInfo{
						Type:        proto.Type_TYPE_MESSAGE,
						MessageName: wrapperspb.String(makeWhereMessageName(action.Name.Value)),
					},
				},
				// Include pagination fields
				{
					Name:        "first",
					MessageName: makeInputMessageName(action.Name.Value),
					Optional:    true,
					Type: &proto.TypeInfo{
						Type: proto.Type_TYPE_INT,
					},
				},
				{
					Name:        "after",
					MessageName: makeInputMessageName(action.Name.Value),
					Optional:    true,
					Type: &proto.TypeInfo{
						Type: proto.Type_TYPE_STRING,
					},
				},
				{
					Name:        "last",
					MessageName: makeInputMessageName(action.Name.Value),
					Optional:    true,
					Type: &proto.TypeInfo{
						Type: proto.Type_TYPE_INT,
					},
				},
				{
					Name:        "before",
					MessageName: makeInputMessageName(action.Name.Value),
					Optional:    true,
					Type: &proto.TypeInfo{
						Type: proto.Type_TYPE_STRING,
					},
				},
			},
		})
	default:
		panic("unhandled operation type when creating input message types")
	}
}

func (scm *Builder) makeModel(decl *parser.DeclarationNode) {
	parserModel := decl.Model
	protoModel := &proto.Model{
		Name: parserModel.Name.Value,
	}
	for _, section := range parserModel.Sections {
		switch {
		case section.Fields != nil:
			fields := scm.makeFields(section.Fields, protoModel.Name)
			protoModel.Fields = append(protoModel.Fields, fields...)

		case section.Functions != nil:
			ops := scm.makeActions(section.Functions, protoModel.Name, proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM)
			protoModel.Operations = append(protoModel.Operations, ops...)

		case section.Operations != nil:
			ops := scm.makeActions(section.Operations, protoModel.Name, proto.OperationImplementation_OPERATION_IMPLEMENTATION_AUTO)
			protoModel.Operations = append(protoModel.Operations, ops...)

		case section.Attribute != nil:
			scm.applyModelAttribute(parserModel, protoModel, section.Attribute)
		default:
			// this is possible if the user defines an empty block in the schema e.g. "fields {}"
			// this isn't really an error so we can just ignore these sections
		}
	}

	if decl.Model.Name.Value == parser.ImplicitIdentityModelName {
		protoModel.Operations = append(protoModel.Operations, scm.makeAuthenticate())
		protoModel.Operations = append(protoModel.Operations, scm.makeRequestPasswordReset())
		protoModel.Operations = append(protoModel.Operations, scm.makePasswordReset())
	}

	scm.proto.Models = append(scm.proto.Models, protoModel)
}

func (scm *Builder) makeAuthenticate() *proto.Operation {
	inputMessageName := makeInputMessageName(parser.AuthenticateOperationName)
	responseMessageName := makeResponseMessageName(parser.AuthenticateOperationName)
	emailPasswordMessageName := makeInputMessageName("EmailPassword")

	op := proto.Operation{
		ModelName:           parser.ImplicitIdentityModelName,
		Name:                parser.AuthenticateOperationName,
		Implementation:      proto.OperationImplementation_OPERATION_IMPLEMENTATION_RUNTIME,
		Type:                proto.OperationType_OPERATION_TYPE_WRITE,
		InputMessageName:    inputMessageName,
		ResponseMessageName: responseMessageName,
	}

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name: emailPasswordMessageName,
		Fields: []*proto.MessageField{
			{
				Name:        "email",
				MessageName: emailPasswordMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
			{
				Name:        "password",
				MessageName: emailPasswordMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
		},
	})

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name: inputMessageName,
		Fields: []*proto.MessageField{
			{
				Name:        "createIfNotExists",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_BOOL},
				Optional:    true,
			},
			{
				Name:        "emailPassword",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_MESSAGE, MessageName: wrapperspb.String(emailPasswordMessageName)},
				Optional:    false,
			},
		},
	})

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name: responseMessageName,
		Fields: []*proto.MessageField{
			{
				Name:        "identityCreated",
				MessageName: responseMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_BOOL},
				Optional:    false,
			},
			{
				Name:        "token",
				MessageName: responseMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
		},
	})

	return &op
}

func (scm *Builder) makeRequestPasswordReset() *proto.Operation {
	inputMessageName := makeInputMessageName(parser.RequestPasswordResetOperationName)
	responseMessageName := makeResponseMessageName(parser.RequestPasswordResetOperationName)

	op := proto.Operation{
		ModelName:           parser.ImplicitIdentityModelName,
		Name:                parser.RequestPasswordResetOperationName,
		Implementation:      proto.OperationImplementation_OPERATION_IMPLEMENTATION_RUNTIME,
		Type:                proto.OperationType_OPERATION_TYPE_WRITE,
		InputMessageName:    inputMessageName,
		ResponseMessageName: responseMessageName,
	}

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name: inputMessageName,
		Fields: []*proto.MessageField{
			{
				Name:        "email",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
			{
				Name:        "redirectUrl",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
		},
	})

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name:   responseMessageName,
		Fields: []*proto.MessageField{},
	})

	return &op
}

func (scm *Builder) makePasswordReset() *proto.Operation {
	inputMessageName := makeInputMessageName(parser.PasswordResetOperationName)
	responseMessageName := makeResponseMessageName(parser.PasswordResetOperationName)

	op := proto.Operation{
		ModelName:           parser.ImplicitIdentityModelName,
		Name:                parser.PasswordResetOperationName,
		Implementation:      proto.OperationImplementation_OPERATION_IMPLEMENTATION_RUNTIME,
		Type:                proto.OperationType_OPERATION_TYPE_WRITE,
		InputMessageName:    inputMessageName,
		ResponseMessageName: responseMessageName,
	}

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name: inputMessageName,
		Fields: []*proto.MessageField{
			{
				Name:        "token",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
			{
				Name:        "password",
				MessageName: inputMessageName,
				Type:        &proto.TypeInfo{Type: proto.Type_TYPE_STRING},
				Optional:    false,
			},
		},
	})

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name:   responseMessageName,
		Fields: []*proto.MessageField{},
	})

	return &op
}

func (scm *Builder) makeRole(decl *parser.DeclarationNode) {
	parserRole := decl.Role
	protoRole := &proto.Role{
		Name: parserRole.Name.Value,
	}
	for _, section := range parserRole.Sections {
		for _, parserDomain := range section.Domains {
			protoRole.Domains = append(protoRole.Domains, stripQuotes(parserDomain.Domain))
		}
		for _, parserEmail := range section.Emails {
			protoRole.Emails = append(protoRole.Emails, stripQuotes(parserEmail.Email))
		}
	}
	scm.proto.Roles = append(scm.proto.Roles, protoRole)
}

func (scm *Builder) makeAPI(decl *parser.DeclarationNode) {
	parserAPI := decl.API
	protoAPI := &proto.Api{
		Name:      parserAPI.Name.Value,
		ApiModels: []*proto.ApiModel{},
	}
	for _, section := range parserAPI.Sections {
		switch {
		case len(section.Models) > 0:
			for _, parserApiModel := range section.Models {
				protoModel := &proto.ApiModel{
					ModelName: parserApiModel.Name.Value,
				}
				protoAPI.ApiModels = append(protoAPI.ApiModels, protoModel)
			}
		}
	}
	scm.proto.Apis = append(scm.proto.Apis, protoAPI)
}

func (scm *Builder) makeAnyType() {
	any := &proto.Message{
		Name: "Any",
	}

	scm.proto.Messages = append(scm.proto.Messages, any)
}

func (scm *Builder) makeMessage(decl *parser.DeclarationNode) {
	parserMsg := decl.Message

	fields := lo.Map(parserMsg.Fields, func(f *parser.FieldNode, _ int) *proto.MessageField {
		field := &proto.MessageField{
			Name: f.Name.Value,
			Type: &proto.TypeInfo{
				Type:     scm.parserTypeToProtoType(f.Type.Value),
				Repeated: f.Repeated,
			},
			Optional:    f.Optional,
			MessageName: parserMsg.Name.Value,
		}

		if field.Type.Type == proto.Type_TYPE_ENUM {
			field.Type.EnumName = wrapperspb.String(f.Type.Value)
		}

		if field.Type.Type == proto.Type_TYPE_MESSAGE {
			field.Type.MessageName = wrapperspb.String(f.Type.Value)
		}

		if field.Type.Type == proto.Type_TYPE_MODEL {
			field.Type.ModelName = wrapperspb.String(f.Type.Value)
		}

		return field
	})

	scm.proto.Messages = append(scm.proto.Messages, &proto.Message{
		Name:   parserMsg.Name.Value,
		Fields: fields,
	})
}

func (scm *Builder) makeEnum(decl *parser.DeclarationNode) {
	parserEnum := decl.Enum
	enum := &proto.Enum{
		Name:   parserEnum.Name.Value,
		Values: []*proto.EnumValue{},
	}
	for _, value := range parserEnum.Values {
		enum.Values = append(enum.Values, &proto.EnumValue{
			Name: value.Name.Value,
		})
	}
	scm.proto.Enums = append(scm.proto.Enums, enum)
}

func (scm *Builder) makeFields(parserFields []*parser.FieldNode, modelName string) []*proto.Field {
	protoFields := []*proto.Field{}
	for _, parserField := range parserFields {
		protoField := scm.makeField(parserField, modelName)
		protoFields = append(protoFields, protoField)
	}
	return protoFields
}

func (scm *Builder) makeField(parserField *parser.FieldNode, modelName string) *proto.Field {
	typeInfo := scm.parserFieldToProtoTypeInfo(parserField)
	protoField := &proto.Field{
		ModelName: modelName,
		Name:      parserField.Name.Value,
		Type:      typeInfo,
		Optional:  parserField.Optional,
	}

	// Handle @unique attribute at model level which expresses
	// unique constrains across multiple fields
	model := query.Model(scm.asts, modelName)
	for _, attr := range query.ModelAttributes(model) {
		if attr.Name.Value != parser.AttributeUnique {
			continue
		}

		value, _ := attr.Arguments[0].Expression.ToValue()
		fieldNames := lo.Map(value.Array.Values, func(v *parser.Operand, i int) string {
			return v.Ident.ToString()
		})

		if !lo.Contains(fieldNames, parserField.Name.Value) {
			continue
		}

		protoField.UniqueWith = lo.Filter(fieldNames, func(v string, i int) bool {
			return v != parserField.Name.Value
		})
	}

	scm.applyFieldAttributes(parserField, protoField)

	// Auto-inserted foreign key field
	if query.IsForeignKey(scm.asts, model, parserField) {
		modelField := query.Field(model, strings.TrimSuffix(parserField.Name.Value, "Id"))
		protoField.ForeignKeyInfo = &proto.ForeignKeyInfo{
			RelatedModelName:  modelField.Type.Value,
			RelatedModelField: parser.ImplicitFieldNameId,
		}
	}

	// Model field (sibling to foreign key)
	if query.IsModel(scm.asts, parserField.Type.Value) && !parserField.Repeated {
		protoField.ForeignKeyFieldName = wrapperspb.String(fmt.Sprintf("%sId", parserField.Name.Value))
	}

	// If this is a HasMany relationship field - see if we can mark it with
	// an explicit InverseFieldName - i.e. one defined by an @relation attribute.
	if protoField.Type.Type == proto.Type_TYPE_MODEL && protoField.Type.Repeated {
		scm.setExplicitInverseFieldName(parserField, protoField)
	}

	return protoField
}

// setExplicitInverseFieldName works on fields of type Model that are repeated. It looks to
// see if the schema defines an explicit inverse relationship field for it, and when so, sets
// this field's InverseFieldName property accordingly.
func (scm *Builder) setExplicitInverseFieldName(thisParserField *parser.FieldNode, thisProtoField *proto.Field) {

	// We have to look in the related model's fields, to see if any of them have an @relation
	// attribute that refers back to this field.

	nameOfRelatedModel := thisProtoField.Type.ModelName.Value
	relatedModel := query.Model(scm.asts, nameOfRelatedModel)
	for _, remoteField := range query.ModelFields(relatedModel) {
		if !query.FieldHasAttribute(remoteField, parser.AttributeRelation) {
			continue
		}
		relationAttr := query.FieldGetAttribute(remoteField, parser.AttributeRelation)
		inverseFieldName := attributeFirstArgAsIdentifier(relationAttr)
		if inverseFieldName == thisProtoField.Name {
			// We've found the inverse.
			thisProtoField.InverseFieldName = wrapperspb.String(remoteField.Name.Value)
			return
		}
	}
}

// attributeFirstArgAsIdentifier fishes out the identifier being held
// by the first argument of the given attribute. It must only be called when
// you know that it is well formed for that.
func attributeFirstArgAsIdentifier(attr *parser.AttributeNode) string {
	expr := attr.Arguments[0].Expression
	operand, _ := expr.ToValue()
	theString := operand.Ident.Fragments[0].Fragment
	return theString
}

func (scm *Builder) makeActions(actions []*parser.ActionNode, modelName string, impl proto.OperationImplementation) []*proto.Operation {
	protoOps := []*proto.Operation{}

	for _, action := range actions {
		protoOp := scm.makeAction(action, modelName, impl)
		protoOps = append(protoOps, protoOp)
	}
	return protoOps
}

func (scm *Builder) makeAction(action *parser.ActionNode, modelName string, impl proto.OperationImplementation) *proto.Operation {
	protoOp := &proto.Operation{
		ModelName:        modelName,
		InputMessageName: makeInputMessageName(action.Name.Value),
		Name:             action.Name.Value,
		Implementation:   impl,
		Type:             scm.mapToOperationType(action.Type.Value),
	}

	model := query.Model(scm.asts, modelName)

	if action.IsArbitraryFunction() {
		// if its an arbitrary function, then the input will exist in scm.Messages unless the inputs were defined inline
		// output messages will always be defined in scm.Messages
		usesAny := action.Inputs[0].Type.ToString() == parser.MessageFieldTypeAny
		usingInlineInputs := true

		for _, ast := range scm.asts {
			for _, d := range ast.Declarations {
				if d.Message != nil && d.Message.Name.Value == action.Inputs[0].Type.ToString() {
					usingInlineInputs = false
				}
			}
		}

		switch {
		case usesAny:
			protoOp.InputMessageName = action.Inputs[0].Type.ToString()
		case usingInlineInputs:
			scm.makeActionInputMessages(model, action, impl)
		default:
			protoOp.InputMessageName = action.Inputs[0].Type.ToString()
		}

		protoOp.ResponseMessageName = action.Returns[0].Type.ToString()
	} else {
		// we need to generate the messages representing the inputs to the scm.Messages
		scm.makeActionInputMessages(model, action, impl)
	}

	scm.applyActionAttributes(action, protoOp, modelName)

	return protoOp
}

func (scm *Builder) inferParserInputType(
	model *parser.ModelNode,
	op *parser.ActionNode,
	input *parser.ActionInputNode,
	impl proto.OperationImplementation,
) (t *proto.TypeInfo, target []string, targetsOptionalField bool) {
	idents := input.Type.Fragments
	protoType := scm.parserTypeToProtoType(idents[0].Fragment)

	var modelName *wrapperspb.StringValue
	var fieldName *wrapperspb.StringValue
	var enumName *wrapperspb.StringValue

	if protoType == proto.Type_TYPE_ENUM {
		enumName = &wrapperspb.StringValue{
			Value: idents[0].Fragment,
		}
	}

	// If any target field is optional, then the input becomes optional,
	// regardless of how it's specified in the schema definition
	targetsOptionalField = false

	if protoType == proto.Type_TYPE_UNKNOWN {
		// If we haven't been able to resolve the type of the input it
		// must be a model field, so we need to resolve it

		var field *parser.FieldNode
		currModel := model

		for _, ident := range idents {

			target = append(target, ident.Fragment)

			field = query.ModelField(currModel, ident.Fragment)

			if field.Optional {
				targetsOptionalField = true
			}

			m := query.Model(scm.asts, field.Type.Value)
			if m != nil {
				currModel = m
			}
		}

		protoType = scm.parserFieldToProtoTypeInfo(field).Type

		modelName = &wrapperspb.StringValue{
			Value: currModel.Name.Value,
		}
		fieldName = &wrapperspb.StringValue{
			Value: field.Name.Value,
		}

		if protoType == proto.Type_TYPE_ENUM {
			enumName = &wrapperspb.StringValue{
				Value: field.Type.Value,
			}
		}
	}

	return &proto.TypeInfo{
		Type:      protoType,
		Repeated:  false,
		ModelName: modelName,
		FieldName: fieldName,
		EnumName:  enumName,
	}, target, targetsOptionalField
}

// parserType could be a built-in type or a user-defined model or enum
func (scm *Builder) parserTypeToProtoType(parserType string) proto.Type {
	switch {
	case parserType == parser.FieldTypeText:
		return proto.Type_TYPE_STRING
	case parserType == parser.FieldTypeID:
		return proto.Type_TYPE_ID
	case parserType == parser.FieldTypeBoolean:
		return proto.Type_TYPE_BOOL
	case parserType == parser.FieldTypeNumber:
		return proto.Type_TYPE_INT
	case parserType == parser.FieldTypeDate:
		return proto.Type_TYPE_DATE
	case parserType == parser.FieldTypeDatetime:
		return proto.Type_TYPE_DATETIME
	case parserType == parser.FieldTypeSecret:
		return proto.Type_TYPE_SECRET
	case parserType == parser.FieldTypePassword:
		return proto.Type_TYPE_PASSWORD
	case query.IsModel(scm.asts, parserType):
		return proto.Type_TYPE_MODEL
	case query.IsEnum(scm.asts, parserType):
		return proto.Type_TYPE_ENUM
	case query.IsMessage(scm.asts, parserType):
		return proto.Type_TYPE_MESSAGE
	case parserType == parser.MessageFieldTypeAny:
		return proto.Type_TYPE_ANY
	default:
		return proto.Type_TYPE_UNKNOWN
	}
}

func (scm *Builder) explicitInputToTypeInfo(input *parser.ActionInputNode) *proto.TypeInfo {
	protoType := scm.parserTypeToProtoType(input.Type.Fragments[0].Fragment)

	disallowedExplicitInputTypes := []proto.Type{
		proto.Type_TYPE_MODEL,
		proto.Type_TYPE_MESSAGE,
		proto.Type_TYPE_ANY,
		proto.Type_TYPE_UNKNOWN}

	if lo.Contains(disallowedExplicitInputTypes, protoType) {
		panic(fmt.Sprintf("explicit input field %s cannot be of type %s", input.Name(), protoType))
	}

	var enumName *wrapperspb.StringValue
	if protoType == proto.Type_TYPE_ENUM {
		enumName = &wrapperspb.StringValue{
			Value: query.Enum(scm.asts, input.Type.Fragments[0].Fragment).Name.Value,
		}
	}

	return &proto.TypeInfo{
		Type:     protoType,
		EnumName: enumName,
	}
}

func (scm *Builder) parserFieldToProtoTypeInfo(field *parser.FieldNode) *proto.TypeInfo {

	protoType := scm.parserTypeToProtoType(field.Type.Value)
	var modelName *wrapperspb.StringValue
	var enumName *wrapperspb.StringValue

	switch protoType {
	case proto.Type_TYPE_MODEL:
		modelName = &wrapperspb.StringValue{
			Value: query.Model(scm.asts, field.Type.Value).Name.Value,
		}
	case proto.Type_TYPE_ENUM:
		enumName = &wrapperspb.StringValue{
			Value: query.Enum(scm.asts, field.Type.Value).Name.Value,
		}
	}

	return &proto.TypeInfo{
		Type:      protoType,
		Repeated:  field.Repeated,
		ModelName: modelName,
		EnumName:  enumName,
	}
}

func (scm *Builder) applyModelAttribute(parserModel *parser.ModelNode, protoModel *proto.Model, attribute *parser.AttributeNode) {
	switch attribute.Name.Value {
	case parser.AttributePermission:
		perm := scm.permissionAttributeToProtoPermission(attribute)
		perm.ModelName = protoModel.Name
		protoModel.Permissions = append(protoModel.Permissions, perm)
	}
}

func (scm *Builder) applyActionAttributes(action *parser.ActionNode, protoOperation *proto.Operation, modelName string) {
	for _, attribute := range action.Attributes {
		switch attribute.Name.Value {
		case parser.AttributePermission:
			perm := scm.permissionAttributeToProtoPermission(attribute)
			perm.ModelName = modelName
			perm.OperationName = wrapperspb.String(protoOperation.Name)
			protoOperation.Permissions = append(protoOperation.Permissions, perm)
		case parser.AttributeWhere:
			expr, _ := attribute.Arguments[0].Expression.ToString()
			where := &proto.Expression{Source: expr}
			protoOperation.WhereExpressions = append(protoOperation.WhereExpressions, where)
		case parser.AttributeSet:
			expr, _ := attribute.Arguments[0].Expression.ToString()
			set := &proto.Expression{Source: expr}
			protoOperation.SetExpressions = append(protoOperation.SetExpressions, set)
		case parser.AttributeValidate:
			expr, _ := attribute.Arguments[0].Expression.ToString()
			set := &proto.Expression{Source: expr}
			protoOperation.ValidationExpressions = append(protoOperation.ValidationExpressions, set)
		}
	}
}

func (scm *Builder) applyFieldAttributes(parserField *parser.FieldNode, protoField *proto.Field) {
	for _, fieldAttribute := range parserField.Attributes {
		switch fieldAttribute.Name.Value {
		case parser.AttributeUnique:
			protoField.Unique = true
		case parser.AttributePrimaryKey:
			protoField.PrimaryKey = true
		case parser.AttributeDefault:
			defaultValue := &proto.DefaultValue{}
			if len(fieldAttribute.Arguments) == 1 {
				expr := fieldAttribute.Arguments[0].Expression
				source, _ := expr.ToString()
				defaultValue.Expression = &proto.Expression{
					Source: source,
				}
			} else {
				defaultValue.UseZeroValue = true
			}
			protoField.DefaultValue = defaultValue
		case parser.AttributeRelation:
			// We cannot process this field attribute here. But here is an explanation
			// of why that is so - for future readers.
			//
			// This attribute (the @relation attribute) is placed one HasOne relation fields in the input schema -
			// to specify a field in its related model that is its inverse. We decided this because
			// it seems most intuitive for the user - given that to use 1:many relations at all,
			// you HAVE TO HAVE the hasOne end.
			//
			// HOWEVER we want the InverseFieldName field property in the protobuf representation
			// to live on the RELATED model's field, i.e. the repeated relationship field - NOT this field.
			//
			// The problem is that the related model might not even be present yet in the proto.Schema that is
			// currently under construction - because the call-graph of the construction process builds the proto
			// for each model in turn, and might not have reached the related model yet.
			//
			// INSTEAD we sort it all out when we reach hasMany fields at the other end of the inverse relation.
			// See the call to setExplicitInverseFieldName() at the end of scm.makeField().
		}
	}
}

func (scm *Builder) permissionAttributeToProtoPermission(attr *parser.AttributeNode) *proto.PermissionRule {
	pr := &proto.PermissionRule{}
	for _, arg := range attr.Arguments {
		switch arg.Label.Value {
		case "expression":
			expr, _ := arg.Expression.ToString()
			pr.Expression = &proto.Expression{Source: expr}
		case "roles":
			value, _ := arg.Expression.ToValue()
			for _, item := range value.Array.Values {
				pr.RoleNames = append(pr.RoleNames, item.Ident.Fragments[0].Fragment)
			}
		case "actions":
			value, _ := arg.Expression.ToValue()
			for _, v := range value.Array.Values {
				pr.OperationsTypes = append(pr.OperationsTypes, scm.mapToOperationType(v.Ident.Fragments[0].Fragment))
			}
		}
	}
	return pr
}

func (scm *Builder) mapToOperationType(parsedOperation string) proto.OperationType {
	switch parsedOperation {
	case parser.ActionTypeCreate:
		return proto.OperationType_OPERATION_TYPE_CREATE
	case parser.ActionTypeUpdate:
		return proto.OperationType_OPERATION_TYPE_UPDATE
	case parser.ActionTypeGet:
		return proto.OperationType_OPERATION_TYPE_GET
	case parser.ActionTypeList:
		return proto.OperationType_OPERATION_TYPE_LIST
	case parser.ActionTypeDelete:
		return proto.OperationType_OPERATION_TYPE_DELETE
	case parser.ActionTypeRead:
		return proto.OperationType_OPERATION_TYPE_READ
	case parser.ActionTypeWrite:
		return proto.OperationType_OPERATION_TYPE_WRITE
	default:
		return proto.OperationType_OPERATION_TYPE_UNKNOWN
	}
}

// stripQuotes removes all double quotes from the given string, regardless of where they are.
func stripQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}

func makeInputMessageName(opName string, subMessageNames ...string) string {
	if len(subMessageNames) > 0 {
		subFieldNames := strings.Join(
			lo.Map(subMessageNames, func(s string, _ int) string { return casing.ToCamel(s) }),
			"")

		return fmt.Sprintf("%s%sInput", casing.ToCamel(opName), subFieldNames)
	}
	return fmt.Sprintf("%sInput", casing.ToCamel(opName))
}

func makeWhereMessageName(opName string) string {
	return fmt.Sprintf("%sWhere", casing.ToCamel(opName))
}

func makeValuesMessageName(opName string) string {
	return fmt.Sprintf("%sValues", casing.ToCamel(opName))
}

func makeResponseMessageName(opName string) string {
	return fmt.Sprintf("%sResponse", casing.ToCamel(opName))
}
