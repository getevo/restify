package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/db/schema"
	"math"
)

var Prefix = "/admin/rest"

type App struct{}

func (app App) Register() error {
	if !db.Enabled {
		return fmt.Errorf("database is not enabled. restify plugin cannot be registered without a running database. please enable the database in your configuration file")
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
	var controller Controller
	for idx, _ := range resources {
		for i, _ := range resources[idx].Actions {
			resources[idx].Actions[i].RegisterRouter()
		}
	}
	evo.Get(Prefix+"/models", controller.Models)
	return nil
}

func (app App) Priority() application.Priority {
	return math.MaxInt32
}

func (app App) Name() string {
	return "restify"
}
