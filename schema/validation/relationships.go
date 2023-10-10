package validation

import (
	"fmt"

	"github.com/teamkeel/keel/schema/node"
	"github.com/teamkeel/keel/schema/parser"
	"github.com/teamkeel/keel/schema/query"
	"github.com/teamkeel/keel/schema/validation/errorhandling"
)

const (
	learnMore = "To learn more about relationships, visit https://docs.keel.so/models#relationships"
)

func RelationshipsRules(asts []*parser.AST, errs *errorhandling.ValidationErrors) Visitor {
	var currentModel *parser.ModelNode
	candidates := map[*parser.FieldNode][]*query.Relationship{}
	alreadyErrored := map[*parser.FieldNode]bool{}

	return Visitor{
		EnterModel: func(model *parser.ModelNode) {
			// For each relationship field, we generate possible candidates of fields
			// from the other model to form the relationship.  A relationship should
			// only ever have one candidate.
			//candidates = map[*parser.FieldNode][]*query.Relationship{}
			currentModel = model
		},

		LeaveModel: func(_ *parser.ModelNode) {

			for field := range candidates {
				if len(candidates[field]) == 1 {
					otherField := candidates[field][0].Field
					otherModel := candidates[field][0].Model
					if len(candidates[otherField]) == 1 && candidates[otherField][0].Field != field {
						// Field already in another relationship
						if !alreadyErrored[field] {
							errs.AppendError(makeRelationshipError(
								fmt.Sprintf("Field '%s' on model %s is already in a relationship with field '%s'", otherField.Name.Value, otherModel.Name.Value, candidates[otherField][0].Field.Name.Value),
								learnMore,
								field,
							))
							alreadyErrored[field] = true
						}
					}
				}

				if len(candidates[field]) > 1 {
					for i, candidate := range candidates[field] {
						// Skip the first relationship candidate match
						// since we can assume it to be valid.  For all further
						// candidates we return a validation error.  Each field
						// only have a single candidate on the other end.
						if i == 0 {
							continue
						}

						if candidate.Field == nil {
							continue
						}

						switch {
						case query.ValidOneToHasMany(field, candidate.Field):
							if !alreadyErrored[field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot determine which field on the %s model to form a one to many relationship with", candidate.Model.Name.Value),
									fmt.Sprintf("Use @relation to refer to a %s[] field on the %s model which is not yet in a relationship", currentModel.Name.Value, candidate.Model.Name.Value),
									field,
								))
								alreadyErrored[field] = true
							}
						case query.ValidOneToHasMany(candidate.Field, field):
							if !alreadyErrored[candidate.Field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot associate with field '%s' on model %s to form a one to many relationship as a relationship may already exist", field.Name.Value, currentModel.Name.Value),
									fmt.Sprintf("Use @relation to refer to a %s[] field on the %s model which is not yet in a relationship", candidate.Model.Name.Value, currentModel.Name.Value),
									candidate.Field,
								))
								alreadyErrored[candidate.Field] = true
							}
							if !alreadyErrored[field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot associate with field '%s' on model %s to form a one to many relationship as a relationship may already exist", candidate.Field.Name.Value, candidate.Model.Name.Value),
									"",
									field,
								))
								alreadyErrored[field] = true
							}
						case query.ValidUniqueOneToHasOne(field, candidate.Field):
							if !alreadyErrored[field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot determine which field on the %s model to form a one to one relationship with", candidate.Model.Name.Value),
									fmt.Sprintf("Use @relation to refer to a %s field on the %s model which is not yet in a relationship", currentModel.Name.Value, candidate.Model.Name.Value),
									field,
								))
								alreadyErrored[field] = true
							}
						case query.ValidUniqueOneToHasOne(candidate.Field, field):
							if !alreadyErrored[candidate.Field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot associate with field '%s' on model %s to form a one to one relationship as a relationship may already exist", field.Name.Value, currentModel.Name.Value),
									fmt.Sprintf("Use @relation to refer to a %s field on the %s model which is not yet in a relationship", candidate.Model.Name.Value, currentModel.Name.Value),
									candidate.Field,
								))
								alreadyErrored[candidate.Field] = true
							}
							if !alreadyErrored[field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot associate with field '%s' on model %s to form a one to one relationship as a relationship may already exist", candidate.Field.Name.Value, candidate.Model.Name.Value),
									learnMore,
									field,
								))
								alreadyErrored[field] = true
							}
						default:
							if !alreadyErrored[field] {
								errs.AppendError(makeRelationshipError(
									fmt.Sprintf("Cannot associate with field '%s' on model %s to form a relationship", candidate.Field.Name.Value, candidate.Model.Name.Value),
									learnMore,
									field,
								))
								alreadyErrored[field] = true
							}
						}
					}
				}
			}

			currentModel = nil
		},
		EnterField: func(currentField *parser.FieldNode) {
			if currentModel == nil {
				// If this is not a model field, then exit.
				return
			}

			// Check that the @relation attribute, if any, is define with exactly a single identifier.
			relationAttr := query.FieldGetAttribute(currentField, parser.AttributeRelation)

			var relation string
			if relationAttr != nil {
				var ok bool
				relation, ok = query.RelationAttributeValue(relationAttr)
				if !ok {
					errs.AppendError(makeRelationshipError(
						"The @relation value must refer to a field on the related model",
						fmt.Sprintf("For example, @relation(fieldName). %s", learnMore),
						relationAttr,
					))
					return
				}
			}

			// Check that the field type is a model.
			otherModel := query.Model(asts, currentField.Type.Value)
			if otherModel == nil {
				if relationAttr != nil {
					errs.AppendError(makeRelationshipError(
						"The @relation attribute cannot be used on non-model fields",
						learnMore,
						currentField,
					))
				}

				// If the field type is not a model, then this is not a relationship
				return
			}

			if relationAttr != nil {
				// @relation cannot be defined on a repeated field
				if currentField.Repeated {
					errs.AppendError(makeRelationshipError(
						"The @relation attribute must be defined on the other side of a one to many relationship",
						learnMore,
						relationAttr,
					))
					return
				}

				// @relation field does not exist
				otherField := query.Field(otherModel, relation)
				if otherField == nil {
					errs.AppendError(makeRelationshipError(
						fmt.Sprintf("The field '%s' does not exist on the %s model", relation, otherModel.Name.Value),
						learnMore,
						relationAttr.Arguments[0],
					))
					return
				}

				// @relation field type is not of this model
				if otherField.Type.Value != currentModel.Name.Value {
					errs.AppendError(makeRelationshipError(
						fmt.Sprintf("The field '%s' on the %s model must be of type %s in order to establish a relationship", relation, otherModel.Name.Value, currentModel.Name.Value),
						learnMore,
						relationAttr.Arguments[0],
					))
					return
				}

				// @relation field on other model is @unique
				if query.FieldIsUnique(otherField) {
					errs.AppendError(makeRelationshipError(
						fmt.Sprintf("Cannot create a relationship to the unique field '%s' on the %s model", relation, otherModel.Name.Value),
						fmt.Sprintf("In a one to one relationship, only this side must be marked as @unique. %s", learnMore),
						relationAttr.Arguments[0],
					))
					return
				}

				// This field is not @unique and relation field on other model is not repeated
				if !query.FieldIsUnique(currentField) && !otherField.Repeated {
					errs.AppendError(makeRelationshipError(
						"A one to one relationship requires a single side to be @unique",
						fmt.Sprintf("In a one to one relationship, the '%s' field must be @unique. %s", currentField.Name.Value, learnMore),
						currentField.Name,
					))
					return
				}

				// This field is @unique and relation field on other model is repeated
				if query.FieldIsUnique(currentField) && otherField.Repeated {
					errs.AppendError(makeRelationshipError(
						fmt.Sprintf("A one to one relationship cannot be made with repeated field '%s' on the %s model", otherField.Name.Value, otherModel.Name.Value),
						fmt.Sprintf("Either make '%s' non-repeated or define a new non-repeated field on %s. %s", otherField.Name.Value, otherModel.Name.Value, learnMore),
						relationAttr.Arguments[0],
					))
					return
				}

				// If belongsTo has @relation, check the field name matches hasMany
				otherFieldRelationAttribute := query.FieldGetAttribute(otherField, parser.AttributeRelation)
				if otherFieldRelationAttribute != nil {
					if _, ok := query.RelationAttributeValue(otherFieldRelationAttribute); ok {
						if query.FieldIsUnique(currentField) {
							errs.AppendError(makeRelationshipError(
								fmt.Sprintf("Cannot form a one to one relation with '%s' as it may already be in a relationship", otherField.Name.Value),
								fmt.Sprintf("In a one to one relationship, only the '%s' field must have the @relation attribute defined. %s", currentField.Name.Value, learnMore),
								currentField.Name,
							))
						} else {
							errs.AppendError(makeRelationshipError(
								fmt.Sprintf("Cannot form a one to many relation with '%s' as it may already be in a relationship", otherField.Name.Value),
								fmt.Sprintf("In a one to many relationship, only the '%s' field must have the @relation attribute defined. %s", currentField.Name.Value, learnMore),
								currentField.Name,
							))
						}

					}
				}
			}

			// Determine all the possible candidate relationships between this field and the related model.
			fieldCandidates := query.GetRelationshipCandidates(asts, currentModel, currentField)

			if len(fieldCandidates) > 0 {
				candidates[currentField] = fieldCandidates
			}

			if len(fieldCandidates) == 0 && currentField.Repeated {
				errs.AppendError(makeRelationshipError(
					fmt.Sprintf("The field '%s' does not have an associated field on the related %s model", currentField.Name.Value, currentField.Type.Value),
					fmt.Sprintf("In a one to many relationship, the related belongs-to field must exist on the %s model. %s", currentField.Type.Value, learnMore),
					currentField,
				))
			}
		},
	}
}

func makeRelationshipError(message string, hint string, node node.ParserNode) *errorhandling.ValidationError {
	return errorhandling.NewValidationErrorWithDetails(
		errorhandling.RelationshipError,
		errorhandling.ErrorDetails{
			Message: message,
			Hint:    hint,
		},
		node,
	)
}
