package jsonschema_validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/utils"
)

type Config struct {
	Draft         int                        `json:"draft" validate:"required,oneof=6"`
	DefaultSchema string                     `json:"default_schema" validate:"omitempty,json,max=1048576"`
	Schemas       map[string]*SchemaResource `json:"schemas" validate:"dive,required"`
}

type SchemaResource struct {
	Schema string `json:"schema" validate:"omitempty,json,max=1048576"`
}

type SchemaValidatorPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &SchemaValidatorPlugin{}
	p.Name = "jsonschema-validator"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func validateSchema(schema string) (*openapi3.Schema, error) {
	openapiSchema := &openapi3.Schema{}
	err := openapiSchema.UnmarshalJSON([]byte(schema))
	if err != nil {
		return nil, fmt.Errorf("value must be a valid jsonschema")
	}
	err = openapiSchema.Validate(context.Background(), openapi3.EnableSchemaFormatValidation())
	if err != nil {
		return openapiSchema, err
	}
	return openapiSchema, nil
}

func (p *SchemaValidatorPlugin) ValidateConfig() error {
	err := utils.Validate(p.Config)
	if err != nil {
		return err
	}

	e := errs.NewValidateError(errors.New("request validation"))

	var defaultSchema *openapi3.Schema
	if p.Config.DefaultSchema != "" {
		defaultSchema, err = validateSchema(p.Config.DefaultSchema)
		if err != nil {
			e.Fields = map[string]interface{}{
				"default_schema": err,
			}
		}
	}

	for event, schema := range p.Config.Schemas {
		field := fmt.Sprintf("schemas[%s]", event)
		if schema == nil || schema.Schema == "" {
			err = defaultSchema.Validate(context.Background(), openapi3.EnableSchemaFormatValidation())
			if err != nil {
				e.Fields[field] = map[string]string{
					"schema": err.Error(),
				}
				return e
			}
		} else {
			_, err = validateSchema(schema.Schema)
			if err != nil {
				e.Fields[field] = map[string]string{
					"schema": err.Error(),
				}
				return e
			}
		}
	}
	return nil
}

func (p *SchemaValidatorPlugin) ExecuteInbound(inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	var event map[string]any
	body := inbound.RawBody
	if err = json.Unmarshal(body, &event); err != nil {
		return
	}

	eventType, ok := event["event_type"].(string)
	if !ok || eventType == "" {
		res.Payload = body
		return
	}

	data := event["data"]
	if data == nil {
		res.Payload = body
		return
	}

	schema, ok := p.Config.Schemas[eventType]
	if !ok || schema == nil {
		res.Payload = body
		return
	}

	validator := jsonschema.New([]byte(schema.Schema))
	e := validator.Validate(&jsonschema.ValidatorContext{
		HTTPRequest: &jsonschema.HTTPRequest{
			R:    inbound.Request,
			Data: data.(map[string]any),
		},
	})
	if e != nil {
		response.JSON(inbound.Response, 400, types.ErrorResponse{
			Message: "Request Validation",
			Error:   e,
		})
		res.Terminated = true
		return
	}
	res.Payload = body
	return
}
