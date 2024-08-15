package main

import (
	"example/apps/user"
	"github.com/getevo/evo/v2"
	"github.com/getevo/restify"
)

func main() {
	evo.Setup()

	evo.Register(restify.App{}, user.App{})

	evo.Run()
}
