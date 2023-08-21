package validation

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/schema/node"
	"github.com/teamkeel/keel/schema/parser"
	"github.com/teamkeel/keel/schema/query"
	"github.com/teamkeel/keel/schema/validation/errorhandling"
)

// UniqueAttributeRule validates that unique attributes are valid according to the following rules:
// - @unique can't be used on Timestamp fields
// - @unique can't be used on has-many relations
// - composite @unique attributes must not have duplicate field names
// - composite @unique can't specify has-many fields
func UniqueAttributeRule(asts []*parser.AST, errs *errorhandling.ValidationErrors) Visitor {
	var currentModel *parser.ModelNode
	var currentField *parser.FieldNode

	currentModelIsBuiltIn := false

	return Visitor{
		EnterModel: func(m *parser.ModelNode) {
			if m.BuiltIn {
				currentModelIsBuiltIn = true
			}

			currentModel = m
		},
		LeaveModel: func(m *parser.ModelNode) {
			currentModel = nil
			currentModelIsBuiltIn = false
		},
		EnterField: func(f *parser.FieldNode) {
			if f.BuiltIn {
				return
			}
			currentField = f
		},
		LeaveField: func(f *parser.FieldNode) {
			currentField = nil
		},
		EnterAttribute: func(attr *parser.AttributeNode) {
			if currentModelIsBuiltIn {
				return
			}
			if attr.Name.Value != parser.AttributeUnique {
				return
			}

			compositeUnique := currentField == nil

			switch {
			case compositeUnique:
				if len(attr.Arguments) > 0 {
					value, _ := attr.Arguments[0].Expression.ToValue()

					if value.Array != nil {
						fieldNames := lo.Map(value.Array.Values, func(o *parser.Operand, _ int) string {
							return o.Ident.ToString()
						})

						// check there are no duplicate field names specified in the composite uniqueness
						// constraint e.g @unique([fieldA, fieldA])

						dupes := findDuplicateConstraints(fieldNames)

						if len(dupes) > 0 {
							for _, dupe := range dupes {
								// find the last occurrence of the duplicate in the composite uniqueness constraint values list
								// so we can highlight that node in the validation error.
								_, index, found := lo.FindLastIndexOf(value.Array.Values, func(o *parser.Operand) bool {
									return o.Ident.ToString() == dupe
								})

								if found {
									errs.AppendError(uniqueRestrictionError(value.Array.Values[index].Node, fmt.Sprintf("Field '%s' has already been specified as a constraint", dupe)))
								}
							}
						}

						// check every field specified in the unique constraint against the standard
						// restrictions for @unique attribute usage
						for i, uniqueField := range fieldNames {
							field := query.ModelField(currentModel, uniqueField)

							if field == nil {
								// the field isnt a recognised field on the model, so abort as this is covered
								// by another validation
								continue
							}
							if permitted, reason := uniquePermitted(field); !permitted {
								errs.AppendError(uniqueRestrictionError(value.Array.Values[i].Node, reason))
							}
						}
					}
				}
			default:
				// in this case, we know we are dealing with a @unique attribute attached
				// to a field
				if permitted, reason := uniquePermitted(currentField); !permitted {
					errs.AppendError(uniqueRestrictionError(attr.Node, reason))
				}
			}
		},
	}
}

func uniqueRestrictionError(node node.Node, reason string) *errorhandling.ValidationError {
	return errorhandling.NewValidationErrorWithDetails(
		errorhandling.TypeError,
		errorhandling.ErrorDetails{
			Message: reason,
		},
		node,
	)
}

func findDuplicateConstraints(constraints []string) (dupes []string) {
	seen := map[string]bool{}

	for _, constraint := range constraints {
		if _, found := seen[constraint]; found {
			dupes = append(dupes, constraint)

			continue
		}

		seen[constraint] = true
	}

	return dupes
}

func uniquePermitted(f *parser.FieldNode) (bool, string) {
	// if the field is repeated and not a scalar type, then it is a has-many relationship
	if f.Repeated && !f.IsScalar() {
		return false, "@unique is not permitted on has many relationships"
	}

	if f.Type.Value == parser.FieldTypeDatetime {
		return false, "@unique is not permitted on Timestamp fields"
	}

	return true, ""
}
