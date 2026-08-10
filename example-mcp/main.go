// Example application exposing SQLite tables to AI agents, over MCP, without
// writing a tool by hand.
//
// Every model that carries restify.MCP is published as five tools —
// list, get, create, update and delete — built from the model definition and
// served by the same handlers that answer the REST routes.
//
// Run it:
//
//	go run .
//
// Then list the tools:
//
//	curl -s http://127.0.0.1:8080/mcp -H 'Content-Type: application/json' \
//	  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
//
// And call one:
//
//	curl -s http://127.0.0.1:8080/mcp -H 'Content-Type: application/json' \
//	  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"invoice_list",
//	       "arguments":{"filters":[{"field":"status","op":"eq","value":"paid"}]}}}'
package main

import (
	"fmt"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/restify"
)

type App struct{}

func (App) Register() error {
	db.UseModel(Customer{}, Invoice{}, AuditLog{})
	if err := evo.GetDBO().AutoMigrate(Customer{}, Invoice{}, AuditLog{}); err != nil {
		return err
	}
	return seed()
}

func (App) Router() error    { return nil }
func (App) WhenReady() error { return nil }
func (App) Name() string     { return "shop" }

func (App) Priority() application.Priority { return application.DEFAULT }

// seed puts a couple of rows in place so that a read tool has something to
// return on a fresh database.
func seed() error {
	var count int64
	if err := db.Model(&Customer{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var customers = []Customer{
		{Name: "Acme Spa", Email: "billing@acme.example", Country: "IT"},
		{Name: "Globex Srl", Email: "ap@globex.example", Country: "IT"},
		{Name: "Initech Ltd", Email: "finance@initech.example", Country: "GB"},
	}
	if err := db.Create(&customers).Error; err != nil {
		return err
	}

	var invoices = []Invoice{
		{CustomerID: customers[0].ID, Reference: "INV-1001", Total: 240.50, Status: "paid", IssuedAt: time.Now().Add(-72 * time.Hour)},
		{CustomerID: customers[0].ID, Reference: "INV-1002", Total: 99.00, Status: "sent", IssuedAt: time.Now().Add(-48 * time.Hour)},
		{CustomerID: customers[1].ID, Reference: "INV-1003", Total: 1500.00, Status: "paid", IssuedAt: time.Now().Add(-24 * time.Hour)},
		{CustomerID: customers[2].ID, Reference: "INV-1004", Total: 42.00, Status: "draft", IssuedAt: time.Now()},
	}
	return db.Create(&invoices).Error
}

func main() {
	if err := evo.Setup(sqliteDriver{}); err != nil {
		fmt.Println("setup failed:", err)
		return
	}

	evo.Register(restify.App{}, App{})

	if err := evo.Run(); err != nil {
		fmt.Println("run failed:", err)
	}
}
