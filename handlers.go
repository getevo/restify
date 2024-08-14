package restify

import (
	"github.com/getevo/evo/v2/lib/errors"
	"gorm.io/gorm/clause"
)

type Handler struct {
}

func (Handler) ModelInfo(context *Context) *errors.HTTPError {
	if !context.HasPermission("VIEW") {
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
	info.Endpoints = resources[context.Action.Resource.Table].Actions
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
func (Handler) Create(context *Context) *errors.HTTPError {
	if !context.HasPermission("CREATE") {
		return &ErrorPermissionDenied
	}
	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()
	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, 400)
	}

	if obj, ok := ptr.(interface{ OnBeforeCreate(context *Context) error }); ok {
		err := obj.OnBeforeCreate(context)
		if err != nil {
			return context.Error(err, 500)
		}
	}

	if obj, ok := ptr.(interface{ ValidateCreate(context *Context) error }); ok {
		if err := obj.ValidateCreate(context); err != nil {
			return context.Error(err, 412)
		}
	}

	if !context.Validate(ptr) {
		return nil
	}

	if err := dbo.Omit(clause.Associations).Create(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	if obj, ok := ptr.(interface{ OnAfterCreate(context *Context) error }); ok {
		if err := obj.OnAfterCreate(context); err != nil {
			return context.Error(err, 500)
		}
	}
	context.Response.Data = ptr
	return nil
}

func (Handler) BatchCreate(context *Context) *errors.HTTPError {
	if !context.HasPermission("CREATE") {
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
		if obj, ok := v.(interface{ OnBeforeCreate(context *Context) error }); ok {
			err := obj.OnBeforeCreate(context)
			if err != nil {
				return context.Error(err, 500)
			}
		}
		if obj, ok := v.(interface{ ValidateCreate(context *Context) error }); ok {
			if err := obj.ValidateCreate(context); err != nil {
				return context.Error(err, 412)
			}
		}
		if !context.Validate(v) {
			return nil
		}
	}

	if err := dbo.Omit(clause.Associations).Create(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	for i := 0; i < object.Len(); i++ {
		var v = object.Index(i).Addr().Interface()
		if obj, ok := v.(interface{ OnAfterCreate(context *Context) error }); ok {
			if err := obj.OnAfterCreate(context); err != nil {
				return context.Error(err, 500)
			}
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
func (Handler) Update(context *Context) *errors.HTTPError {
	if !context.HasPermission("UPDATE") {
		return &ErrorPermissionDenied
	}
	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
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
	if obj, ok := ptr.(interface{ OnBeforeUpdate(context *Context) error }); ok {
		if err := obj.OnBeforeUpdate(context); err != nil {
			return context.Error(err, 500)
		}
	}

	if obj, ok := ptr.(interface{ ValidateUpdate(context *Context) error }); ok {
		if err := obj.ValidateUpdate(context); err != nil {
			return context.Error(err, 500)
		}
	}

	if !context.Validate(ptr) {
		return nil
	}

	//evo.Dump(ptr)
	if err := dbo.Debug().Omit(clause.Associations).Save(ptr).Error; err != nil {
		return context.Error(err, 500)
	}

	if obj, ok := ptr.(interface{ OnAfterUpdate(context *Context) error }); ok {
		if err := obj.OnAfterUpdate(context); err != nil {
			return context.Error(err, 500)
		}
	}
	context.Response.Data = ptr

	return nil
}

func (Handler) BatchUpdate(context *Context) *errors.HTTPError {
	if !context.HasPermission("UPDATE") {
		return &ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	var query = context.GetDBO().Model(ptr)
	var httpErr *errors.HTTPError
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}
	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, 500)
	}
	query.Debug().Omit(clause.Associations).Updates(ptr)
	if context.Request.Query("return").String() != "" {
		var slice = context.CreateIndirectSlice()
		ptr = slice.Addr().Interface()
		query.Find(ptr)

		for i := 0; i < slice.Len(); i++ {
			var v = slice.Index(i).Addr().Interface()
			if obj, ok := v.(interface{ OnAfterUpdate(context *Context) error }); ok {
				if err := obj.OnAfterUpdate(context); err != nil {
					return context.Error(err, 500)
				}
			}

			if obj, ok := v.(interface{ OnAfterGet(context *Context) error }); ok {
				if err := obj.OnAfterGet(context); err != nil {
					return context.Error(err, 500)
				}
			}
		}
	}

	context.Response.Data = ptr

	return nil
}

// Delete deletes an object from the database.
// It takes a Context pointer as a parameter.
// It returns an error if an error occurs during the deletion process.
func (Handler) Delete(context *Context) *errors.HTTPError {
	if !context.HasPermission("DELETE") {
		return &ErrorPermissionDenied
	}
	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()
	key, httpErr := context.FindByPrimaryKey(ptr)
	if httpErr != nil {
		return httpErr
	}
	if !key {
		return &ErrorObjectNotExist
	}
	if obj, ok := ptr.(interface{ OnBeforeDelete(context *Context) error }); ok {
		if err := obj.OnBeforeDelete(context); err != nil {
			return context.Error(err, 500)
		}
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

	if obj, ok := ptr.(interface{ OnAfterDelete(context *Context) error }); ok {
		if err := obj.OnAfterDelete(context); err != nil {
			return context.Error(err, 500)
		}
	}

	return nil
}

// All queries the database and retrieves all objects based on the given context.
// It applies filters, handles OnBefore and OnAfter events, and sets the response.
// It returns an error if any occurred during the process.
func (Handler) All(context *Context) *errors.HTTPError {
	if !context.HasPermission("VIEW") {
		return &ErrorPermissionDenied
	}
	var dbo = context.GetDBO()

	var slice = context.CreateIndirectSlice()
	ptr := slice.Addr().Interface()
	if obj, ok := context.CreateIndirectObject().Addr().Interface().(interface{ OnBeforeGet(context *Context) error }); ok {
		if err := obj.OnBeforeGet(context); err != nil {
			return context.Error(err, 500)
		}
	}

	var httpErr *errors.HTTPError
	dbo, httpErr = context.ApplyFilters(dbo)
	if httpErr != nil {
		return httpErr
	}
	if err := dbo.Find(ptr).Error; err != nil {
		return context.Error(err, 500)
	}
	context.Response.Total = int64(slice.Len())
	context.Response.Size = slice.Len()

	if _, ok := context.CreateIndirectObject().Addr().Interface().(interface{ OnAfterGet(context *Context) error }); ok {
		for i := 0; i < slice.Len(); i++ {
			if obj, ok := slice.Index(i).Addr().Interface().(interface{ OnAfterGet(context *Context) error }); ok {
				if err := obj.OnAfterGet(context); err != nil {
					return context.Error(err, 500)
				}
			}
		}
	}

	context.Response.Data = ptr
	context.SetResponse(ptr)
	return nil
}

// Paginate applies pagination to a database query based on the context provided.
// It modifies the context's response object with the paginated data.
func (Handler) Paginate(context *Context) *errors.HTTPError {
	if !context.HasPermission("VIEW") {
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
	var httpErr *errors.HTTPError
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
	if _, ok := context.CreateIndirectObject().Addr().Interface().(interface{ OnAfterGet(context *Context) error }); ok {
		for i := 0; i < slice.Len(); i++ {
			if obj, ok := slice.Index(i).Addr().Interface().(interface{ OnAfterGet(context *Context) error }); ok {
				if err := obj.OnAfterGet(context); err != nil {
					return context.Error(err, 500)
				}
			}
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
func (Handler) Get(context *Context) *errors.HTTPError {
	if !context.HasPermission("VIEW") {
		return &ErrorPermissionDenied
	}
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()
	if obj, ok := ptr.(interface{ OnBeforeGet(context *Context) error }); ok {
		if err := obj.OnBeforeGet(context); err != nil {
			return context.Error(err, 500)
		}
	}
	key, err := context.FindByPrimaryKey(ptr)
	if err != nil {
		return err
	}
	if !key {
		return &ErrorObjectNotExist
	}

	if obj, ok := ptr.(interface{ OnAfterGet(context *Context) error }); ok {
		if err := obj.OnAfterGet(context); err != nil {
			return context.Error(err, 500)
		}
	}

	context.Response.Data = ptr
	return nil
}

func (h Handler) BatchDelete(context *Context) *errors.HTTPError {
	if !context.HasPermission("DELETE") {
		return &ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	var query = context.GetDBO().Model(ptr)
	var httpErr *errors.HTTPError
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return httpErr
	}
	query.Debug().Omit(clause.Associations).Delete(ptr)

	return nil
}
