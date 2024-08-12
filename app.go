package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
)

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
	return nil
}

func (app App) Priority() application.Priority {
	return application.LOWEST
}

func (app App) Name() string {
	return "restify"
}
