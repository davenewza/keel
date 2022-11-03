package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/teamkeel/keel/proto"
)

type HttpFunctionsClient struct {
	Port  string
	Host  string
	Https bool
}

func (h *HttpFunctionsClient) Request(ctx context.Context, actionName string, opType proto.OperationType, body map[string]any) (any, error) {
	b, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	protocol := "http"
	if h.Https {
		protocol = "https"
	}

	baseUrl := h.Host
	if h.Port != "" {
		baseUrl = baseUrl + ":" + h.Port
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/%s", protocol, baseUrl, actionName), bytes.NewReader(b))

	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	b, err = io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var response interface{}

	err = json.Unmarshal(b, &response)
	if err != nil {
		return nil, errors.New("invalid json: " + string(b))
	}

	return response, nil
}

func (h *HttpFunctionsClient) ToGraphQL(ctx context.Context, response any, opType proto.OperationType) (interface{}, error) {
	responseMap, ok := response.(map[string]any)
	if !ok {
		fmt.Printf("unknown response from custom function runtime (not map): %v\n", responseMap)
		return nil, errors.New("unknown response from custom function runtime")
	}

	errs, hasErrors := responseMap["errors"].([]map[string]string)

	if ok && hasErrors && len(errs) > 0 {
		return nil, errors.New(errs[0]["message"])
	}

	// Handles returning a value / error given that different operations
	// return different response shapes.
	switch opType {
	case proto.OperationType_OPERATION_TYPE_CREATE, proto.OperationType_OPERATION_TYPE_GET, proto.OperationType_OPERATION_TYPE_UPDATE:
		object, hasObject := responseMap["object"]

		if !hasObject {
			fmt.Printf("unknown response from custom function runtime (no object): %v\n", responseMap)
			return nil, errors.New("unknown response from custom function runtime")
		}

		return object, nil
	case proto.OperationType_OPERATION_TYPE_LIST:
		collection, hasCollection := responseMap["collection"]

		if !hasCollection {
			fmt.Printf("unknown response from custom function runtime (no collection): %v\n", responseMap)
			return nil, errors.New("unknown response from custom function runtime")
		}

		return collection, nil
	case proto.OperationType_OPERATION_TYPE_DELETE:
		success, hasSuccess := responseMap["success"]

		if !hasSuccess {
			fmt.Printf("unknown response from custom function runtime (no success): %v\n", responseMap)
			return nil, errors.New("unknown response from custom function runtime")
		}

		return success, nil
	}

	return nil, fmt.Errorf("unsupported operation type %s", opType)
}