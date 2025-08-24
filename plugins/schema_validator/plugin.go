package schema_validator

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/schema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/utils"
)

type Config struct {
	EventSchemas map[string]string `json:"event_schemas" validate:"dive,required"`
}

type SchemaValidatorPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &SchemaValidatorPlugin{}
	p.Name = "schema_validator"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *SchemaValidatorPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *SchemaValidatorPlugin) ExecuteInbound(inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	jsonchemaValidator := jsonschema.New(p.Config.)
	if err != nil {
		return
	}
	result.Payload = req.Body
	return
}
