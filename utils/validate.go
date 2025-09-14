package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-playground/validator/v10"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"reflect"
	"strings"
	"sync"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	// register jsonschema validation
	validate.RegisterValidation("jsonschema", func(fl validator.FieldLevel) bool {
		schemaDef, ok := fl.Field().Interface().(json.RawMessage)
		if !ok {
			return false
		}
		err := jsonschemaValidate(string(schemaDef))
		if err != nil {
			return false
		}
		return true
	})
}

var mux sync.RWMutex
var formatters = make(map[string]func(fe validator.FieldError) string)

func init() {
	RegisterFormatter("required", func(fe validator.FieldError) string {
		return "required field missing"
	})
	RegisterFormatter("oneof", func(fe validator.FieldError) string {
		return fmt.Sprintf("invalid value: %s", fe.Value())
	})
	RegisterFormatter("gt", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be > %s", fe.Param())
	})
	RegisterFormatter("gte", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be >= %s", fe.Param())
	})
	RegisterFormatter("lt", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be < %s", fe.Param())
	})
	RegisterFormatter("lte", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be <= %s", fe.Param())
	})
	RegisterFormatter("min", func(fe validator.FieldError) string {
		return fmt.Sprintf("length must be at least %s", fe.Param())
	})
	RegisterFormatter("max", func(fe validator.FieldError) string {
		return fmt.Sprintf("length must be at most %s", fe.Param())
	})
	RegisterFormatter("jsonschema", func(fe validator.FieldError) string {
		schemaDef, ok := fe.Value().(json.RawMessage)
		if !ok {
			return "invalid jsonschema"
		}
		err := jsonschemaValidate(string(schemaDef))
		if err != nil {
			return err.Error()
		}
		return "invalid jsonschema"
	})
}

func jsonschemaValidate(defs string) error {
	schema := &openapi3.Schema{}
	err := schema.UnmarshalJSON([]byte(defs))
	if err != nil {
		return err
	}
	err = schema.Validate(context.Background(), openapi3.EnableSchemaFormatValidation())
	if err != nil {
		return err
	}
	return nil
}

func RegisterValidation(tag string, fn validator.Func) {
	err := validate.RegisterValidation(tag, fn)
	if err != nil {
		panic(err)
	}
}

func RegisterFormatter(tag string, fn func(fe validator.FieldError) string) {
	mux.Lock()
	defer mux.Unlock()
	formatters[tag] = fn
}

func Validate(v interface{}) error {
	err := validate.Struct(v)
	if err != nil {
		validateErr := errs.NewValidateError(errs.ErrRequestValidation)
		for _, e := range err.(validator.ValidationErrors) {
			fields := strings.Split(e.Namespace(), ".")
			node := validateErr.Fields
			for i := 1; i < len(fields); i++ {
				fieldName := fields[i]
				if i < len(fields)-1 {
					if node[fieldName] == nil {
						node[fieldName] = make(map[string]interface{})
					}
					node = node[fieldName].(map[string]interface{})
				} else {
					node[fieldName] = formatError(e)
				}
			}
		}
		return validateErr
	}
	return nil
}

func formatError(fe validator.FieldError) string {
	mux.RLock()
	defer mux.RUnlock()
	if formatter, ok := formatters[fe.Tag()]; ok {
		return formatter(fe)
	}
	return fe.Error()
}
