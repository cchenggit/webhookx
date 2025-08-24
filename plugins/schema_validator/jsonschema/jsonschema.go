package jsonschema

import (
	"context"
	"github.com/kaptinlin/jsonschema"
	"github.com/webhookx-io/webhookx/plugins/schema_validator"
)

type JSONSchema struct {
	compiler  *jsonschema.Compiler
	opts      *Options
	schemaDef string
}

type Options struct {
	Timeout int
}

func New(schemaDef string, opts *Options) *JSONSchema {
	return &JSONSchema{
		compiler:  jsonschema.NewCompiler(),
		schemaDef: schemaDef,
		opts:      opts,
	}
}

func (s *JSONSchema) Validate(ctx context.Context, v *schema_validator.ValidatorContext) error {
	schema, err := s.compiler.CompileBatch([]byte(s.schemaDef))
	if err != nil {
		return err
	}

	result := schema.ValidateJSON(v.HTTPRequest.Body)
	if result.IsValid() {
		return nil
	} else {
		return result
	}
}
