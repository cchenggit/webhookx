package schema_validator

import (
	"github.com/webhookx-io/webhookx/db/entities"
	"net/http"
)

type Validator interface {
	Validate(ctx *ValidatorContext) error
}

type ValidatorContext struct {
	HTTPRequest *HTTPRequest

	Workspace *entities.Workspace
	Source    *entities.Source
	Event     *entities.Event
}

type HTTPRequest struct {
	R    *http.Request
	Body []byte
}

type HTTPResponse struct {
	Code    int
	Headers map[string]string
	Body    string
}

type ValidateResult struct {
	ReturnValue  interface{}
	HTTPResponse *HTTPResponse
}
