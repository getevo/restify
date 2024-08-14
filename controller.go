package restify

import "github.com/getevo/evo/v2"

// Controller represents a controller type.
type Controller struct{}

// Models returns a collection of resources.
func (c Controller) Models(request *evo.Request) interface{} {
	return resources
}
