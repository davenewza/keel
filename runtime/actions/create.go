package actions

import (
	"github.com/teamkeel/keel/runtime/common"
)

func Create(scope *Scope, input map[string]any) (res map[string]any, err error) {
	query := NewQuery(scope.model)

	// Begin a transaction and defer rollback which will run if a commit hasn't occurred.
	query.Begin(scope.context)
	// Defer ensures a rollback as the function may return early due to an error.
	defer func() {
		if err != nil {
			query.Rollback(scope.context)
		}
	}()

	// Generate the SQL statement
	statement, err := GenerateCreateStatement(query, scope, input)
	if err != nil {
		return nil, err
	}

	// Execute database request, expecting a single result
	result, err := statement.ExecuteToSingle(scope.context)
	if err != nil {
		return nil, err
	}

	// Retrieve the newly created row so we can check permissions
	query.Where(IdField(), Equals, Value(result["id"]))

	// Check permissions and roles conditions
	isAuthorised, err := query.isAuthorised(scope, input)
	if err != nil {
		return nil, err
	}

	if !isAuthorised {
		return nil, common.RuntimeError{Code: common.ErrPermissionDenied, Message: "not authorized to access this operation"}
	}

	err = query.Commit(scope.context)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GenerateCreateStatement(query *QueryBuilder, scope *Scope, input map[string]any) (*Statement, error) {
	defaultValues, err := initialValueForModel(scope.model, scope.schema)
	if err != nil {
		return nil, err
	}

	query.AddWriteValues(defaultValues)

	err = query.captureWriteValues(scope, input)
	if err != nil {
		return nil, err
	}

	err = query.captureSetValues(scope, input)
	if err != nil {
		return nil, err
	}

	// Return the inserted row
	query.AppendReturning(AllFields())

	return query.InsertStatement(), nil
}
