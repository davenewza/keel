package actions

import (
	"context"

	"github.com/segmentio/ksuid"
	"github.com/teamkeel/keel/proto"
	"github.com/teamkeel/keel/runtime/runtimectx"

	"github.com/iancoleman/strcase"
)

type Identity struct {
	Id       ksuid.KSUID
	Email    string
	Password string
}

// Authenticate will return the identity if it is successfully authenticated or when a new identity is created.
func Authenticate(ctx context.Context, schema *proto.Schema, model *proto.Model, args map[string]any) (*Identity, error) {
	db, err := runtimectx.GetDB(ctx)
	if err != nil {
		return nil, err
	}

	identity := Identity{
		Email:    args["emailPassword"].(map[string]interface{})["email"].(string),
		Password: args["emailPassword"].(map[string]interface{})["password"].(string),
	}

	var record map[string]any
	result := db.Table(strcase.ToSnake(model.Name)).Limit(1).Where("email", identity.Email).Find(&record)

	if result.Error != nil {
		return nil, result.Error
	}

	if record != nil {
		id, err := ksuid.Parse(record["id"].(string))

		if err != nil {
			return nil, err
		}

		identity.Id = id

		authenticated := identity.Email == record["email"] && identity.Password == record["password"]
		if authenticated {
			return &identity, nil
		} else {
			return nil, nil
		}
	} else if args["createIfNotExists"] == true {
		modelMap, err := initialValueForModel(model, schema)
		if err != nil {
			return nil, err
		}

		modelMap[strcase.ToSnake("email")] = identity.Email
		modelMap[strcase.ToSnake("password")] = identity.Password

		if err := db.Table(strcase.ToSnake(model.Name)).Create(modelMap).Error; err != nil {
			return nil, err
		}

		identity.Id = modelMap["id"].(ksuid.KSUID)

		return &identity, nil
	}

	return nil, nil
}