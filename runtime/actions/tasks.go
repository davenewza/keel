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

func GetTask(scope *Scope, input map[string]any) (map[string]any, error) {
	ctx, span := tracer.Start(scope.Context, "Get Task")
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
	query.AppendSelect(AllFields())

	result, err := query.SelectStatement().ExecuteToSingle(ctx)
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

func CreateTask(scope *Scope, input map[string]any) (map[string]any, error) {
	_, span := tracer.Start(scope.Context, "Create Task")
	defer span.End()

	var err error
	typedInput := typed.New(input)

	topicType := typedInput.String("type")
	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)

	query := NewQuery(taskModel)
	query.AddWriteValue(Field("type"), Value(topicType))
	query.AppendReturning(AllFields())
	statement := query.InsertStatement(scope.Context)

	newTask, err := statement.ExecuteToSingle(scope.Context)
	if err != nil {
		return nil, err
	}

	fieldsModel := proto.FindModel(scope.Schema.Models, fieldsModelName(topicType))
	fieldsQuery := NewQuery(fieldsModel)

	for _, v := range fieldsModel.Fields {
		value, has := input[v.Name]
		if has {
			fieldsQuery.AddWriteValue(Field(v.Name), Value(value))
			continue
		}
		if v.Name == parser.TaskFieldNameTaskId {
			fieldsQuery.AddWriteValue(Field(parser.TaskFieldNameTaskId), Value(newTask["id"]))
		}
	}

	query.AppendReturning(AllFields())
	statement = fieldsQuery.InsertStatement(scope.Context)

	newFields, err := statement.ExecuteToSingle(scope.Context)
	if err != nil {
		return nil, err
	}

	newTask["fields"] = newFields

	return newTask, nil
}

func GetNextTask(scope *Scope, input map[string]any) (map[string]any, error) {
	_, span := tracer.Start(scope.Context, "Get Next Task")
	defer span.End()

	//query := NewQuery(parser.TaskModelName)

	// Open or assigned to me
	// ordered by createdAt
	// where deferredy is null or < now()

	return nil, nil
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

	inputs, err := getInputsForTask(scope, result["id"].(string), result["type"].(string))
	if err != nil {
		return nil, err
	}
	if inputs != nil {
		result["inputs"] = inputs
	}

	fields, err := getFieldsForTask(scope, result["id"].(string), result["type"].(string))
	if err != nil {
		return nil, err
	}
	if fields != nil {
		result["fields"] = fields
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

func ListTopics(scope *Scope, _ map[string]any) (map[string]any, error) {
	ctx, span := tracer.Start(scope.Context, "ListTopics")
	defer span.End()

	taskModel := proto.FindModel(scope.Schema.Models, parser.TaskModelName)
	if taskModel == nil {
		return nil, errors.New("tasks are not enabled for this project")
	}

	query := NewQuery(taskModel)
	query.AppendSelect(Field(parser.TaskFieldNameType))
	query.AppendSelect(Raw("count(*)"))
	query.AppendGroupBy(Field(parser.TaskFieldNameType))

	result, _, err := query.SelectStatement().ExecuteToMany(ctx, nil)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return map[string]any{
		"topics": result,
	}, nil
}

// fieldsModelName returns the model name for the task of the given type/topic
func fieldsModelName(topic string) string {
	return fmt.Sprintf("%sFields", topic)
}

// inputsModelName returns the model name for the task of the given type/topic
func inputsModelName(topic string) string {
	return fmt.Sprintf("%sInputs", topic)
}

// getInputsForTask will return the inputs model for the given task
func getInputsForTask(scope *Scope, taskId string, taskType string) (map[string]any, error) {
	inputsModel := proto.FindModel(scope.Schema.Models, inputsModelName(taskType))
	inputsQuery := NewQuery(inputsModel)
	err := inputsQuery.Where(Field(parser.TaskFieldNameTaskId), Equals, Value(taskId))
	if err != nil {
		return nil, err
	}
	return inputsQuery.SelectStatement().ExecuteToSingle(scope.Context)
}

// getFieldsForTask will return the fields model for the given task
func getFieldsForTask(scope *Scope, taskId string, taskType string) (map[string]any, error) {
	fieldsModel := proto.FindModel(scope.Schema.Models, fieldsModelName(taskType))
	fieldsQuery := NewQuery(fieldsModel)
	err := fieldsQuery.Where(Field(parser.TaskFieldNameTaskId), Equals, Value(taskId))
	if err != nil {
		return nil, err
	}
	return fieldsQuery.SelectStatement().ExecuteToSingle(scope.Context)
}
