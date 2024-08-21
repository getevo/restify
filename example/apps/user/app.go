package user

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/restify"
)

type App struct{}

func (app App) Register() error {
	restify.SetPrefix("/admin/rest")
	restify.EnablePostman()

	//restify.SetPostmanAuthorization(restify.AuthTypeHeader, "username", "password") //force user and password authentication in header
	restify.SetPostmanAuthorization(restify.AuthTypeBasic) // force basic authentication on Restify collection
	db.UseModel(User{}, Order{}, Product{}, Article{})
	evo.GetDBO().AutoMigrate(Article{}, User{}, Order{}, Product{})
	return nil
}

func (app App) Router() error {

	return nil
}

func (app App) WhenReady() error {

	return nil
}

func (app App) Priority() application.Priority {
	return application.DEFAULT
}

func (app App) Name() string {
	return "user"
}
