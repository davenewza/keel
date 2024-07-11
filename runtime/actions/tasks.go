package actions

import (
	"errors"
	"fmt"
	"time"

	"github.com/karlseguin/typed"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/auth"
	"github.com/teamkeel/keel/runtime/common"
	"github.com/teamkeel/keel/schema/parser"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func CreateTask(scope *Scope, input map[string]any) (map[string]any, error) {
	var err error
	typedInput := typed.New(input)

	topic := typedInput.String("topic")
	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)
	if taskModel == nil {
		return nil, errors.New("topic does not exist")
	}

	input = map[string]any{
		"typeType": "Type",
		"status":   "New",
	}

	query := NewQuery(taskModel)
	err = query.captureWriteValues(scope, input)
	if err != nil {
		return nil, err
	}
	query.AppendReturning(AllFields())
	statement := query.InsertStatement(scope.Context)

	newTask, err := statement.ExecuteToSingle(scope.Context)
	if err != nil {
		return nil, err
	}

	_ = proto.FindModel(scope.Schema.Models, topic) //

	return newTask, nil
}

func CancelTask(scope *Scope, input map[string]any) (map[string]any, error) {
	ctx, span := tracer.Start(scope.Context, "Cancel Task")
	defer span.End()

	typedInput := typed.New(input)
	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)
	if taskModel == nil {
		return nil, errors.New("tasks are not enabled for this project")
	}

	identity, err := auth.GetIdentity(ctx)
	if err != nil {
		return nil, common.NewPermissionError()
	}

	query := NewQuery(taskModel)
	err = query.Where(IdField(), Equals, Value(typedInput.String("id")))
	if err != nil {
		return nil, fmt.Errorf("applying sql where: %w", err)
	}

	query.AddWriteValues(map[string]*QueryOperand{
		parser.TaskFieldNameStatus:       Value(parser.TaskStatusCancelled),
		parser.TaskFieldNameResolvedById: Value(identity["id"]),
		parser.TaskFieldNameResolvedAt:   Value(time.Now()),
	})
	query.AppendSelect(AllFields())
	query.AppendReturning(AllFields())

	result, err := query.UpdateStatement(ctx).ExecuteToSingle(ctx)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if result == nil {
		return nil, common.NewNotFoundError()
	}

	return result, nil
}

func DeferTask(scope *Scope, input map[string]any) (map[string]any, error) {
	ctx, span := tracer.Start(scope.Context, "Defer Task")
	defer span.End()

	typedInput := typed.New(input)
	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)
	if taskModel == nil {
		return nil, errors.New("tasks are not enabled for this project")
	}

	query := NewQuery(taskModel)
	err := query.Where(IdField(), Equals, Value(typedInput.String("id")))
	if err != nil {
		return nil, fmt.Errorf("applying sql where: %w", err)
	}
	query.AddWriteValues(map[string]*QueryOperand{
		parser.TaskFieldNameDeferredUntil: Value(input[parser.TaskFieldNameDeferredUntil]),
		parser.TaskFieldNameStatus:        Value(parser.TaskStatusDeferred),
	})
	query.AppendSelect(AllFields())
	query.AppendReturning(AllFields())

	result, err := query.UpdateStatement(ctx).ExecuteToSingle(ctx)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if result == nil {
		return nil, common.NewNotFoundError()
	}

	return result, nil
}

func AssignTask(scope *Scope, input map[string]any) (map[string]any, error) {
	ctx, span := tracer.Start(scope.Context, "Assign Task")
	defer span.End()

	typedInput := typed.New(input)
	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)
	if taskModel == nil {
		return nil, errors.New("tasks are not enabled for this project")
	}

	query := NewQuery(taskModel)
	err := query.Where(IdField(), Equals, Value(typedInput.String("id")))
	if err != nil {
		return nil, fmt.Errorf("applying sql where: %w", err)
	}
	query.AddWriteValues(map[string]*QueryOperand{
		parser.TaskFieldNameAssignedToId: Value(typedInput.String(parser.TaskFieldNameAssignedToId)),
		parser.TaskFieldNameAssignedAt:   Value(time.Now()),
		parser.TaskFieldNameStatus:       Value(parser.TaskStatusAssigned),
	})
	query.AppendSelect(AllFields())
	query.AppendReturning(AllFields())

	result, err := query.UpdateStatement(ctx).ExecuteToSingle(ctx)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if result == nil {
		return nil, common.NewNotFoundError()
	}

	return result, nil
}
