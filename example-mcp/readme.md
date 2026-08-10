# MCP example

A SQLite backed application that publishes its models to AI agents as MCP tools,
without a single tool written by hand. It is the smallest thing that exercises
the whole path: [`models.go`](models.go) declares three models,
[`config.yml`](config.yml) turns the feature on, and that is the entire setup.

- `Customer` — carries `restify.API` and `restify.MCP`, so it gets all five tools.
  It also carries a string array, the managed timestamps and soft delete, which
  are the column types that need translating before an agent can be told about
  them.
- `Invoice` — marked too, but with `restify.DisableDelete`, so it gets four tools.
- `AuditLog` — has REST endpoints and no `restify.MCP`, so no agent ever sees it.

## Run it

```shell
go run .
```

The database is created and seeded on first run.

## List the tools

```shell
curl -s http://127.0.0.1:8080/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

```
customer_list  customer_get  customer_create  customer_update  customer_delete
invoice_list   invoice_get   invoice_create   invoice_update
```

## Call one

```shell
curl -s http://127.0.0.1:8080/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{
        "name":"invoice_list",
        "arguments":{"filters":[{"field":"status","op":"eq","value":"paid"}],
                     "order":"total.desc"}}}'
```

## Exercise everything

[`test-mcp.sh`](test-mcp.sh) walks every tool and every failure mode — a missing
primary key, an unknown column, a validation error — and prints what came back:

```shell
bash test-mcp.sh
```

## Try the switches

In `config.yml`:

- `Restify.MCP.Enabled: false` — no tools at all
- `Restify.MCP.ReadOnly: true` — the four read tools only
- `Restify.MCP.Prefix: "shop_"` — `shop_invoice_list`, and so on

The REST API is served the whole time at `/admin/rest`, from the same handlers:

```shell
curl -sg 'http://127.0.0.1:8080/admin/rest/invoice/paginate?status[eq]=paid'
```

## Notes

- The SQLite driver in [`sqlite.go`](sqlite.go) is a local one: it is the
  smallest `db.Driver` that satisfies `evo.Setup`, using the pure Go SQLite
  build so no C toolchain is needed.
- No bearer token is configured here, which is fine on a loopback address. Set
  `MCP.token` in `config.yml` for anything else.
