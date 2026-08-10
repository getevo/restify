package restify

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"gorm.io/gorm/schema"
)

// MCP is a feature marker. Embed it in a model to expose that model to AI
// agents as MCP tools:
//
//	type User struct {
//	    restify.API
//	    restify.MCP
//	    ID   uint   `json:"id"`
//	    Name string `json:"name"`
//	}
//
// The marker is opt-in rather than opt-out on purpose. An HTTP route costs
// nothing until it is called, but every tool definition is sent to the model on
// every conversation, so a server that exposes forty models would spend the
// context budget of the agent before it asks its first question.
type MCP struct{}

// MCPSettings is the Restify.MCP section of the configuration file.
type MCPSettings struct {
	// Enabled is the master switch. Nothing is registered while it is false,
	// whatever the models declare.
	Enabled bool `json:"enabled" yaml:"enabled" description:"Expose models marked with restify.MCP as MCP tools" default:"false"`

	// ReadOnly drops the create, update and delete tools, leaving an agent
	// that can read the database and change nothing.
	ReadOnly bool `json:"read_only" yaml:"read_only" description:"Expose the read tools only" default:"false"`

	// Prefix is prepended to every generated tool name, for telling two
	// services apart when a client is connected to both.
	Prefix string `json:"prefix" yaml:"prefix" description:"Prefix prepended to every generated tool name" default:""`
}

var mcpSettings MCPSettings

// loadMCPSettings reads the Restify.MCP section. Keys are read one at a time
// rather than cast as a section, so a configuration file with no Restify
// section at all is not an error.
func loadMCPSettings() {
	_ = settings.Register(
		settings.SettingDomain{
			Title:       "Restify",
			Domain:      "RESTIFY",
			Description: "Restify configurations",
			Visible:     true,
		},
		settings.Setting{
			Domain:      "RESTIFY",
			Name:        "MCP.ENABLED",
			Title:       "Expose MCP tools",
			Description: "Expose models marked with restify.MCP as MCP tools",
			Type:        "bool",
			Value:       "false",
			Visible:     true,
		},
		settings.Setting{
			Domain:      "RESTIFY",
			Name:        "MCP.READ_ONLY",
			Title:       "MCP read only",
			Description: "Expose the read tools only",
			Type:        "bool",
			Value:       "false",
			Visible:     true,
		},
		settings.Setting{
			Domain:      "RESTIFY",
			Name:        "MCP.PREFIX",
			Title:       "MCP tool prefix",
			Description: "Prefix prepended to every generated tool name",
			Type:        "text",
			Value:       "",
			Visible:     true,
		},
	)

	mcpSettings.Enabled = settings.Get("Restify.MCP.Enabled", false).Bool()
	mcpSettings.ReadOnly = settings.Get("Restify.MCP.ReadOnly", false).Bool()
	mcpSettings.Prefix = settings.Get("Restify.MCP.Prefix", "").String()
}

// mcpFilterOperators lists the filter conditions a tool call may use, in the
// order they are advertised to the model. It is a subset of what the query
// grammar accepts: every operator here is one filterMapper implements.
var mcpFilterOperators = []string{
	"eq", "neq", "gt", "gte", "lt", "lte",
	"in", "notin", "between", "contains", "search", "isnull", "notnull",
}

// mcpResource is everything the tool handlers of one resource need, resolved
// once at registration time so that no reflection happens per call.
type mcpResource struct {
	resource *Resource

	// keys maps the argument name of each primary key onto its column name.
	keys []mcpKey

	// columns lists the column names a filter may name.
	columns []string
}

type mcpKey struct {
	Arg    string
	Column string
}

// registerMCPTools exposes every model that carries both the API and the MCP
// markers. It runs from WhenReady, which the framework calls before it mounts
// the MCP endpoint, so the tools are in place by the time a client can list
// them.
func registerMCPTools() {
	var tables []string
	for table := range Resources {
		tables = append(tables, table)
	}
	// Registration order decides the order of tools/list, so keep it stable
	// rather than at the mercy of map iteration.
	sort.Strings(tables)

	var tools []evo.MCPTool
	for _, table := range tables {
		var resource = Resources[table]
		if !resource.Feature.API || !resource.Feature.MCP {
			continue
		}
		tools = append(tools, mcpResourceTools(resource)...)
	}

	if len(tools) == 0 {
		log.Warning("restify: mcp is enabled but no model carries the restify.MCP marker")
		return
	}

	evo.RegisterMCPTool(tools...)
	log.Infof("restify: exposed %d mcp tools", len(tools))
}

// mcpResourceTools builds the tools of a single resource. A tool is only built
// when the endpoint it delegates to exists, so the DisableList, DisableCreate,
// DisableUpdate and DisableDelete markers gate the tools exactly as they gate
// the routes.
func mcpResourceTools(resource *Resource) []evo.MCPTool {
	var m = &mcpResource{resource: resource}

	for _, field := range resource.Schema.PrimaryFields {
		m.keys = append(m.keys, mcpKey{Arg: mcpArgName(field), Column: field.DBName})
	}
	for _, field := range resource.Schema.Fields {
		if mcpIsRelation(field) {
			continue
		}
		m.columns = append(m.columns, field.DBName)
	}

	var name = mcpSettings.Prefix + resource.Table
	var subject = fmt.Sprintf("%s (table `%s`)", resource.Schema.Name, resource.Table)
	var tools []evo.MCPTool

	if action := m.action(MethodGET, "paginate"); action != nil {
		tools = append(tools, evo.MCPTool{
			Name:        name + "_list",
			Title:       "List " + resource.Schema.Name,
			Description: m.listDescription(subject),
			Input:       mcpListInputSample(m),
			ReadOnly:    true,
			Idempotent:  true,
			Handler:     m.listHandler(action),
		})
	}

	if action := m.action(MethodGET, m.keyURL()); action != nil && len(m.keys) > 0 {
		tools = append(tools, evo.MCPTool{
			Name:        name + "_get",
			Title:       "Get " + resource.Schema.Name,
			Description: fmt.Sprintf("Fetch a single row of %s by its primary key.", subject),
			Input:       mcpSample(m.resource, mcpKeyFields),
			ReadOnly:    true,
			Idempotent:  true,
			Handler:     m.keyHandler(action, "GET", false),
		})
	}

	if mcpSettings.ReadOnly {
		return tools
	}

	if action := m.action(MethodPUT, ""); action != nil {
		tools = append(tools, evo.MCPTool{
			Name:        name + "_create",
			Title:       "Create " + resource.Schema.Name,
			Description: fmt.Sprintf("Insert one row into %s. Send the fields of the new row as the arguments.", subject),
			Input:       mcpSample(m.resource, mcpCreateFields),
			Handler:     m.writeHandler(action, "PUT"),
		})
	}

	if action := m.action(MethodPatch, m.keyURL()); action != nil && len(m.keys) > 0 {
		tools = append(tools, evo.MCPTool{
			Name:  name + "_update",
			Title: "Update " + resource.Schema.Name,
			Description: fmt.Sprintf("Update one row of %s. The primary key selects the row; "+
				"every other argument is optional and only the fields you send are written.", subject),
			Input:      mcpSample(m.resource, mcpUpdateFields),
			Idempotent: true,
			Handler:    m.writeHandler(action, "PATCH"),
		})
	}

	if action := m.action(MethodDELETE, m.keyURL()); action != nil && len(m.keys) > 0 {
		tools = append(tools, evo.MCPTool{
			Name:        name + "_delete",
			Title:       "Delete " + resource.Schema.Name,
			Description: fmt.Sprintf("Delete one row of %s, selected by its primary key.", subject),
			Input:       mcpSample(m.resource, mcpKeyFields),
			Destructive: true,
			Idempotent:  true,
			Handler:     m.keyHandler(action, "DELETE", true),
		})
	}

	return tools
}

// listDescription spells out the filter grammar and the columns it may name,
// because that is the only place the model can learn them.
func (m *mcpResource) listDescription(subject string) string {
	return fmt.Sprintf("List rows of %s, one page at a time.\n\n"+
		"Filter with `filters`: each entry names a column, an operator and a value. "+
		"`in` and `notin` take a comma separated list, `between` takes two comma separated dates, "+
		"`isnull` and `notnull` take no value. Order with `order`, for example `created_at.desc`.\n\n"+
		"Columns: %s.", subject, strings.Join(m.columns, ", "))
}

// action finds the endpoint of this resource with the given method and URL. The
// URL is what SetAction stored, so a primary key endpoint reads ":id".
func (m *mcpResource) action(method Method, url string) *Endpoint {
	var want = strings.Trim(url, "/")
	for _, action := range m.resource.Actions {
		if action.Method == method && strings.Trim(action.URL, "/") == want {
			return action
		}
	}
	return nil
}

// keyURL rebuilds the URL SetAction generates for a primary key endpoint.
func (m *mcpResource) keyURL() string {
	var url string
	for _, field := range m.resource.Schema.PrimaryFields {
		url += "/:" + field.DBName
	}
	return url
}

// --- tool handlers ---------------------------------------------------------

// mcpListArgs decodes the arguments of a list call. It mirrors the schema
// advertised by mcpListInputSample: the schema is built by reflection so it can
// name this resource's columns, but decoding is done against this static type.
type mcpListArgs struct {
	Filters []struct {
		Field string `json:"field"`
		Op    string `json:"op"`
		Value any    `json:"value"`
	} `json:"filters"`
	Order        string `json:"order"`
	Fields       string `json:"fields"`
	Associations string `json:"associations"`
	Page         int    `json:"page"`
	Size         int    `json:"size"`
}

func (m *mcpResource) listHandler(action *Endpoint) evo.MCPToolHandler {
	return func(c *evo.MCPContext) any {
		var args mcpListArgs
		if len(c.Arguments) > 0 {
			if err := json.Unmarshal(c.Arguments, &args); err != nil {
				return fmt.Errorf("invalid arguments: %w", err)
			}
		}

		var input = &Input{Verb: "GET"}
		if args.Page > 0 {
			input.Set("page", strconv.Itoa(args.Page))
		}
		if args.Size > 0 {
			input.Set("size", strconv.Itoa(args.Size))
		}
		input.Set("order", args.Order)
		input.Set("fields", args.Fields)
		input.Set("associations", args.Associations)

		for _, filter := range args.Filters {
			if !mcpContains(m.columns, filter.Field) {
				return fmt.Errorf("unknown column %q, available columns are %s",
					filter.Field, strings.Join(m.columns, ", "))
			}
			var op = strings.ToLower(strings.TrimSpace(filter.Op))
			if op == "" {
				op = "eq"
			}
			if !mcpContains(mcpFilterOperators, op) {
				return fmt.Errorf("unknown operator %q, available operators are %s",
					filter.Op, strings.Join(mcpFilterOperators, ", "))
			}
			input.AddFilter(filter.Field, op, mcpFilterValue(filter.Value))
		}

		response, err := m.run(action, c.Request, input)
		if err != nil {
			return err
		}
		return response
	}
}

// keyHandler serves the tools that address one row by primary key and send no
// body: get and delete.
func (m *mcpResource) keyHandler(action *Endpoint, verb string, confirm bool) evo.MCPToolHandler {
	return func(c *evo.MCPContext) any {
		input, err := m.keyInput(c, verb)
		if err != nil {
			return err
		}

		response, err := m.run(action, c.Request, input)
		if err != nil {
			return err
		}
		if confirm {
			// A delete has nothing to return but the fact that it happened.
			var result = map[string]any{"deleted": true}
			for _, key := range m.keys {
				result[key.Arg] = input.Params[key.Column]
			}
			return result
		}
		return response.Data
	}
}

// writeHandler serves the tools that carry a body: create and update.
func (m *mcpResource) writeHandler(action *Endpoint, verb string) evo.MCPToolHandler {
	return func(c *evo.MCPContext) any {
		var input = &Input{Verb: verb}
		if action.PKUrl {
			keyed, err := m.keyInput(c, verb)
			if err != nil {
				return err
			}
			input = keyed
		}

		if len(c.Arguments) == 0 {
			return fmt.Errorf("no fields given")
		}
		// The arguments are the row: they are named after the model's json
		// tags, which is exactly what the handlers decode.
		input.Body = c.Arguments

		response, err := m.run(action, c.Request, input)
		if err != nil {
			return err
		}
		return response.Data
	}
}

// keyInput reads the primary key out of the call arguments. The check is
// explicit rather than left to FindByPrimaryKey, which falls back to the zero
// value of the field when a key is missing — for a delete that would mean
// addressing whatever row holds the zero key.
func (m *mcpResource) keyInput(c *evo.MCPContext, verb string) (*Input, error) {
	var input = &Input{Verb: verb, Params: map[string]string{}}
	for _, key := range m.keys {
		var value = c.Arg(key.Arg).String()
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("missing primary key %q", key.Arg)
		}
		input.Params[key.Column] = value
	}
	return input, nil
}

// run executes an endpoint handler outside HTTP and reports the outcome the way
// a tool result expects: a value on success, an error the model can act on
// otherwise.
func (m *mcpResource) run(action *Endpoint, request *evo.Request, input *Input) (*Pagination, error) {
	var context = &Context{
		Request: request,
		Input:   input,
		Action:  action,
		Object:  m.resource.Ref,
		Schema:  m.resource.Schema,
		Response: &Pagination{
			TotalPages: 1,
			Total:      1,
			Page:       1,
			Size:       1,
			Success:    true,
		},
	}

	context.HandleError(action.Handler(context))

	if !context.Response.Success {
		return nil, mcpError(context.Response)
	}
	return context.Response, nil
}

// mcpError turns a failed response into one error the model can read, folding
// the field level validation errors into the message.
func mcpError(response *Pagination) error {
	var message = response.Error
	if message == "" {
		message = "the call failed"
	}
	if len(response.ValidationError) > 0 {
		var parts []string
		for _, item := range response.ValidationError {
			parts = append(parts, item.Field+" "+item.Error)
		}
		message += ": " + strings.Join(parts, "; ")
	}
	return fmt.Errorf("%s", message)
}

// --- input schemas ---------------------------------------------------------

// mcpFieldMode selects which fields of a model an input schema describes.
type mcpFieldMode int

const (
	// mcpKeyFields describes the primary key alone.
	mcpKeyFields mcpFieldMode = iota
	// mcpCreateFields describes the fields a new row may set.
	mcpCreateFields
	// mcpUpdateFields describes the primary key plus every writable field,
	// all of them optional.
	mcpUpdateFields
)

// mcpSample returns a zero value whose type describes the arguments of a tool.
// The framework derives the JSON Schema from it, so the schema the model reads
// is generated from the model definition itself.
//
// The type is built rather than reused because a model is not a good argument
// list: it carries relations, embedded structs and column types such as a
// decimal or a JSON document that do not survive being described field by
// field. This flattens it to the columns that can actually be sent.
func mcpSample(resource *Resource, mode mcpFieldMode) any {
	var fields []reflect.StructField
	var seen = map[string]bool{}
	var softDelete = mcpHasSoftDelete(resource.Schema)

	for _, field := range resource.Schema.Fields {
		if mcpIsRelation(field) {
			continue
		}
		switch mode {
		case mcpKeyFields:
			if !field.PrimaryKey {
				continue
			}
		case mcpCreateFields:
			if !field.Creatable || mcpIsManaged(field, softDelete) {
				continue
			}
			// A generated key is the database's to assign.
			if field.PrimaryKey && field.AutoIncrement {
				continue
			}
		case mcpUpdateFields:
			if !field.PrimaryKey && (!field.Updatable || mcpIsManaged(field, softDelete)) {
				continue
			}
		}

		var name = mcpArgName(field)
		if name == "" || seen[field.Name] {
			continue
		}
		seen[field.Name] = true

		fields = append(fields, reflect.StructField{
			Name: field.Name,
			Type: mcpArgType(field),
			Tag:  reflect.StructTag(mcpArgTag(field, name, mode)),
		})
	}

	if len(fields) == 0 {
		return struct{}{}
	}
	return reflect.New(reflect.StructOf(fields)).Elem().Interface()
}

// mcpListInputSample describes the arguments of a list call. The filter entry
// is built for this resource so that the column names are advertised as the
// only accepted values rather than described in prose.
func mcpListInputSample(m *mcpResource) any {
	var filter = reflect.StructOf([]reflect.StructField{
		{
			Name: "Field",
			Type: reflect.TypeOf(""),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"field" validation:"required,in(%s)" description:"Column to filter on"`,
				strings.Join(m.columns, ","))),
		},
		{
			Name: "Op",
			Type: reflect.TypeOf(""),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"op" default:"eq" validation:"in(%s)" description:"Comparison to apply"`,
				strings.Join(mcpFilterOperators, ","))),
		},
		{
			Name: "Value",
			Type: reflect.TypeOf(""),
			Tag:  `json:"value" description:"Value to compare against. Comma separated for in, notin and between; omitted for isnull and notnull"`,
		},
	})

	return reflect.New(reflect.StructOf([]reflect.StructField{
		{
			Name: "Filters",
			Type: reflect.SliceOf(filter),
			Tag:  `json:"filters" description:"Conditions every returned row must satisfy"`,
		},
		{
			Name: "Order",
			Type: reflect.TypeOf(""),
			Tag:  `json:"order" description:"Sort order, as column.asc or column.desc, comma separated for more than one"`,
		},
		{
			Name: "Fields",
			Type: reflect.TypeOf(""),
			Tag:  `json:"fields" description:"Comma separated columns to return, all of them when omitted"`,
		},
		{
			Name: "Associations",
			Type: reflect.TypeOf(""),
			Tag:  `json:"associations" description:"Comma separated relations to load, or 1 for all of them"`,
		},
		{
			Name: "Page",
			Type: reflect.TypeOf(0),
			Tag:  `json:"page" default:"1" validation:">=1" description:"Page to return, counted from 1"`,
		},
		{
			Name: "Size",
			Type: reflect.TypeOf(0),
			Tag:  `json:"size" default:"10" validation:">=10,<=100" description:"Rows per page, from 10 to 100"`,
		},
	})).Elem().Interface()
}

// mcpArgName is the name a field is addressed by in a tool call: its json name,
// falling back to the column name when the field has no usable json tag.
func mcpArgName(field *schema.Field) string {
	var name, _, _ = strings.Cut(field.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return field.DBName
	}
	return name
}

// mcpArgTag builds the struct tag of one argument. The validation rules are
// copied from the model so that the constraint advertised to the model is the
// one the handlers enforce, with one change: an update writes only the fields
// it is given, so nothing but the primary key is required.
func mcpArgTag(field *schema.Field, name string, mode mcpFieldMode) string {
	var parts = []string{fmt.Sprintf(`json:"%s"`, name)}

	var description = field.Comment
	if description == "" {
		description = field.Tag.Get("description")
	}
	if field.PrimaryKey && mode != mcpCreateFields {
		if description == "" {
			description = "Primary key of the row to address"
		} else {
			description += " (primary key of the row to address)"
		}
	}
	if description != "" {
		parts = append(parts, fmt.Sprintf(`description:%q`, description))
	}

	var rules []string
	switch mode {
	case mcpKeyFields:
		rules = []string{"required"}
	case mcpCreateFields:
		rules = mcpSplitRules(field.Tag.Get("validation"))
	case mcpUpdateFields:
		rules = mcpDropRequired(mcpSplitRules(field.Tag.Get("validation")))
		if field.PrimaryKey {
			rules = append([]string{"required"}, rules...)
		}
	}
	if len(rules) > 0 {
		parts = append(parts, fmt.Sprintf(`validation:%q`, strings.Join(rules, ",")))
	}

	return strings.Join(parts, " ")
}

// mcpArgType is the type an argument is described as. A column whose Go type is
// a struct — a decimal, a JSON document, a null wrapper — cannot be described
// by walking its fields, so it falls back to the type the column stores.
func mcpArgType(field *schema.Field) reflect.Type {
	var t = field.FieldType
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return t
	}

	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return t
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Struct && t.Elem() != reflect.TypeOf(time.Time{}) {
			return reflect.TypeOf("")
		}
		return t
	}

	switch field.DataType {
	case schema.Bool:
		return reflect.TypeOf(false)
	case schema.Int:
		return reflect.TypeOf(int64(0))
	case schema.Uint:
		return reflect.TypeOf(uint64(0))
	case schema.Float:
		return reflect.TypeOf(float64(0))
	case schema.Time:
		return reflect.TypeOf(time.Time{})
	}
	return reflect.TypeOf("")
}

// mcpIsRelation reports whether a schema field is a relation rather than a
// column. A relation carries no GORM data type, the same test the Postman
// documentation uses.
func mcpIsRelation(field *schema.Field) bool {
	return strings.TrimSpace(string(field.GORMDataType)) == ""
}

// mcpIsManaged reports whether a column belongs to the framework rather than to
// the caller: a created or updated timestamp, or the pair of columns that
// implement soft delete.
//
// These are writable over HTTP, where a stray created_at in a request body is
// nobody's plan. A tool argument is different: it is a suggestion to a model,
// and a model handed a `deleted` flag will eventually set it instead of calling
// the delete tool.
func mcpIsManaged(field *schema.Field, softDelete bool) bool {
	if field.AutoCreateTime != 0 || field.AutoUpdateTime != 0 {
		return true
	}
	if mcpIsSoftDeleteField(field) {
		return true
	}
	// The flag column that sits next to a soft delete timestamp.
	if softDelete && field.DBName == "deleted" && field.FieldType.Kind() == reflect.Bool {
		return true
	}
	return false
}

// mcpHasSoftDelete reports whether the model soft deletes.
func mcpHasSoftDelete(s *schema.Schema) bool {
	for _, field := range s.Fields {
		if mcpIsSoftDeleteField(field) {
			return true
		}
	}
	return false
}

// mcpIsSoftDeleteField reports whether a column drives soft delete, which GORM
// expresses as a field type that contributes its own delete clauses.
func mcpIsSoftDeleteField(field *schema.Field) bool {
	var t = field.FieldType
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var iface = reflect.TypeOf((*schema.DeleteClausesInterface)(nil)).Elem()
	return t.Implements(iface) || reflect.PointerTo(t).Implements(iface)
}

// mcpSplitRules splits a validation tag into rules, leaving the commas inside
// parentheses alone so that a rule such as in(a,b,c) survives.
func mcpSplitRules(tag string) []string {
	if strings.TrimSpace(tag) == "" {
		return nil
	}

	var rules []string
	var depth int
	var current strings.Builder
	for _, r := range tag {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				if rule := strings.TrimSpace(current.String()); rule != "" {
					rules = append(rules, rule)
				}
				current.Reset()
				continue
			}
		}
		current.WriteRune(r)
	}
	if rule := strings.TrimSpace(current.String()); rule != "" {
		rules = append(rules, rule)
	}
	return rules
}

// mcpDropRequired removes the required rule, which an update must not advertise
// for a field the caller is free to leave alone.
func mcpDropRequired(rules []string) []string {
	var kept []string
	for _, rule := range rules {
		if strings.EqualFold(strings.TrimSpace(rule), "required") {
			continue
		}
		kept = append(kept, rule)
	}
	return kept
}

// mcpFilterValue renders a filter value as the query grammar carries it. A
// model that answers with a number, a boolean or a list is understood rather
// than rejected.
func mcpFilterValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		var parts []string
		for _, item := range v {
			parts = append(parts, mcpFilterValue(item))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(v)
	}
}

func mcpContains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
