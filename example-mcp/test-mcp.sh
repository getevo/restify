#!/usr/bin/env bash
# Drives every generated tool against a running example server and prints the
# result of each call. Start the app first, then run: bash test-mcp.sh
set -u

URL=${URL:-http://127.0.0.1:8080/mcp}
ID=0

call() {
	local label="$1" name="$2" args="$3"
	ID=$((ID + 1))
	echo "--- $label"
	curl -s "$URL" -H 'Content-Type: application/json' \
		-d "{\"jsonrpc\":\"2.0\",\"id\":$ID,\"method\":\"tools/call\",\"params\":{\"name\":\"$name\",\"arguments\":$args}}" |
		python -c "
import json,sys
d = json.load(sys.stdin)
r = d.get('result', d)
if r.get('isError'):
    print('  ERROR:', r['content'][0]['text'])
else:
    c = r.get('structuredContent', r.get('content'))
    print(' ', json.dumps(c)[:400])
"
}

call "list paid invoices"        invoice_list   '{"filters":[{"field":"status","op":"eq","value":"paid"}]}'
call "list invoices over 100"    invoice_list   '{"filters":[{"field":"total","op":"gt","value":100}],"order":"total.desc","fields":"id,reference,total"}'
call "list with relation"        invoice_list   '{"filters":[{"field":"reference","op":"contains","value":"INV-100"}],"associations":"Customer","size":2}'
call "get invoice 1"             invoice_get    '{"id":1}'
call "get missing invoice"       invoice_get    '{"id":999}'
call "get without key"           invoice_get    '{}'
call "unknown column"            invoice_list   '{"filters":[{"field":"secret","op":"eq","value":"x"}]}'
call "create customer"           customer_create '{"name":"Wayne Enterprises","email":"ap@wayne.example","country":"US"}'
call "create invalid customer"   customer_create '{"name":"No Email"}'
call "create invoice"            invoice_create '{"customer_id":1,"reference":"INV-2001","total":10.5,"status":"draft"}'
call "create bad status"         invoice_create '{"customer_id":1,"reference":"INV-2002","total":1,"status":"archived"}'
call "update invoice 1"          invoice_update '{"id":1,"status":"sent","total":260}'
call "delete customer 3"         customer_delete '{"id":3}'
call "delete without key"        customer_delete '{}'
call "list customers"            customer_list  '{"order":"id.asc"}'
