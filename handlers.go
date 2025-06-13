package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm/clause"
	"regexp"
	"strings"
)

type Handler struct {
}

var columnNameRegex = regexp.MustCompile(`^\w+$`)

func (Handler) ModelInfo(context *Context) *Error {
	if !context.RestPermission(PermissionsModelInfo, context.CreateIndirectObject()) {
		return &ErrorPermissionDenied
	}
	var info = Info{
		Name: context.Object.Type().Name(),
		ID:   context.Schema.Table,
	}

	for _, item := range context.Schema.Fields {
		info.Fields = append(info.Fields, Field{
			Name:    item.Name,
			DBName:  item.DBName,
			Type:    item.FieldType.Name(),
			Default: item.DefaultValue,
			PK:      item.PrimaryKey,
		})
	}
	info.Endpoints = Resources[context.Action.Resource.Table].Actions
	context.Response.Data = info
	return nil
}

// Create takes a Context as input and creates a new object.
// It uses the context's Request and DBO to perform the creation.
// The object to be created is retrieved from the context's Object field.
// The object is parsed from the request's body using the BodyParser method.
// The object can optionally implement the OnBeforeCreate method, which is called OnBefore the creation.
// The object can optionally implement the ValidateCreate method, which is called to validate the object OnBefore creation.
// The object is then created in the database using the DBO's Create method.
// If the object implements the OnAfterCreate method, it is called OnAfter the creation.
// The created object is set as the data in the context's Response field.
// Returns an error if any error occurs during the creation process.
func (Handler) Create(context *Context) *Error {

	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()
	err := context.Request.BodyParser(ptr)
	if !context.RestPermission(PermissionCreate, object) {
		return &ErrorPermissionDenied
	}
	if err != nil {
		return context.Error(err, 400)
	}

	httpError := callBeforeCreateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	context.applyOverrides(object)

	if err := dbo.Omit(clause.Associations).Create(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	httpError = callAfterCreateHook(ptr, context)
	if httpError != nil {
		return httpError
	}
	context.Response.Data = ptr
	return nil
}

func (Handler) BatchCreate(context *Context) *Error {
	if !context.RestPermission(PermissionBatchCreate, context.CreateIndirectObject()) {
		return &ErrorPermissionDenied
	}
	var dbo = context.GetDBO()
	object := context.CreateIndirectSlice()
	ptr := object.Addr().Interface()

	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, 400)
	}

	for i := 0; i < object.Len(); i++ {
		var v = object.Index(i).Addr().Interface()
		httpError := callBeforeCreateHook(v, context)
		if httpError != nil {
			return httpError
		}
		context.applyOverrides(object.Index(i))
	}

	if err := dbo.Omit(clause.Associations).Create(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	for i := 0; i < object.Len(); i++ {
		var v = object.Index(i).Addr().Interface()
		httpError := callAfterCreateHook(v, context)
		if httpError != nil {
			return httpError
		}
		httpError = callAfterGetHook(v, context)
		if httpError != nil {
			return httpError
		}
	}

	context.Response.Data = ptr
	return nil
}

// Update updates an object in the database based on the provided context.
// It retrieves the database object, checks if it exists in the database,
// parses the request body to update the object, and executes the updates
// on the database. It also calls the OnBeforeUpdate and ValidateUpdate methods
// if they are implemented by the object to perform any necessary operations
// OnBefore and OnAfter the update. Finally, it sets the updated object as the response
// data in the context.
func (Handler) Update(context *Context) *Error {

	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	if !context.RestPermission(PermissionUpdate, object) {
		return &ErrorPermissionDenied
	}
	ptr := object.Addr().Interface()
	key, httpErr := context.FindByPrimaryKey(ptr)

	if httpErr != nil {
		return httpErr
	}
	if !key {
		return &ErrorObjectNotExist
	}
	err := context.Request.BodyParser(ptr)

	if err != nil {
		return context.Error(err, 500)
	}

	httpError := callBeforeUpdateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	context.applyOverrides(object)
	//evo.Dump(ptr)
	if err := dbo.Omit(clause.Associations).Save(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	httpError = callAfterUpdateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	context.Response.Data = ptr

	return nil
}

func (Handler) BatchUpdate(context *Context) *Error {
	if !context.RestPermission(PermissionBatchUpdate, context.CreateIndirectObject()) {
		return &ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	var query = context.GetDBO().Model(ptr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}
	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, 500)
	}
	if context.Request.Query("unsafe").String() == "" {
		stmt := query.Statement
		if stmt != nil && stmt.Clauses["WHERE"].Expression == nil {
			return &ErrorUnsafe
		}
	}

	httpError := callBeforeUpdateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	context.applyOverrides(object)
	query.Omit(clause.Associations).Where("1=1").Updates(ptr)
	if context.Request.Query("return").String() != "" {
		var slice = context.CreateIndirectSlice()
		ptr = slice.Addr().Interface()
		query.Find(ptr)

		for i := 0; i < slice.Len(); i++ {
			var v = slice.Index(i).Addr().Interface()
			httpError = callAfterUpdateHook(v, context)

			if httpError != nil {
				return context.Error(err, 500)
			}

			httpError = callAfterGetHook(v, context)
			if httpError != nil {
				return context.Error(err, 500)
			}
		}
	}

	context.Response.Data = ptr

	return nil
}

// Delete deletes an object from the database.
// It takes a Context pointer as a parameter.
// It returns an error if an error occurs during the deletion process.
func (Handler) Delete(context *Context) *Error {

	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	if !context.RestPermission(PermissionDelete, object) {
		return &ErrorPermissionDenied
	}
	ptr := object.Addr().Interface()
	key, httpErr := context.FindByPrimaryKey(ptr)
	if httpErr != nil {
		return httpErr
	}

	if !key {
		return &ErrorObjectNotExist
	}

	httpError := callBeforeDeleteHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	// Try soft-delete
	if obj, ok := ptr.(interface{ Delete(v bool) }); ok {
		obj.Delete(true)
		if err := dbo.Updates(ptr).Error; err != nil {
			return context.Error(err, 500)
		}
	} else {
		if err := dbo.Delete(ptr).Error; err != nil {
			return context.Error(err, 500)
		}
	}

	httpError = callAfterDeleteHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	return nil
}

// All queries the database and retrieves all objects based on the given context.
// It applies filters, handles OnBefore and OnAfter events, and sets the response.
// It returns an error if any occurred during the process.
func (Handler) All(context *Context) *Error {
	obj := context.CreateIndirectObject()
	if !context.RestPermission(PermissionViewAll, obj) {
		return &ErrorPermissionDenied
	}

	var dbo = context.GetDBO()

	var slice = context.CreateIndirectSlice()
	ptr := slice.Addr().Interface()

	var httpErr *Error
	dbo, httpErr = context.ApplyFilters(dbo)
	if httpErr != nil {
		return httpErr
	}
	if err := dbo.Find(ptr).Error; err != nil {
		return context.Error(err, 500)
	}
	context.Response.Total = int64(slice.Len())
	context.Response.Size = slice.Len()

	for i := 0; i < slice.Len(); i++ {
		if httpError := callAfterGetHook(slice.Index(i).Addr().Interface(), context); httpError != nil {
			return httpError
		}
	}
	evo.Dump(ptr)
	context.Response.Data = ptr
	context.SetResponse(ptr)
	return nil
}

// Paginate applies pagination to a database query based on the context provided.
// It modifies the context's response object with the paginated data.
func (Handler) Paginate(context *Context) *Error {
	obj := context.CreateIndirectObject()
	if !context.RestPermission(PermissionViewPagination, obj) {
		return &ErrorPermissionDenied
	}

	var slice = context.CreateIndirectSlice()

	if obj, ok := context.CreateIndirectObject().Addr().Interface().(interface{ OnBeforeGet(context *Context) error }); ok {
		if err := obj.OnBeforeGet(context); err != nil {
			return context.Error(err, 500)
		}
	}

	ptr := slice.Addr().Interface()
	var p Pagination
	p.SetLimit(context.Request.Query("size").Int())
	p.SetCurrentPage(context.Request.Query("page").Int())
	context.Response.Size = p.Limit
	context.Response.Offset = p.GetOffset()
	context.Response.Page = p.Page

	var query = context.GetDBO().Model(ptr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}
	query.Model(ptr).Count(&context.Response.Total)
	p.Records = int(context.Response.Total)
	p.SetPages()
	context.Response.TotalPages = p.Pages
	if err := query.Limit(p.Limit).Offset(p.GetOffset()).Find(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	for i := 0; i < slice.Len(); i++ {
		if httpError := callAfterGetHook(slice.Index(i).Addr().Interface(), context); httpError != nil {
			return httpError
		}
	}

	context.Response.Data = ptr
	context.SetResponse(ptr)
	return nil
}

// Get is a function that retrieves an object from the context.
// It performs pre- and post-get operations on the object if they are implemented.
// It finds the object by its primary key, sets it as the response data in the context, and returns nil if successful.
// If the object does not exist, it returns an error of type ErrorObjectNotExist.
// It returns an error if any operation fails.
func (Handler) Get(context *Context) *Error {
	obj := context.CreateIndirectObject()
	if !context.RestPermission(PermissionViewGet, obj) {
		return &ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	exits, err := context.FindByPrimaryKey(ptr)
	if err != nil {
		return err
	}
	if !exits {
		return &ErrorObjectNotExist
	}

	if httpError := callAfterGetHook(ptr, context); httpError != nil {
		return httpError
	}

	context.Response.Data = ptr
	return nil
}

// BatchDelete delete multiple objects in the database
func (h Handler) BatchDelete(context *Context) *Error {

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()
	if !context.RestPermission(PermissionBatchDelete, object) {
		return &ErrorPermissionDenied
	}

	var query = context.GetDBO().Model(ptr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}

	if context.Request.Query("unsafe").String() == "" {
		stmt := query.Statement
		if stmt != nil && stmt.Clauses["WHERE"].Expression == nil {
			return &ErrorUnsafe
		}
	}

	query.Omit(clause.Associations).Delete(ptr)

	return nil
}

// Set updates the collection by creating new items that don't already exist
// and removing any items that are not present in the provided list.
func (h Handler) Set(context *Context) *Error {
	if !context.RestPermission(PermissionSet, context.CreateIndirectObject()) {
		return &ErrorPermissionDenied
	}

	input := context.CreateIndirectSlice()
	loader := context.CreateIndirectSlice()
	inputPtr := input.Addr().Interface()
	loaderPtr := loader.Addr().Interface()

	err := context.Request.BodyParser(inputPtr)
	if err != nil {
		return context.Error(err, 400)
	}

	var query = context.GetDBO().Model(loaderPtr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}

	if context.Request.Query("unsafe").String() == "" {
		stmt := query.Statement
		if stmt != nil && stmt.Clauses["WHERE"].Expression == nil {
			return &ErrorUnsafe
		}
	}
	query.Unscoped().Find(loaderPtr)
	var dbo = context.GetDBO()

	for j := 0; j < loader.Len(); j++ {
		loaderItem := loader.Index(j)
		var exists = false
		for i := 0; i < input.Len(); i++ {
			inputItem := input.Index(i)
			if equal(loaderItem, inputItem) {
				exists = true
				break
			}
		}
		if !exists {
			var ptr = loaderItem.Addr().Interface()

			httpError := callBeforeDeleteHook(ptr, context)
			if httpError != nil {
				return context.Error(err, 500)
			}

			if err := dbo.Unscoped().Delete(ptr).Error; err != nil {
				return context.Error(err, 500)
			}

			httpError = callAfterDeleteHook(ptr, context)
			if httpError != nil {
				return httpError
			}

		}
	}

	for i := 0; i < input.Len(); i++ {
		inputItem := input.Index(i)
		var exists = false
		for j := 0; j < loader.Len(); j++ {
			loaderItem := loader.Index(j)
			if equal(loaderItem, inputItem) {
				exists = true
				break
			}
		}
		if !exists {
			var ptr = inputItem.Addr().Interface()
			httpError := callBeforeCreateHook(ptr, context)
			if httpError != nil {
				return context.Error(err, 500)
			}

			if obj, ok := ptr.(interface{ ValidateCreate(context *Context) error }); ok {
				if err := obj.ValidateCreate(context); err != nil {
					return context.Error(err, 412)
				}
			}
			context.applyOverrides(inputItem)
			dbo.Create(inputItem.Addr().Interface())

			httpError = callAfterCreateHook(ptr, context)
			if httpError != nil {
				return context.Error(err, 500)
			}
		}
	}

	if context.Request.Query("return").String() != "" {
		query.Unscoped().Find(loaderPtr)
		for i := 0; i < loader.Len(); i++ {
			var v = loader.Index(i).Addr().Interface()
			if httpError := callAfterGetHook(v, context); httpError != nil {
				return httpError
			}
		}
		context.Response.Data = loader.Interface()
		return nil
	}
	context.Response.Data = nil
	return nil
}

var aggregateRegex = regexp.MustCompile(`(?mi)([a-z0-9_*\-]+)\.(count|sum|min|max|avg|first|last)`)

func (h Handler) Aggregate(context *Context) *Error {
	if !context.RestPermission(PermissionAggregate, context.CreateIndirectObject()) {
		return &ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	var query = context.GetDBO().Model(ptr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}

	var fieldsInput = context.Request.Query("fields").String()
	if fieldsInput == "" {
		httpErr = &Error{
			Code:    400,
			Message: "fields parameter is required",
		}
		return httpErr
	}
	var fields = strings.Split(fieldsInput, ",")
	var _select = ""
	for _, item := range fields {
		match := aggregateRegex.FindStringSubmatch(item)
		if len(match) == 3 {
			fieldName := match[1]
			funcName := strings.ToUpper(match[2])
			var alias = fieldName + "." + strings.ToLower(funcName)
			if fieldName != "*" {
				fieldName = "`" + fieldName + "`"
			}
			_select += "," + fmt.Sprintf("%s(%s) AS `%s`", funcName, fieldName, alias)
		}
	}
	if len(_select) > 0 {
		_select = _select[1:]
		query = query.Select(_select)
	} else {
		httpErr = &Error{
			Code:    400,
			Message: "fields parameter should contain aggregate functions field_name.aggregate_function",
		}
		return httpErr
	}

	var groupByInput = context.Request.Query("group_by").String()

	if columnNameRegex.MatchString(groupByInput) {
		query = query.Group(groupByInput)
		query = query.Select(_select, groupByInput)

		var result []map[string]interface{}
		if err := query.Scan(&result).Error; err != nil {
			log.Error(err)
			return context.Error(fmt.Errorf("unable to execute aggregate query"), 500)
		}
		context.Response.Data = result
	} else {

		var result map[string]interface{}
		if err := query.Scan(&result).Error; err != nil {
			log.Error(err)
			return context.Error(fmt.Errorf("unable to execute aggregate query"), 500)
		}
		context.Response.Data = result
	}

	return nil
}
