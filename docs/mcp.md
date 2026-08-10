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

For example, this field

```go
Status string `gorm:"column:status;size:16" validation:"required,in(draft,sent,paid)" description:"Lifecycle state" json:"status"`
```

is advertised as

```json
{
  "status": {
    "type": "string",
    "description": "Lifecycle state",
    "enum": ["draft", "sent", "paid"]
  }
}
```

and `status` is listed as required by `_create`.

Some things are deliberately left out of the write tools:

- **generated keys**, which the database assigns
- **fields that are not creatable or not updatable**, per the GORM schema
- **managed columns** — created and updated timestamps, and the soft delete
  pair. These are writable over HTTP, where a stray `created_at` in a request
  body is nobody's plan. Handing an agent a `deleted` flag is different: it will
  eventually set it instead of calling the delete tool.

They are still returned in results, and can still be filtered on.

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

Because the framework filters `tools/list` by permission as well, a caller never
sees a tool it cannot call.

## Results

A read or write tool answers with the row or rows, as structured content plus a
JSON mirror. `_list` answers with the pagination envelope — `data`, `total`,
`total_pages`, `current_page` — so an agent can tell a first page from the whole
table. `_delete` answers with the key it deleted.

A failure comes back as a tool execution error rather than a result, carrying the
message and any validation errors:

```
validation failed: status value must be one of: draft,sent,paid
```

## How it works

Every handler used to read its parameters straight off the HTTP request. They
now read them through the context, which falls back to a
[`restify.Input`](../input.go) when the call did not arrive over HTTP. An MCP
tool fills that in from the call arguments and invokes the endpoint handler
directly, so there is one implementation of listing, filtering, validating and
saving — not two that can drift apart.

## See also

- [EVO MCP server](https://github.com/getevo/evo/blob/master/docs/mcp.md) — the endpoint, transport and authentication
- [Endpoints](./endpoints.md) — the REST surface the tools delegate to
- [Permissions](./permissions.md) — how access control is decided
- [Customization](./customization.md#features) — the feature markers
