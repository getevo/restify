package restify

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/errors"
	"github.com/getevo/evo/v2/lib/text"
	"github.com/gofiber/fiber/v2/log"
	"github.com/iancoleman/strcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

// resources is a map that holds a collection of *Resource objects.
var resources = map[string]*Resource{}

type Method string

const (
	MethodGET    Method = "GET"
	MethodPOST   Method = "POST"
	MethodPatch  Method = "POST"
	MethodPUT    Method = "PUT"
	MethodDELETE Method = "DELETE"
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
}

type Endpoint struct {
	Name        string                                   `json:"name"`
	Label       string                                   `json:"-"`
	Method      Method                                   `json:"method"`
	URL         string                                   `json:"-"`
	PKUrl       bool                                     `json:"pk_url"`
	AbsoluteURI string                                   `json:"url"`
	Description string                                   `json:"description"`
	Handler     func(context *Context) *errors.HTTPError `json:"-"`
	Resource    *Resource                                `json:"-"`
	URLParams   []Filter                                 `json:"-"`
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
	Request  *evo.Request
	Object   reflect.Value
	Sample   interface{}
	Action   *Endpoint
	Response *Pagination
	Schema   *schema.Schema
	Code     int
}

// handler handles the incoming request and returns a response.
// It takes in a `Request` object and returns an `interface{}`.
// It creates a new `Context` object with the request, action, object, and default response.
// If the action has a handler defined
func (action *Endpoint) handler(request *evo.Request) interface{} {
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
	default:
		log.Fatalf("invalid method %s for %s@%s", action.Method, action.Name, action.Resource.Name)
	}
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
func (context *Context) HandleError(error *errors.HTTPError) {
	if error != nil && len(error.Errors) > 0 {
		context.Response.Error = error.Errors[0]
		context.Response.Success = false
		context.Code = error.StatusCode
	}

}

func (context *Context) HasPermission(s string) bool {
	return true
}

func (context *Context) Error(err error, code int) *errors.HTTPError {
	return &errors.HTTPError{
		StatusCode: code,
		Errors:     []string{err.Error()},
	}
}

func (context *Context) GetDBO() *gorm.DB {
	var dbo = evo.GetDBO()
	if context.Request.Header("language") != "" {
		dbo = db.Set("lang", context.Request.Header("language"))
	} else {
		if context.Request.Cookie("l10n-language") != "" {
			dbo = db.Set("lang", context.Request.Cookie("l10n-language"))
		}
	}
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
func (context *Context) FindByPrimaryKey(input interface{}) (bool, *errors.HTTPError) {
	var dbo = context.GetDBO().Debug()
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
	var httpErr *errors.HTTPError
	dbo, httpErr = filterMapper(context.Request.QueryString(), context, dbo)
	return dbo.Where(strings.Join(where, " AND "), params...).Take(input).RowsAffected != 0, httpErr
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
