package restify

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/outcome"
)

// Controller represents a controller type.
type Controller struct{}

// ModelsHandler returns all registered models.
func (c Controller) ModelsHandler(request *evo.Request) interface{} {
	return Resources
}

func (c Controller) PostmanHandler(request *evo.Request) any {
	b, err := collection.ToJson()
	if err != nil {
		return err
	}
	return outcome.Response{
		StatusCode:  200,
		ContentType: "application/json",
		Data:        b,
		Headers: map[string]string{
			"Content-Disposition": "attachment; filename=postman_collection.json",
		},
	}
}
