package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/text"
	"github.com/getevo/postman"
	"github.com/gofiber/fiber/v2/log"
	"github.com/iancoleman/strcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"reflect"
	"strconv"
	"strings"
)

// Resources is a map that holds a collection of *Resource objects.
var Resources = map[string]*Resource{}

type Permission string
type Permissions []string

type Method string

const (
	MethodGET    Method = "GET"
	MethodPOST   Method = "POST"
	MethodPatch  Method = "PATCH"
	MethodPUT    Method = "PUT"
	MethodDELETE Method = "DELETE"

	PermissionsModelInfo     Permission = "VIEW+MODEL_INFO"
	PermissionCreate         Permission = "CREATE"
	PermissionUpdate         Permission = "UPDATE"
	PermissionBatchCreate    Permission = "BATCH+CREATE"
	PermissionBatchUpdate    Permission = "BATCH+UPDATE"
	PermissionDelete         Permission = "DELETE"
	PermissionBatchDelete    Permission = "BATCH+DELETE"
	PermissionViewGet        Permission = "VIEW+GET"
	PermissionViewAll        Permission = "VIEW+ALL"
	PermissionAggregate      Permission = "VIEW+AGGREGATE"
	PermissionViewPagination Permission = "VIEW+PAGINATION"
	PermissionSet            Permission = "SET"
)

// Resource represents a resource in an API.
// It holds information about the object, actions, path, schema, table, name, model, JavaScript model,
// and parameters of the resource.
type Resource struct {
	Instance            any            `json:"-"`
	PrimaryFieldDBNames []string       `json:"primary_key"`
	Actions             []*Endpoint    `json:"actions"`
	Schema              *schema.Schema `json:"-"`
	Type                reflect.Type   `json:"-"`
	Ref                 reflect.Value  `json:"-"`
	Table               string         `json:"table"`
	Path                string         `json:"path"`
	Name                string         `json:"model"`
	Feature             Feature        `json:"feature"`
	PostmanGroup        *postman.Item  `json:"-"`
}

func (res *Resource) SetAction(action *Endpoint) {
	action.Name = strcase.ToCamel(action.Name)
	action.Resource = res
	if action.Method == "" {
		action.Method = MethodPOST
	}
	if action.URL == "" {
		action.URL = strcase.ToSnake(action.Name)
	}
	action.URL = strings.Trim(action.URL, "/")
	if action.PKUrl {
		for _, item := range res.Schema.PrimaryFields {
			action.URL += "/:" + item.DBName
		}
	}

	for _, item := range action.URLParams {
		action.URL += "/:" + item.Name
	}

	res.Path = res.Table
	action.AbsoluteURI = "/" + strings.Trim(Prefix+"/"+res.Path+"/"+strings.Trim(action.URL, "/"), "/")
	action.Resource = res

	res.Actions = append(res.Actions, action)

	req := postman.Request{
		Url: &postman.Url{
			Raw: "{{ base_url }}" + action.AbsoluteURI,
			Host: []string{
				"{{ base_url }}",
			},
			Path: []string{
				action.AbsoluteURI,
			},
		},
		Method:      string(action.Method),
		Description: action.GenerateDescription(),
		Body: &postman.Body{
			Mode: postman.BodyModeRaw,
			Raw:  "",
		},
	}
	for key, value := range postmanHeaders {
		req.Header = append(req.Header, postman.Header{Key: key, Value: value})
	}
	req.Body.SetLanguage("json")
	if action.AcceptData {

		if action.Batch {
			var data []any
			for i := 0; i < 3; i++ {
				data = append(data, ModelDataFaker(res.Schema))
			}
			req.Body.Raw = PrettyJson(data)
		} else {
			var data = ModelDataFaker(res.Schema)
			req.Body.Raw = PrettyJson(data)
		}

	}
	if action.Pagination {
		req.Url.AddQuery("page", ":page", "specify page to load (optional)")
		req.Url.AddQuery("size", ":size", "specify size of results (optional, default 10, max 100)")
	}

	res.PostmanGroup.AppendItem(postman.Item{
		Name:    action.Name,
		Request: &req,
	})
}

type Endpoint struct {
	Name              string                        `json:"name"`
	Label             string                        `json:"-"`
	Method            Method                        `json:"method"`
	URL               string                        `json:"-"`
	PKUrl             bool                          `json:"pk_url"`
	AbsoluteURI       string                        `json:"url"`
	Description       string                        `json:"description"`
	Handler           func(context *Context) *Error `json:"-"`
	Resource          *Resource                     `json:"-"`
	URLParams         []Filter                      `json:"-"`
	Batch             bool                          `json:"batch"`
	AcceptData        bool                          `json:"accept_data"`
	Filterable        bool                          `json:"filterable"`
	Pagination        bool                          `json:"pagination"`
	PostmanCollection postman.Collection            `json:"-"`
}

// Filter represents a filter for data retrieval.
// It contains the following properties:
//
// - Title: the title of the filter.
// - Type: the type of the filter.
// - Options: dictionary of options for the filter.
// - Name: the name of the filter.
// - Filter: the filter condition to be applied.
type Filter struct {
	Title   string             `json:"title,omitempty"`
	Type    string             `json:"type,omitempty"`
	Options Dictionary[string] `json:"options,omitempty"`
	Name    string             `json:"name,omitempty"`
	Filter  string             `json:"-"`
}

// Context represents the context of an HTTP request.
// It contains information about the request, the object being processed,
// the sample data, the action to be performed, the response, and the schema.
type Context struct {
	Request    *evo.Request
	DBO        *gorm.DB
	Object     reflect.Value
	Sample     interface{}
	Action     *Endpoint
	Response   *Pagination
	Schema     *schema.Schema
	Conditions []Condition
	override   *reflect.Value
	Code       int
}

type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// handler handles the incoming request and returns a response.
// It takes in a `Request` object and returns an `interface{}`.
// It creates a new `Context` object with the request, action, object, and default response.
// If the action has a handler defined
func (action *Endpoint) handler(request *evo.Request) interface{} {
	request.Write("test")
	return nil
	context := &Context{
		Request: request,
		Action:  action,
		Object:  action.Resource.Ref,
		Response: &Pagination{
			TotalPages: 1,
			Total:      1,
			Page:       1,
			Size:       1,
			Success:    true,
		},
	}

	context.Schema = action.Resource.Schema
	if action.Handler != nil {
		context.HandleError(action.Handler(context))
	} else {
		context.HandleError(&ErrorHandlerNotFound)
	}
	var response = context.PrepareResponse()
	if context.Code == 0 {
		request.Status(200)
	} else {
		request.Status(context.Code)
	}

	request.SetHeader("Content-Type", "application/json; charset=utf-8")
	request.Write(text.ToJSON(response))
	return nil
}

func (action *Endpoint) RegisterRouter() {
	switch action.Method {
	case MethodGET:
		evo.Get(action.AbsoluteURI, action.handler)
	case MethodPOST:
		evo.Post(action.AbsoluteURI, action.handler)
	case MethodPUT:
		evo.Put(action.AbsoluteURI, action.handler)
	case MethodDELETE:
		evo.Delete(action.AbsoluteURI, action.handler)
	case MethodPatch:
		evo.Patch(action.AbsoluteURI, action.handler)
	default:
		log.Fatalf("invalid method %s for %s@%s", action.Method, action.Name, action.Resource.Name)
	}
}

func (action *Endpoint) GenerateDescription() string {

	var description = []string{
		action.Description,
		"---",
	}

	if action.AcceptData {
		description = append(description, "- Accepts body in `application/json`,`application/x-www-form-urlencoded` and `multipart/form-data` format.")
	}

	if action.Batch {
		description = append(description, "- Supports batch operations. You can send multiple requests in one request body, this mode requires body `application/json` as array of objects formatted.")
	}

	if action.Pagination {
		description = append(description, "- Supports pagination. You can specify the page number and size using query parameters: `page` and `size`.")
	}

	if action.Filterable {
		description = append(description, "- Supports filterable Resources. You can filter Resources by using query parameters: `field[op]=value`. refer to [Query Parameters Explanation](https://github.com/getevo/restify/blob/master/docs/endpoints.md#query-parameters-explanation)")
	}

	if action.PKUrl {
		description = append(description, "- This endpoint requires a primary key in the URL as following format "+action.AbsoluteURI)
	}

	if action.AcceptData {
		description = append(description, "---")
		description = append(description, "### Acceptable fields and their types:")
		description = append(description, "| Field | Type | Description | Validation |")
		description = append(description, "| ------ | ------ | ------ | ------ |")
		for _, field := range action.Resource.Schema.Fields {
			var jsonField = strings.Split(field.Tag.Get("json"), ",")[0]
			if field.Tag.Get("json") == "-" || strings.Contains(field.Tag.Get("json"), "omit_decode") {
				jsonField = field.Name
			}

			if strings.TrimSpace(string(field.GORMDataType)) == "" {
				continue
			}

			var additional []string
			if field.PrimaryKey {
				additional = append(additional, "`Primary Key`")
			}
			if field.AutoIncrement {
				additional = append(additional, "`AutoIncrement`")
			}
			if field.Unique {
				additional = append(additional, "`Unique`")
			}
			if field.HasDefaultValue {
				additional = append(additional, "`Default:"+field.DefaultValue+"`")
			}
			if field.FieldType.Kind() == reflect.Ptr {
				additional = append(additional, "`Accept Null`")
			}
			if field.Size > 0 {
				additional = append(additional, "`Size:"+strconv.Itoa(field.Size)+"`")
			}
			if field.Precision > 0 {
				additional = append(additional, "`Precision:"+strconv.Itoa(field.Precision)+"`")
			}
			if field.Scale > 0 {
				additional = append(additional, "`Scale:"+strconv.Itoa(field.Scale)+"`")
			}
			if !field.Updatable {
				additional = append(additional, "`Cannot be updated`")
			}
			if !field.Creatable {
				additional = append(additional, "`Cannot be created`")
			}
			if !field.Readable {
				additional = append(additional, "`Unreadable`")
			}
			var validation = "`none`"
			if field.Tag.Get("validation") != "" {
				validation = field.Tag.Get("validation")
			}

			description = append(description, fmt.Sprintf("| `%s` | %s | %s | %s |", jsonField, field.GORMDataType, strings.Join(additional, ","), validation))
		}

	}

	if action.Filterable {
		var associations []string
		for _, field := range action.Resource.Schema.Fields {
			if strings.TrimSpace(string(field.GORMDataType)) == "" {

				if field.FieldType.Kind() == reflect.Slice {
					associations = append(associations, fmt.Sprintf("| %s | `%s` | %s |", field.Name, "has many", "associations="+field.Name))
				} else {
					associations = append(associations, fmt.Sprintf("| %s | `%s` | %s |", field.Name, "belongs to", "associations="+field.Name))
				}

			}
		}
		if len(associations) > 0 {
			description = append(description, "---")
			description = append(description, "### Loadable Associations:")
			description = append(description, "| Association | Type | URL Pattern |")
			description = append(description, "| ------ | ------ | ------ |")
			description = append(description, associations...)
			description = append(description, "\n\nmore information: [Query Parameters Explanation](https://github.com/getevo/restify/blob/master/docs/endpoints.md#query-parameters-explanation)")
		}
	}

	return strings.Join(description, "\n")
}

// CreateIndirectObject is a method of the Context type that returns a new indirect reflect.Value of the context Object's type.
func (context *Context) CreateIndirectObject() reflect.Value {
	return reflect.Indirect(reflect.New(context.Object.Type()))
}

// CreateIndirectSlice returns a new indirect reflect value of a slice of the type of the Object field in the Context.
func (context *Context) CreateIndirectSlice() reflect.Value {
	return reflect.Indirect(reflect.New(reflect.SliceOf(context.Object.Type())))
}

// PrepareResponse is a method of the Context type.
// It returns the response object of the context.
// If the response is not successful (success flag is false), it sets the data, size, page, total, total pages, and offset fields of the response to 0.
// It then returns the response object.
func (context *Context) PrepareResponse() *Pagination {
	if !context.Response.Success {
		context.Response.Data = 0
		context.Response.Size = 0
		context.Response.Page = 0
		context.Response.Total = 0
		context.Response.TotalPages = 0
		context.Response.Offset = 0
	}
	return context.Response
}

// SetResponse is a method of the Context type that sets the response data.
// It takes a response interface{} as a parameter and marshals it to JSON.
// If the response is nil, it returns immediately.
// If the response is not a slice, it marshals it as a single element slice.
// Otherwise, it marshals the response as is.
// If there is an error during marshaling, it returns immediately without setting the response.
// Note: The response is set to the `context.Request` using the `JSON` method.
func (context *Context) SetResponse(response interface{}) {
	if response == nil {
		context.Request.WriteResponse(fmt.Errorf("invalid response"))
		return
	}

	var v = reflect.ValueOf(response)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		err := context.Request.JSON([]interface{}{response})
		if err != nil {
			return
		}
	} else {
		err := context.Request.JSON(response)
		if err != nil {
			return
		}
	}

}

// HandleError is a method of the Context type that sets the error message in the Response field and marks the Response as unsuccessful.
// It takes an error parameter.
func (context *Context) HandleError(error *Error) {
	if error != nil {
		context.Response.Error = error.Message
		context.Response.Success = false
		context.Code = error.Code
	}

}

func (context *Context) RestPermission(permission Permission, object reflect.Value) bool {
	var ptr = object.Addr().Interface()
	if obj, ok := ptr.(interface {
		RestPermission(permission Permissions, context *Context) bool
	}); ok {
		return obj.RestPermission(permission.ToPermissions(), context)
	}

	if permissionHandler != nil {
		return permissionHandler(permission.ToPermissions(), context)
	}

	return true
}

func (context *Context) Error(err error, code int) *Error {
	return &Error{
		Code:    code,
		Message: err.Error(),
	}
}

func (context *Context) GetDBO() *gorm.DB {
	var dbo = db.GetContext(context, context.Request)
	return dbo
}

// Field represents a field in a data structure.
// It contains metadata about the field, such as its name, database name, type, default value, and whether it is a primary key.
type Field struct {
	Name      string `json:"label"`
	FieldName string `json:"-"`
	DBName    string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Default   string `json:"default,omitempty"`
	PK        bool   `json:"pk,omitempty"`
}

// Info represents a structured information object.
//
// It contains the following fields:
// - Name: The name of the object
// - ID: The ID of the object
// - Fields: An array of Field objects that represent the fields of the object
// - Endpoints: An array of Endpoint objects that represent the endpoints associated with the object.
type Info struct {
	Name      string      `json:"name,omitempty"`
	ID        string      `json:"id,omitempty"`
	Fields    []Field     `json:"fields,omitempty"`
	Endpoints []*Endpoint `json:"endpoints,omitempty"`
}

// FindByPrimaryKey is a method that searches for a record in the database based on the primary key values provided.
// The method takes an input parameter, which can be a struct or a
func (context *Context) FindByPrimaryKey(input interface{}) (bool, *Error) {
	var dbo = context.GetDBO()
	var association = context.Request.Query("associations").String()
	if association != "" {
		if association == "1" || association == "true" {
			dbo = dbo.Preload(clause.Associations)
		} else if association == "deep" {
			var preload = getAssociations("", context.Schema)
			for _, item := range preload {
				dbo = dbo.Preload(item)
			}
		} else {
			var ls = strings.Split(association, ",")
			for _, item := range ls {
				dbo = dbo.Preload(item)
			}
		}

	}
	var where []string
	var params []interface{}
	for _, field := range context.Action.Resource.Schema.PrimaryFields {
		var v interface{} = context.Request.Param(field.DBName).String()
		if v == "" {
			v = getValueByFieldName(input, field.Name)
		}
		where = append(where, field.DBName+" = ?")
		params = append(params, v)
	}

	var join = context.Request.Query("join").String()
	if len(join) > 0 {
		if relations := relationsMapper(join); relations != "" {
			dbo = dbo.Preload(relations)
		}
	}
	var httpErr *Error
	dbo, httpErr = filterMapper(context.Request.QueryString(), context, dbo)
	dbo = dbo.Where(strings.Join(where, " AND "), params...)

	return dbo.Take(input).RowsAffected != 0, httpErr
}

func getAssociations(prefix string, s *schema.Schema, loaded ...string) []string {
	var preload []string
	if len(loaded) == 0 {
		loaded = []string{s.Table}
	}

	var relations []*schema.Relationship
	relations = append(relations, s.Relationships.HasOne...)
	relations = append(relations, s.Relationships.BelongsTo...)
	relations = append(relations, s.Relationships.HasMany...)

	for idx, _ := range relations {

		var relation = relations[idx]

		var chunks = strings.Split(loaded[0], ".")
		if len(chunks) > 2 && chunks[len(chunks)-2] == s.Table {
			continue
		}
		if len(chunks) > 4 {
			continue
		}
		loaded[0] += "." + relation.Field.Schema.Table
		preload = append(preload, strings.TrimLeft(prefix+"."+relation.Field.Name, "."))
		for i, _ := range relation.FieldSchema.Relationships.Relations {
			var item = relation.FieldSchema.Relationships.Relations[i]
			if item.Schema.Table == s.Table {
				continue
			}
			preload = append(preload, getAssociations(prefix+"."+relation.Field.Name, item.Schema, loaded...)...)

		}

	}
	return preload
}

// getValueByFieldName retrieves the value of a field by its name from the given input object.
// It uses reflection to access the field value and returns it as an interface{}.
// If the field does not exist, it returns nil.
func getValueByFieldName(input interface{}, field string) interface{} {
	ref := reflect.ValueOf(input)
	f := reflect.Indirect(ref).FieldByName(field)
	return f.Interface()
}

// relationsMapper maps the given string of relations into a formatted nested relation string.
// It splits the input string by comma and then splits each relation by dot.
// It converts the first letter of each relation to uppercase using proper language casing rules.
// It joins the titled relation strings with dots.
// If the resulting nested relation string is not empty, it is returned. Otherwise, an empty string is returned.
func relationsMapper(joins string) string {
	relations := strings.Split(joins, ",")
	for _, relation := range relations {
		nestedRelationsSlice := strings.Split(relation, ".")
		titledSlice := make([]string, len(nestedRelationsSlice))
		for i, relation := range nestedRelationsSlice {
			titledSlice[i] = cases.Title(language.English, cases.NoLower).String(relation)
		}
		nestedRelation := strings.Join(titledSlice, ".")
		if len(nestedRelation) > 0 {
			return nestedRelation
		}
	}
	return ""
}

func (p Permissions) Has(perms ...string) bool {
	for _, s := range perms {
		for _, item := range p {

			if strings.ToUpper(string(item)) == strings.ToUpper(s) {
				return true
			}
		}
	}
	return false
}

func (p Permission) ToPermissions() Permissions {
	var chunks = strings.Split(string(p), "+")
	return chunks
}

func (context *Context) SetCondition(condition string, op string, value any) {
	context.Conditions = append(context.Conditions, Condition{condition, op, value})
}

func (context *Context) Override(value any) {
	var ref = reflect.ValueOf(value)
	for ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}

	if ref.Kind() != reflect.Struct && ref.Type() != context.Object.Type() {
		return
	}
	context.override = &ref
}

func (context *Context) applyOverrides(ptr reflect.Value) {
	if context.override == nil {
		return
	}

	for i := 0; i < context.override.NumField(); i++ {
		field := context.override.Field(i)
		if !field.IsZero() {
			ptr.Field(i).Set(field)
		}
	}

}

func (context *Context) AddValidationErrors(errs ...error) {
	if len(errs) > 0 {
		context.Response.Success = false
		context.Code = 412
		for _, item := range errs {
			var chunks = strings.SplitN(item.Error(), " ", 2)
			var v = ValidationError{
				Field: chunks[0],
			}
			if len(chunks) > 1 {
				v.Error = chunks[1]
			}
			context.Response.ValidationError = append(context.Response.ValidationError, v)
		}
	}
}
