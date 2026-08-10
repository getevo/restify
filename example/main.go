package main

import (
	"example/apps/user"
	"github.com/getevo/evo/v2"
	"github.com/getevo/restify"
)

func main() {
	if err := evo.Setup(sqliteDriver{}); err != nil {
		panic(err)
	}

	evo.Register(restify.App{}, user.App{})

	evo.Run()
}
