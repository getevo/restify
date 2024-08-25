package user

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/restify"
)

type App struct{}

func (app App) Register() error {
	restify.SetPrefix("/admin/rest")
	restify.EnablePostman()

	restify.SetPostmanAuthorization(restify.AuthTypeBasic) // force basic authentication on Restify collection

	//global hooks
	restify.OnBeforeSave(func(obj any, c *restify.Context) error {

		if user, ok := obj.(*User); ok {
			if user.Username == "unallowed" {
				c.AddValidationErrors(fmt.Errorf("this username is not allowed"))
				return fmt.Errorf("validation error")
			}
		}

		fmt.Println("Global OnAfterSave Hook Called!")
		return nil
	})

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
