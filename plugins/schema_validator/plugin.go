package schema_validator

import (
	"encoding/json"
	"errors"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/schema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/utils"
)

type Config struct {
	EventSchemas    []*EventTypeSchema         `json:"event_schemas" validate:"dive"`
	EventSchemasMap map[string]json.RawMessage `json:"-"`
}

type EventTypeSchema struct {
	EventType  string          `json:"event_type" validate:"required,max=100"`
	JSONSchema json.RawMessage `json:"jsonschema" validate:"required,jsonschema,max=1048576"`
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
	p.Config.EventSchemasMap = make(map[string]json.RawMessage)
	for _, es := range p.Config.EventSchemas {
		p.Config.EventSchemasMap[es.EventType] = es.JSONSchema
	}

	return p, nil
}

func (p *SchemaValidatorPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *SchemaValidatorPlugin) ExecuteInbound(inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	// parse body to get event type
	var event map[string]any
	body := inbound.RawBody
	if err = json.Unmarshal(body, &event); err != nil {
		return
	}

	eventType, ok := event["event_type"].(string)
	if !ok || eventType == "" {
		return res, errors.New("invalid event_type field")
	}

	data := event["data"]
	if data == nil {
		return res, errors.New("missing data field")
	}

	schemaDef, ok := p.Config.EventSchemasMap[eventType]
	if !ok || len(schemaDef) == 0 {
		// no schema defined for this event type, skip validation
		res.Payload = body
		return res, nil
	}

	req := &jsonschema.HTTPRequest{
		R:    inbound.Request,
		Data: data.(map[string]any),
	}
	jsonchemaValidator := jsonschema.New(schemaDef)
	res.RequestError = jsonchemaValidator.Validate(&jsonschema.ValidatorContext{
		HTTPRequest: req,
	})
	res.Payload = body
	return
}
