package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/db/schema"
	"github.com/getevo/postman"
	"math"
)

var Prefix = "/admin/rest"

var permissionHandler func(permissions Permissions, context *Context) bool
var collection *postman.Collection

type App struct{}

func (app App) Register() error {
	if !db.Enabled {
		return fmt.Errorf("database is not enabled. restify plugin cannot be registered without a running database. please enable the database in your configuration file")
	}
	collection = postman.NewCollection("Restify", "")

	if postmanAuthType != "none" {
		collection.Auth = &postman.Auth{
			Type: postman.AuthType(postmanAuthType),
		}
	}
	return nil
}

func (app App) Router() error {

	return nil
}

func (app App) WhenReady() error {
	for idx, _ := range schema.Models {
		var model = schema.Models[idx]
		UseModel(model.Sample)
	}
	app.registerHooks()
	var controller Controller
	for idx, _ := range Resources {
		for i, _ := range Resources[idx].Actions {
			Resources[idx].Actions[i].RegisterRouter()
		}
	}
	evo.Get(Prefix+"/models", controller.ModelsHandler)
	if postmanRegistered {
		evo.Get(Prefix+"/postman", controller.PostmanHandler)
	}
	return nil
}

func (app App) Priority() application.Priority {
	return math.MaxInt32
}

func (app App) Name() string {
	return "restify"
}
