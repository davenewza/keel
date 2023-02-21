package jsonschema

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/proto"
	"github.com/xeipuuv/gojsonschema"
)

type JSONSchema struct {
	// Type is generally just a string, but when we need a type to be
	// null it is a list containing the type and the string "null".
	// In JSON output for most cases we just want a string, not a list
	// of one string, so we use any here so it can be either
	Type any `json:"type,omitempty"`

	// The enum field needs to be able to contains strings and null,
	// so we use *string here
	Enum []*string `json:"enum,omitempty"`

	// Validation for strings
	Format string `json:"format,omitempty"`

	// Validation for objects
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
	Required             []string              `json:"required,omitempty"`

	// For arrays
	Items *JSONSchema `json:"items,omitempty"`

	// Used to link to a type defined in the root $defs
	Ref string `json:"$ref,omitempty"`

	// Only used in the root JSONSchema object to define types that
	// can then be referenced using $ref
	Components *Components `json:"components,omitempty"`
}

type Components struct {
	Schemas map[string]JSONSchema `json:"schemas"`
}

// ValidateRequest validates that the input is valid for the given operation and schema.
// If validation errors are found they will be contained in the returned result. If an error
// is returned then validation could not be completed, likely to do an invalid JSON schema
// being created.
func ValidateRequest(ctx context.Context, schema *proto.Schema, op *proto.Operation, input map[string]any) (*gojsonschema.Result, error) {
	requestType := JSONSchemaForOperation(ctx, schema, op)
	return gojsonschema.Validate(gojsonschema.NewGoLoader(requestType), gojsonschema.NewGoLoader(input))
}

func JSONSchemaForOperation(ctx context.Context, schema *proto.Schema, op *proto.Operation) JSONSchema {
	inputMessage := proto.FindMessage(schema.Messages, op.InputMessageName)
	return JSONSchemaForMessage(ctx, schema, op, inputMessage)
}

// Generates JSONSchema for an operation by generating properties for the root input message.
// Any subsequent nested messages are referenced.
func JSONSchemaForMessage(ctx context.Context, schema *proto.Schema, op *proto.Operation, message *proto.Message) JSONSchema {
	components := Components{
		Schemas: map[string]JSONSchema{},
	}

	root := JSONSchema{
		Type:                 "object",
		Properties:           map[string]JSONSchema{},
		AdditionalProperties: boolPtr(false),
	}

	for _, field := range message.Fields {
		prop := jsonSchemaForField(ctx, field, op, schema)

		// Merge components from this request schema into OpenAPI components
		if prop.Components != nil {
			for name, comp := range prop.Components.Schemas {
				components.Schemas[name] = comp
			}
			prop.Components = nil
		}

		root.Properties[field.Name] = prop

		// If the input is not optional then mark it required in the JSON schema
		if !field.Optional {
			root.Required = append(root.Required, field.Name)
		}
	}

	if len(components.Schemas) > 0 {
		root.Components = &components
	}

	return root
}

func jsonSchemaForField(ctx context.Context, field *proto.MessageField, op *proto.Operation, schema *proto.Schema) JSONSchema {
	components := &Components{
		Schemas: map[string]JSONSchema{},
	}

	prop := JSONSchema{}
	nullable := field.Optional

	if field.Type.Type == proto.Type_TYPE_MESSAGE {
		// If the field is a nested message, then need to create a ref instead.
		prop = JSONSchema{Ref: fmt.Sprintf("#/components/schemas/%s", field.Type.MessageName.Value)}
		// And add the nested message to schema components.
		message := proto.FindMessage(schema.Messages, field.Type.MessageName.Value)
		component := JSONSchemaForMessage(ctx, schema, op, message)

		// If that nested message component has ref fields itself, then its components must be bundled.
		if component.Components != nil {
			for name, comp := range component.Components.Schemas {
				components.Schemas[name] = comp
			}
			component.Components = nil
		}

		components.Schemas[field.Type.MessageName.Value] = component
	} else {
		if op != nil && op.Type == proto.OperationType_OPERATION_TYPE_LIST && field.IsModelField() {
			// If field is a list operation query input then generate query component.
			// TODO: to remove with https://linear.app/keel/issue/BLD-305/list-query-input-types-in-proto
			name, component := jsonSchemaForQueryObject(ctx, op, field, schema)

			if nullable {
				component.allowNull()
				name = "Nullable" + name
			}

			prop = JSONSchema{Ref: fmt.Sprintf("#/components/schemas/%s", name)}
			components.Schemas[name] = component

		} else {
			switch field.Type.Type {
			case proto.Type_TYPE_ID:
				prop.Type = "string"
			case proto.Type_TYPE_STRING:
				prop.Type = "string"
			case proto.Type_TYPE_BOOL:
				prop.Type = "boolean"
			case proto.Type_TYPE_INT:
				prop.Type = "number"
			case proto.Type_TYPE_DATE, proto.Type_TYPE_DATETIME, proto.Type_TYPE_TIMESTAMP:
				// date-time format allows both YYYY-MM-DD and full ISO8601/RFC3339 format
				prop.Type = "string"
				prop.Format = "date-time"
			case proto.Type_TYPE_ENUM:
				// For enum's we actually don't need to set the `type` field at all
				name := field.Type.EnumName.Value
				enum, _ := lo.Find(schema.Enums, func(e *proto.Enum) bool {
					return e.Name == field.Type.EnumName.Value
				})

				component := JSONSchema{}
				for _, v := range enum.Values {
					component.Enum = append(component.Enum, &v.Name)
				}

				if nullable {
					component.allowNull()
					name = "Nullable" + name
				}

				prop = JSONSchema{Ref: fmt.Sprintf("#/components/schemas/%s", name)}
				components.Schemas[name] = component
			}

			if nullable {
				prop.allowNull()
			}
		}
	}

	if len(components.Schemas) > 0 {
		prop.Components = components
	}

	return prop
}

// TODO: to remove with https://linear.app/keel/issue/BLD-305/list-query-input-types-in-proto
func jsonSchemaForQueryObject(ctx context.Context, op *proto.Operation, input *proto.MessageField, schema *proto.Schema) (string, JSONSchema) {
	switch input.Type.Type {
	case proto.Type_TYPE_ID:
		t := JSONSchema{
			Type: "string",
		}
		return "IDQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals": t,
				"oneOf":  {Type: "array", Items: &t},
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_STRING:
		t := JSONSchema{
			Type: "string",
		}
		return "StringQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals":     t,
				"startsWith": t,
				"endsWith":   t,
				"contains":   t,
				"oneOf":      {Type: "array", Items: &t},
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_DATE:
		t := JSONSchema{
			Type:   "string",
			Format: "date-time",
		}
		return "DateQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals":     t,
				"before":     t,
				"onOrBefore": t,
				"after":      t,
				"onOrAfter":  t,
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_DATETIME, proto.Type_TYPE_TIMESTAMP:
		t := JSONSchema{
			Type:   "string",
			Format: "date-time",
		}
		return "TimestampQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"before": t,
				"after":  t,
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_BOOL:
		return "BooleanQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals": {Type: "boolean"},
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_INT:
		t := JSONSchema{
			Type: "number",
		}
		return "IntQueryInput", JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals":              t,
				"lessThan":            t,
				"lessThanOrEquals":    t,
				"greaterThan":         t,
				"greaterThanOrEquals": t,
			},
			AdditionalProperties: boolPtr(false),
		}
	case proto.Type_TYPE_ENUM:
		t := JSONSchema{}
		enum, _ := lo.Find(schema.Enums, func(e *proto.Enum) bool {
			return e.Name == input.Type.EnumName.Value
		})
		for _, v := range enum.Values {
			t.Enum = append(t.Enum, &v.Name)
		}
		return fmt.Sprintf("%sQueryInput", enum.Name), JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"equals": t,
				"oneOf": {
					Type:  "array",
					Items: &t,
				},
			},
			AdditionalProperties: boolPtr(false),
		}
	}

	return "", JSONSchema{}
}

// allowNull makes sure that s allows null, either by modifying
// the type field or the enum field
//
// This is an area where OpenAPI differs from JSON Schema, from
// the OpenAPI spec:
//
//	| Note that there is no null type; instead, the nullable
//	| attribute is used as a modifier of the base type.
//
// We currently only support JSON schema
func (s *JSONSchema) allowNull() {
	t := s.Type
	switch t := t.(type) {
	case string:
		s.Type = []string{t, "null"}
	case []string:
		if lo.Contains(t, "null") {
			return
		}
		t = append(t, "null")
		s.Type = t
	}

	if len(s.Enum) > 0 && !lo.Contains(s.Enum, nil) {
		s.Enum = append(s.Enum, nil)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
