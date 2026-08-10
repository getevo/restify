package main

import (
	"time"

	"github.com/getevo/evo/v2/lib/db/types"
	"github.com/getevo/restify"
)

// Customer is reachable over REST and over MCP: it carries both markers.
//
// It also carries the column types a real model uses — a string array, a JSON
// document, the managed timestamps and soft delete — which are the types that
// cannot be described to a model by walking their Go fields. They are worth
// having here because that is exactly what the generated schema has to get
// right.
type Customer struct {
	ID      uint              `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name    string            `gorm:"column:name;size:120" validation:"required" description:"Full name of the customer" json:"name"`
	Email   string            `gorm:"column:email;size:190;uniqueIndex" validation:"required,email" description:"Contact email" json:"email"`
	Country string            `gorm:"column:country;size:2" description:"ISO 3166-1 alpha-2 country code" json:"country"`
	Tags    types.StringArray `gorm:"column:tags;type:text" description:"Free form labels" json:"tags"`

	types.CreatedAt
	types.UpdatedAt
	types.SoftDelete

	restify.API
	restify.MCP
}

func (Customer) TableName() string { return "customer" }

// Invoice is also exposed to agents, and is deliberately not deletable: the
// DisableDelete marker removes the REST route and the MCP tool alike.
type Invoice struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CustomerID uint      `gorm:"column:customer_id;index" validation:"required" description:"Customer this invoice belongs to" json:"customer_id"`
	Reference  string    `gorm:"column:reference;size:32;uniqueIndex" validation:"required" description:"Human readable invoice reference, for example INV-1001" json:"reference"`
	Total      float64   `gorm:"column:total" description:"Invoice total in euro" json:"total"`
	Status     string    `gorm:"column:status;size:16" validation:"required,in(draft,sent,paid)" description:"Lifecycle state of the invoice" json:"status"`
	IssuedAt   time.Time `gorm:"column:issued_at" description:"When the invoice was issued" json:"issued_at"`

	Customer *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`

	restify.API
	restify.MCP
	restify.DisableDelete
}

func (Invoice) TableName() string { return "invoice" }

// AuditLog has REST endpoints but no MCP marker, so no agent ever sees it.
// This is what keeps a server's tool list to the models that are meant to be
// reachable rather than every table in the schema.
type AuditLog struct {
	ID      uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Action  string    `gorm:"column:action;size:64" json:"action"`
	At      time.Time `gorm:"column:at" json:"at"`
	Details string    `gorm:"column:details;type:text" json:"details"`

	restify.API
}

func (AuditLog) TableName() string { return "audit_log" }
