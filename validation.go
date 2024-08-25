package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/validation"
)

type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (context *Context) Validate(ptr any) error {
	errs := validation.Struct(ptr)
	if len(errs) > 0 {
		context.AddValidationErrors(errs...)
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (context *Context) ValidateNonZeroFields(ptr any) error {
	errs := validation.StructNonZeroFields(ptr)
	if len(errs) > 0 {
		context.AddValidationErrors(errs...)
		return fmt.Errorf("validation failed")
	}
	return nil
}
