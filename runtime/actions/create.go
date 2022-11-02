package actions

import (
	"errors"

	"github.com/samber/lo"
	"github.com/teamkeel/keel/proto"
)

type CreateAction struct {
	scope *Scope
}

type CreateResult struct {
	Object map[string]any `json:"object"`
}

func (action *CreateAction) Initialise(scope *Scope) ActionBuilder[CreateResult] {
	action.scope = scope
	return action
}

func (action *CreateAction) ApplyExplicitFilters(args WhereArgs) ActionBuilder[CreateResult] {
	return action // no-op
}

func (action *CreateAction) ApplyImplicitFilters(args WhereArgs) ActionBuilder[CreateResult] {
	return action // no-op
}

func (action *CreateAction) IsAuthorised(args WhereArgs) ActionBuilder[CreateResult] {
	if action.scope.Error != nil {
		return action
	}

	isAuthorised, err := DefaultIsAuthorised(action.scope, args)

	if err != nil {
		action.scope.Error = err
		return action
	}

	if !isAuthorised {
		action.scope.Error = errors.New("not authorized to access this operation")
	}

	return action
}

var ExcludedCreateKeys = []string{"created_at", "updated_at", "id"}

func (action *CreateAction) Execute(args WhereArgs) (*ActionResult[CreateResult], error) {
	if action.scope.Error != nil {
		return nil, action.scope.Error
	}

	op := action.scope.operation

	if op.Implementation == proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM {
		values := map[string]any{}
		for key, value := range action.scope.writeValues {
			if lo.Contains(ExcludedCreateKeys, key) {
				continue
			}
			values[key] = value
		}

		return ParseCreateObjectResponse(action.scope.context, op, values)
	}

	err := action.scope.query.WithContext(action.scope.context).Create(action.scope.writeValues).Error
	if err != nil {
		action.scope.Error = err
		return nil, err
	}

	// todo: Use RETURNING statement on INSERT
	// https://linear.app/keel/issue/RUN-146/gorm-use-returning-on-insert-and-update-statements
	result := toLowerCamelMap(action.scope.writeValues)

	return &ActionResult[CreateResult]{
		Value: CreateResult{
			Object: result,
		},
	}, nil
}

func (action *CreateAction) CaptureImplicitWriteInputValues(args ValueArgs) ActionBuilder[CreateResult] {
	if action.scope.Error != nil {
		return action
	}

	// initialise default values
	values, err := initialValueForModel(action.scope.model, action.scope.schema)
	if err != nil {
		action.scope.Error = err
		return action
	}
	action.scope.writeValues = values

	// Delegate to a method that we hope will become more widely used later.
	if err := DefaultCaptureImplicitWriteInputValues(action.scope.operation.Inputs, args, action.scope); err != nil {
		action.scope.Error = err
		return action
	}
	return action
}

func (action *CreateAction) CaptureSetValues(args ValueArgs) ActionBuilder[CreateResult] {
	if action.scope.Error != nil {
		return action
	}

	if err := DefaultCaptureSetValues(action.scope, args); err != nil {
		action.scope.Error = err
		return action
	}
	return action
}
