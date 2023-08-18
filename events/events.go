package events

import (
	"context"
	"fmt"
	"time"
)

type handlerContextKey string

var contextKey handlerContextKey = "eventHandler"

func WithEventHandler(ctx context.Context, handler EventHandler) context.Context {
	return context.WithValue(ctx, contextKey, handler)
}

func HasEventHandler(ctx context.Context) bool {
	return ctx.Value(contextKey) != nil
}

func GetEventHandler(ctx context.Context) (EventHandler, error) {
	v, ok := ctx.Value(contextKey).(EventHandler)
	if !ok {
		return nil, fmt.Errorf("context does not have key or is not EventHandler: %s", contextKey)
	}
	return v, nil
}

// The event payload.
// TODO: Iron out the data.
type Event struct {
	Name       string         `json:"name,omitempty"`
	Model      string         `json:"model,omitempty"`
	OccurredAt time.Time      `json:"occurred_at,omitempty"`
	IdentityId string         `json:"identity_id,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

// The event handler function to be executed for each event generated.
type EventHandler func(ctx context.Context, event *Event) error

// Gather, create and send events which have occurred within the scope of this context.
func GenerateEvents(ctx context.Context) error {

	// 1. Retrieve rows from the audit table by this ctx's trace_id
	// 2. Do we have any events in the schema matching these rows?
	// 3. If so, call handleEvent for each subscriber of that event with the payload.

	if !HasEventHandler(ctx) {
		return nil
	}

	// PLACEHOLDER CODE
	handler, _ := GetEventHandler(ctx)

	testEvent := &Event{
		Name:       "member.created",
		Model:      "member",
		OccurredAt: time.Now(),
		IdentityId: "",
		Data: map[string]any{
			"id":   "123",
			"name": "Boetie",
		},
	}

	_ = handler(ctx, testEvent)

	return nil
}
