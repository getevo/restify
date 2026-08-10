# MCP Tools

Restify already knows your models: their columns, their types, their validation
rules, their permissions and their hooks. That is everything an AI agent needs to
read and edit them, so Restify can publish it as [MCP](https://modelcontextprotocol.io)
tools without you writing a tool by hand.

Add a marker to a model, turn the feature on, and the model is reachable by an
agent through the same handlers that answer its REST routes.

> **Requires a recent EVO.** The endpoint, the transport and the authentication
> belong to the framework, so this feature needs an EVO that has them. That
> version also removed the legacy `lib/model` package: a model still embedding
> `model.CreatedAt` or `model.DeletedAt` moves to `types.CreatedAt` and
> `types.SoftDelete` from `lib/db/types`. `types.SoftDelete` adds a `deleted`
> column next to `deleted_at`, so it is a schema change.

## Quick start

**1. Mark the models an agent may reach**

```go
type Customer struct {
    ID    uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Name  string `gorm:"column:name;size:120" validation:"required" description:"Full name" json:"name"`
    Email string `gorm:"column:email;size:190" validation:"required,email" json:"email"`

    restify.API
    restify.MCP
}
```

**2. Turn it on in `config.yml`**

```yaml
# The framework's MCP endpoint, which is disabled by default
MCP:
  enabled: true
  path: /mcp
  name: my-service
  token: a-secret-token

# Restify's tool generation
Restify:
  MCP:
    Enabled: true
```

**3. Point a client at it**

```shell
curl -s http://127.0.0.1:8080/mcp \
  -H 'Authorization: Bearer a-secret-token' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

A runnable application backed by SQLite is in [`example-mcp`](../example-mcp).
Every request and response on this page was captured from it and abridged for
readability — timestamps are shortened and long payloads elided — so what you
read here is what the server actually answers.

## Configuration

All keys live under the `Restify.MCP` section.

| Key | Default | Description |
|---|---|---|
| `Enabled` | `false` | Master switch. Nothing is registered while it is false, whatever the models declare. |
| `ReadOnly` | `false` | Publish the read tools only, for an agent that may look and not touch. |
| `Prefix` | `""` | Prepended to every tool name, for telling two services apart when a client is connected to both. |

The framework's own `MCP` section still has to be enabled: it owns the endpoint,
the bearer token and the origin allow list. See
[EVO's MCP documentation](https://github.com/getevo/evo/blob/master/docs/mcp.md).

### Example

```yaml
MCP:
  enabled: true
  path: /mcp
  name: restify-demo
  version: 1.0.0
  instructions: Read and edit customers and invoices of the demo shop.
  token: a-secret-token

Restify:
  MCP:
    Enabled: true
    ReadOnly: false
    Prefix: ""
```

With `Prefix: "shop_"` the same models publish `shop_customer_list`,
`shop_invoice_list` and so on. With `ReadOnly: true` only `customer_list`,
`customer_get`, `invoice_list` and `invoice_get` survive.

## The generated tools

Each marked model gets up to five tools, named after its table.

| Tool | Endpoint behind it | Notes |
|---|---|---|
| `<table>_list` | `GET /paginate` | Filtering, ordering, field selection, associations, paging |
| `<table>_get` | `GET /:pk` | One row by primary key |
| `<table>_create` | `PUT /` | One row |
| `<table>_update` | `PATCH /:pk` | Partial: only the fields you send are written |
| `<table>_delete` | `DELETE /:pk` | Soft delete when the model supports it |

A tool exists only when the endpoint behind it exists, so the feature markers
gate both at once: `restify.DisableDelete` removes the route and the tool.
`restify.DisableList` removes both read tools. `Restify.MCP.ReadOnly` removes
the three write tools everywhere.

The tools are annotated the way the protocol expects: the read tools are marked
read-only, `_delete` is marked destructive, and everything except `_create` is
marked idempotent. A client can use those hints to decide what to confirm with
the user.

### Example: three models, nine tools

The example application declares three models
([`example-mcp/models.go`](../example-mcp/models.go)):

```go
type Customer struct {
    // ... columns ...
    restify.API
    restify.MCP            // all five tools
}

type Invoice struct {
    // ... columns ...
    restify.API
    restify.MCP
    restify.DisableDelete  // four tools: no _delete
}

type AuditLog struct {
    // ... columns ...
    restify.API            // REST only, no marker, no tools
}
```

`tools/list` answers with exactly this, in this order:

| Tool | `annotations` |
|---|---|
| `customer_list` | `readOnlyHint`, `idempotentHint` |
| `customer_get` | `readOnlyHint`, `idempotentHint` |
| `customer_create` | none |
| `customer_update` | `idempotentHint` |
| `customer_delete` | `destructiveHint`, `idempotentHint` |
| `invoice_list` | `readOnlyHint`, `idempotentHint` |
| `invoice_get` | `readOnlyHint`, `idempotentHint` |
| `invoice_create` | none |
| `invoice_update` | `idempotentHint` |

`AuditLog` carries no marker, so it has REST routes and no tool. `Invoice`
carries `DisableDelete`, so `invoice_delete` does not exist — calling it is a
protocol level error, not a permission error:

```json
{"jsonrpc":"2.0","id":11,"error":{"code":-32602,"message":"Unknown tool: invoice_delete"}}
```

### Why the marker is opt-in

A REST route costs nothing until someone calls it. A tool definition is
different: every tool is sent to the model at the start of every conversation.
A service with forty models would spend an agent's context budget before it
asked its first question, so a model is only published when it says so.

## Input schemas

The schema a model sees is derived from your model definition, so the constraint
advertised to the agent is the one the handlers enforce:

- names come from the `json` tags
- types come from the columns
- descriptions come from `description` tags, or from `gorm:"comment:..."`
- constraints come from `validation` tags — `required` becomes a required
  property, `in(draft,sent,paid)` becomes an enum, `>=10` becomes a minimum

### Example: a model and the schema it produces

This model:

```go
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
```

produces this `invoice_create` entry in `tools/list`:

```json
{
  "name": "invoice_create",
  "title": "Create Invoice",
  "description": "Insert one row into Invoice (table `invoice`). Send the fields of the new row as the arguments.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "customer_id": { "type": "integer", "description": "Customer this invoice belongs to" },
      "reference":   { "type": "string",  "description": "Human readable invoice reference, for example INV-1001" },
      "total":       { "type": "number",  "description": "Invoice total in euro" },
      "status":      { "type": "string",  "description": "Lifecycle state of the invoice",
                       "enum": ["draft", "sent", "paid"] },
      "issued_at":   { "type": "string",  "description": "When the invoice was issued",
                       "format": "date-time" }
    },
    "required": ["customer_id", "reference", "status"]
  }
}
```

Three things happened there. `id` is gone, because the database assigns it. The
`Customer` relation is gone, because a relation is not a column. And
`validation:"required,in(draft,sent,paid)"` became both a `required` entry and
an `enum` — one declaration, advertised to the model and enforced by the
handler.

`invoice_update` is the same schema with `id` added, marked required, and
nothing else required:

```json
"required": ["id"]
```

That is what "only the fields you send are written" means in schema terms: the
key selects the row, everything else is optional.

`_get` and `_delete` take the key alone:

```json
{
  "type": "object",
  "properties": {
    "id": { "type": "integer", "description": "Primary key of the row to address" }
  },
  "required": ["id"]
}
```

Some things are deliberately left out of the write tools:

- **generated keys**, which the database assigns
- **fields that are not creatable or not updatable**, per the GORM schema
- **managed columns** — created and updated timestamps, and the soft delete
  pair. These are writable over HTTP, where a stray `created_at` in a request
  body is nobody's plan. Handing an agent a `deleted` flag is different: it will
  eventually set it instead of calling the delete tool.

They are still returned in results, and can still be filtered on. The `Customer`
model of the example embeds `types.CreatedAt`, `types.UpdatedAt` and
`types.SoftDelete`, and this is the difference it makes:

```
customer_create / customer_update  →  name, email, country, tags   (+ id on update)
customer_list filters              →  id, name, email, country, tags,
                                      created_at, updated_at, deleted, deleted_at
```

### Filtering

`_list` takes a list of conditions rather than the query string grammar:

```json
{
  "filters": [
    {"field": "status", "op": "in", "value": "sent,paid"},
    {"field": "total", "op": "gt", "value": 100}
  ],
  "order": "total.desc",
  "page": 1,
  "size": 25
}
```

The schema advertises the columns of that model as the only accepted values of
`field`, and these operators as the only accepted values of `op`:

`eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `notin`, `between`, `contains`,
`search`, `isnull`, `notnull`

`in` and `notin` take a comma separated list — or a JSON array, which is
accepted too. `between` takes two comma separated dates. `isnull` and `notnull`
take no value. Page size is between 10 and 100, the same bounds the REST
endpoint applies.

A few more filter examples, all valid against the example's `Invoice`:

```json
{"filters": [{"field": "reference", "op": "contains", "value": "INV-10"}]}
{"filters": [{"field": "status", "op": "in", "value": ["draft", "sent"]}]}
{"filters": [{"field": "issued_at", "op": "between", "value": "2026-01-01,2026-12-31"}]}
{"filters": [{"field": "total", "op": "gte", "value": 100},
             {"field": "total", "op": "lte", "value": 1000}],
 "order": "issued_at.desc", "fields": "id,reference,total"}
{"filters": [{"field": "customer_id", "op": "eq", "value": 1}], "associations": "Customer"}
```

Conditions are combined with AND.

`fields` narrows the columns the database is asked for. The rows come back with
the same shape either way, so a column left out arrives as its zero value —
`"status": ""`, not an absent key.

`associations` loads relations. `1` loads all of them; naming one takes the **Go
field name**, not the JSON name, so the `Customer *Customer` field of `Invoice`
is `Customer` and `customer` is rejected:

```
customer: unsupported relations for schema Invoice
```

That is the REST behaviour, unchanged — the tool passes the value straight
through to the same handler.

## Every tool, end to end

Each block below is a real exchange with the example application. The server was
started with `go run .` in [`example-mcp`](../example-mcp) and seeded with three
customers and four invoices.

### `tools/list`

```shell
curl -s http://127.0.0.1:8080/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### `_list` — a page of rows

```json
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{
  "name":"invoice_list",
  "arguments":{"filters":[{"field":"status","op":"eq","value":"paid"}],
               "order":"total.desc","size":10}}}
```

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{"type": "text", "text": "{\"data\":[ ... ]}"}],
    "structuredContent": {
      "data": [
        {"id": 3, "customer_id": 2, "reference": "INV-1003", "total": 1500,
         "status": "paid", "issued_at": "2026-08-09T06:29:39.57+02:00"},
        {"id": 1, "customer_id": 1, "reference": "INV-1001", "total": 240.5,
         "status": "paid", "issued_at": "2026-08-07T06:29:39.57+02:00"}
      ],
      "total": 2,
      "offset": 0,
      "total_pages": 1,
      "current_page": 1,
      "size": 10,
      "success": true,
      "error": "",
      "type": "",
      "validation_error": null
    }
  }
}
```

`content` carries the same JSON as text, for a client that cannot read
structured results. The envelope is the REST pagination envelope, unchanged, so
an agent can tell a first page from the whole table.

### `_get` — one row

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{
  "name":"invoice_get","arguments":{"id":1}}}
```

```json
"structuredContent": {
  "id": 1, "customer_id": 1, "reference": "INV-1001",
  "total": 240.5, "status": "paid", "issued_at": "2026-08-07T06:29:39.57+02:00"
}
```

### `_create` — insert a row

```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{
  "name":"customer_create",
  "arguments":{"name":"Umbrella SpA","email":"ap@umbrella.example",
               "country":"IT","tags":["vip"]}}}
```

```json
"structuredContent": {
  "id": 4,
  "name": "Umbrella SpA",
  "email": "ap@umbrella.example",
  "country": "IT",
  "tags": ["vip"],
  "created_at": "2026-08-10T06:30:42.95+02:00",
  "updated_at": "2026-08-10T06:30:42.95+02:00",
  "deleted": false,
  "deleted_at": null
}
```

The row comes back as it was written, with the generated key and the managed
columns the agent was not allowed to send.

### `_update` — write some fields

```json
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{
  "name":"invoice_update","arguments":{"id":4,"status":"sent"}}}
```

```json
"structuredContent": {
  "id": 4, "customer_id": 3, "reference": "INV-1004",
  "total": 42, "status": "sent", "issued_at": "2026-08-10T06:29:39.57+02:00"
}
```

Only `status` was sent, so only `status` was written. Everything else is the row
as it stood.

### `_delete` — remove a row

```json
{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{
  "name":"customer_delete","arguments":{"id":4}}}
```

```json
"structuredContent": {"deleted": true, "id": "4"}
```

`Customer` embeds `types.SoftDelete`, so the row is flagged rather than removed
and disappears from `customer_list` on the next call.

## Errors

A failure comes back as a tool execution error rather than a result: the JSON-RPC
call still succeeds, and `isError` tells the client that the text is a problem
the model should read and act on.

**A value the model made up**

```json
{"name":"invoice_create","arguments":{"customer_id":1,"reference":"INV-2001","status":"archived"}}
```

```json
"result": {
  "content": [{"type":"text","text":"validation failed: status value must be one of: draft,sent,paid"}],
  "isError": true
}
```

**A column that does not exist** — the message lists the ones that do, so the
next attempt usually lands:

```json
"text": "unknown column \"colour\", available columns are id, customer_id, reference, total, status, issued_at"
```

**A missing primary key** — checked before the handler runs, because the
fallback would otherwise address whatever row holds the zero key:

```json
"text": "missing primary key \"id\""
```

**A row that is not there**

```json
"text": "object does not exist"
```

**A tool that does not exist** — this one is a protocol error, not a result,
because the client asked for something the server never published:

```json
{"jsonrpc":"2.0","id":11,"error":{"code":-32602,"message":"Unknown tool: invoice_delete"}}
```

## Permissions and hooks

A tool call is not a shortcut past your rules. It builds the same
`restify.Context` a route does and calls the same handler, so:

- `RestPermission` is consulted, with the same permission names
- the global permission handler is consulted
- `OnBeforeCreate`, `OnAfterGet` and every other hook run
- `SetCondition` and `Override` apply
- validation runs, and a failure comes back as an error the model can correct

The context carries the HTTP request of the MCP call, so a permission handler
that reads a header or an authenticated user works unchanged:

```go
func (invoice *Invoice) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    user, err := GetUser(context.Request) // the MCP request
    if err != nil {
        return false
    }
    if permissions.Has("DELETE") && !user.IsAdmin {
        return false
    }
    context.SetCondition("company_id", "=", user.CompanyID)
    return true
}
```

A hook behaves the same way, and cannot tell whether it was reached over HTTP or
over MCP:

```go
func (invoice *Invoice) OnBeforeCreate(context *restify.Context) error {
    if invoice.Status == "" {
        invoice.Status = "draft"
    }
    invoice.IssuedAt = time.Now()
    return nil
}
```

> **`tools/list` is not filtered by permission.** Generated tools are registered
> once at startup, so every published model appears in the list for every
> caller. Access control happens on the call: a denied `RestPermission` comes
> back as an error, not as a missing tool. If you need a model hidden from a
> caller rather than merely refused, do not put the `restify.MCP` marker on it —
> or run that agent against a server with `Restify.MCP.ReadOnly` on.

## Connecting a client

The endpoint is streamable HTTP, POST only. For an MCP client that speaks HTTP,
point it at the path from `MCP.path` and pass the bearer token if one is set —
here is a Claude Code `.mcp.json`:

```json
{
  "mcpServers": {
    "shop": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer a-secret-token" }
    }
  }
}
```

The `MCP.instructions` key of `config.yml` is handed to the model as guidance
about what the server is for, and is worth writing:

```yaml
MCP:
  instructions: Read and edit customers and invoices of the demo shop.
```

Transport details, protocol revisions and the authentication layers are the
framework's, and are documented in
[EVO's MCP documentation](https://github.com/getevo/evo/blob/master/docs/mcp.md#authentication-and-authorization).

## Results

A read or write tool answers with the row or rows, as structured content plus a
JSON mirror. `_list` answers with the pagination envelope — `data`, `total`,
`total_pages`, `current_page` — so an agent can tell a first page from the whole
table. `_delete` answers with `{"deleted": true}` and the key it deleted.

## How it works

Every handler used to read its parameters straight off the HTTP request. They
now read them through the context, which falls back to a
[`restify.Input`](../input.go) when the call did not arrive over HTTP. An MCP
tool fills that in from the call arguments and invokes the endpoint handler
directly, so there is one implementation of listing, filtering, validating and
saving — not two that can drift apart.

The REST API is served the whole time, from the same handlers. These two return
the same rows:

```shell
curl -sg 'http://127.0.0.1:8080/admin/rest/invoice/paginate?status[eq]=paid&order=total.desc'
```

```json
{"name":"invoice_list","arguments":{"filters":[{"field":"status","op":"eq","value":"paid"}],
                                    "order":"total.desc"}}
```

## See also

- [EVO MCP server](https://github.com/getevo/evo/blob/master/docs/mcp.md) — the endpoint, transport and authentication
- [Endpoints](./endpoints.md) — the REST surface the tools delegate to
- [Permissions](./permissions.md) — how access control is decided
- [Customization](./customization.md#features) — the feature markers
- [`example-mcp`](../example-mcp) — the runnable application every example here came from
