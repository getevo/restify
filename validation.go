package restify

import (
	"github.com/getevo/evo/v2/lib/validation"
	"strings"
)

type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (context *Context) Validate(ptr any) bool {
	errs := validation.Struct(ptr)
	if len(errs) > 0 {

		context.Response.Success = false
		context.Code = 412
		for _, item := range errs {
			var chunks = strings.SplitN(item.Error(), " ", 2)
			var v = ValidationError{
				Field: chunks[0],
			}
			if len(chunks) > 1 {
				v.Error = chunks[1]
			}
			context.Response.ValidationError = append(context.Response.ValidationError, v)
		}

		return false
	}

	return true
}
