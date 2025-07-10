package restify

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"regexp"
	"strings"
)

// Handler provides the core CRUD operation handlers for REST API endpoints.
// This struct contains methods that handle HTTP requests for create, read, update,
// delete, and other operations on registered models. Each method corresponds to
// a specific REST endpoint and handles the complete request lifecycle including
// permission checking, validation, database operations, and response formatting.
//
// The Handler methods are automatically registered as route handlers when models
// are processed during application startup. Each method follows a consistent pattern:
//  1. Permission checking using RestPermission
//  2. Input parsing and validation
//  3. Lifecycle hook execution (OnBefore* methods)
//  4. Database operations
//  5. Lifecycle hook execution (OnAfter* methods)
//  6. Response formatting and return
//
// Performance considerations:
//   - Batch operations use chunking to optimize memory usage
//   - Database queries are optimized with appropriate clauses
//   - Logging is used for monitoring and debugging
//   - Error handling provides detailed information for troubleshooting
type Handler struct {
}

// columnNameRegex validates column names to prevent SQL injection in dynamic queries.
// This regex ensures that column names contain only word characters (letters, digits, underscore)
// which is safe for use in SQL queries. It's used in filtering and ordering operations
// to validate user-provided column names before including them in database queries.
//
// Security note: This is a critical security measure to prevent SQL injection attacks
// through column name manipulation in query parameters.
var columnNameRegex = regexp.MustCompile(`^\w+$`)

// ModelInfo returns comprehensive information about a model including its fields and available endpoints.
// This method provides API introspection capabilities, allowing clients to discover:
//   - Model structure and field definitions
//   - Available REST endpoints for the model
//   - Field types, constraints, and metadata
//   - Primary key information
//
// The returned information includes:
//   - Model name and database table name
//   - Field definitions with types, database names, defaults, and primary key flags
//   - Available REST endpoints (actions) for the model
//
// This endpoint is useful for:
//   - API documentation generation
//   - Dynamic form generation in client applications
//   - API discovery and exploration tools
//   - Development and debugging
//
// Example response structure:
//
//	{
//	  "name": "User",
//	  "id": "users",
//	  "fields": [
//	    {
//	      "name": "ID",
//	      "db_name": "id",
//	      "type": "uint",
//	      "default": "",
//	      "pk": true
//	    },
//	    {
//	      "name": "Name",
//	      "db_name": "name",
//	      "type": "string",
//	      "default": "",
//	      "pk": false
//	    }
//	  ],
//	  "endpoints": [...]
//	}
//
// Security note: Requires ModelInfo permission to access model metadata.
// Performance note: This operation is read-only and relatively lightweight.
func (Handler) ModelInfo(context *Context) *Error {
	// Check if the user has permission to access model information
	if !context.RestPermission(PermissionsModelInfo, context.CreateIndirectObject()) {
		return ErrorPermissionDenied
	}

	// Build model information structure
	var info = Info{
		Name: context.Object.Type().Name(), // Go struct name
		ID:   context.Schema.Table,         // Database table name
	}

	// Extract field information from the GORM schema
	for _, item := range context.Schema.Fields {
		info.Fields = append(info.Fields, Field{
			Name:    item.Name,             // Go field name
			DBName:  item.DBName,           // Database column name
			Type:    item.FieldType.Name(), // Go type name
			Default: item.DefaultValue,     // Default value if any
			PK:      item.PrimaryKey,       // Primary key flag
		})
	}

	// Include available endpoints for this model
	info.Endpoints = Resources[context.Action.Resource.Table].Actions

	// Set the response data
	context.Response.Data = info
	return nil
}

// Create handles HTTP POST requests to create a new record in the database.
// This method implements the "C" in CRUD operations and follows the complete
// lifecycle for creating a new resource with proper validation, permission checking,
// and hook execution.
//
// Request flow:
//  1. Parse JSON request body into the model struct
//  2. Check create permissions for the user and resource
//  3. Execute OnBeforeCreate lifecycle hooks for validation and preprocessing
//  4. Apply any field overrides (e.g., setting user ID from context)
//  5. Create the record in the database (excluding associations)
//  6. Execute OnAfterCreate lifecycle hooks for post-processing
//  7. Return the created object in the response
//
// Lifecycle hooks supported:
//   - OnBeforeCreate: Called before database insertion for validation/preprocessing
//   - OnAfterCreate: Called after successful insertion for post-processing
//
// Permission requirements:
//   - User must have PermissionCreate for the specific resource
//   - Model-specific RestPermission method is called if implemented
//
// Example request:
//
//	POST /api/v1/users
//	Content-Type: application/json
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
//	  "age": 30
//	}
//
// Example response:
//
//	{
//	  "data": {
//	    "id": 123,
//	    "name": "John Doe",
//	    "email": "john@example.com",
//	    "age": 30,
//	    "created_at": "2023-01-01T12:00:00Z",
//	    "updated_at": "2023-01-01T12:00:00Z"
//	  }
//	}
//
// Error responses:
//   - 400 Bad Request: Invalid JSON or validation errors
//   - 403 Forbidden: Insufficient permissions
//   - 500 Internal Server Error: Database or server errors
//
// Performance notes:
//   - Associations are omitted from creation to prevent unintended side effects
//   - Database transaction is handled automatically by GORM
//   - Hooks can add processing overhead but provide flexibility
//
// Security notes:
//   - All input is validated through lifecycle hooks
//   - Permission checking prevents unauthorized access
//   - Field overrides can enforce security constraints (e.g., user ownership)
func (Handler) Create(context *Context) *Error {
	// Get database connection from context
	var dbo = context.GetDBO()

	// Create a new instance of the model for this request
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	// Parse the JSON request body into the model struct
	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, StatusBadRequest)
	}

	// Check if the user has permission to create this type of resource
	if !context.RestPermission(PermissionCreate, object) {
		return ErrorPermissionDenied
	}

	// Execute OnBeforeCreate lifecycle hooks for validation and preprocessing
	// This allows models to implement custom validation, data transformation,
	// or business logic before the record is created in the database
	httpError := callBeforeCreateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	// Apply any field overrides based on context (e.g., setting user ID)
	// This ensures certain fields are set based on the authenticated user
	// or other contextual information, regardless of what was sent in the request
	context.applyOverrides(object)

	// Create the record in the database, omitting associations to prevent
	// unintended creation of related records. Associations should be handled
	// separately through their own endpoints or explicit relationship management
	if err := dbo.Omit(clause.Associations).Create(ptr).Error; err != nil {
		return context.Error(err, StatusInternalServerError)
	}

	// Execute OnAfterCreate lifecycle hooks for post-processing
	// This allows models to perform actions after successful creation,
	// such as sending notifications, logging, or updating related records
	httpError = callAfterCreateHook(ptr, context)
	if httpError != nil {
		return httpError
	}

	// Set the created object as the response data
	context.Response.Data = ptr
	return nil
}

func (h Handler) BatchCreate(context *Context) *Error {
	if !context.RestPermission(PermissionBatchCreate, context.CreateIndirectObject()) {
		return ErrorPermissionDenied
	}

	object := context.CreateIndirectSlice()
	ptr := object.Addr().Interface()

	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, StatusBadRequest)
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":   "batch_create_start",
		"resource":    context.Action.Resource.Table,
		"total_items": object.Len(),
	})

	// Process in chunks to optimize memory usage
	const chunkSize = 100
	totalItems := object.Len()

	for chunkStart := 0; chunkStart < totalItems; chunkStart += chunkSize {
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > totalItems {
			chunkEnd = totalItems
		}

		if httpErr := h.processBatchCreateChunk(context, object, chunkStart, chunkEnd); httpErr != nil {
			return httpErr
		}

		LogError(nil, LogLevelDebug, map[string]interface{}{
			"operation":   "batch_create_chunk_complete",
			"resource":    context.Action.Resource.Table,
			"chunk_start": chunkStart,
			"chunk_end":   chunkEnd,
			"processed":   chunkEnd,
			"total":       totalItems,
		})
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":   "batch_create_complete",
		"resource":    context.Action.Resource.Table,
		"total_items": totalItems,
	})

	context.Response.Data = ptr
	return nil
}

// processBatchCreateChunk processes a chunk of items for batch creation
func (h Handler) processBatchCreateChunk(context *Context, object reflect.Value, start, end int) *Error {
	dbo := context.GetDBO()

	// Process before-create hooks for the chunk
	for i := start; i < end; i++ {
		v := object.Index(i).Addr().Interface()
		if httpError := callBeforeCreateHook(v, context); httpError != nil {
			LogError(httpError, LogLevelError, map[string]interface{}{
				"operation":  "batch_create_before_hook",
				"resource":   context.Action.Resource.Table,
				"item_index": i,
			})
			return httpError
		}
		context.applyOverrides(object.Index(i))
	}

	// Create chunk slice for database operation
	chunkSlice := reflect.MakeSlice(object.Type(), 0, end-start)
	for i := start; i < end; i++ {
		chunkSlice = reflect.Append(chunkSlice, object.Index(i))
	}
	chunkPtr := chunkSlice.Addr().Interface()

	// Execute database create for the chunk
	if err := dbo.Omit(clause.Associations).Create(chunkPtr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation":  "batch_create_db_operation",
			"resource":   context.Action.Resource.Table,
			"chunk_size": end - start,
		})
		return context.Error(err, StatusInternalServerError)
	}

	// Process after-create hooks for the chunk
	for i := start; i < end; i++ {
		v := object.Index(i).Addr().Interface()
		if httpError := callAfterCreateHook(v, context); httpError != nil {
			LogError(httpError, LogLevelError, map[string]interface{}{
				"operation":  "batch_create_after_hook",
				"resource":   context.Action.Resource.Table,
				"item_index": i,
			})
			return httpError
		}
		if httpError := callAfterGetHook(v, context); httpError != nil {
			return httpError
		}
	}

	return nil
}

// Update updates an object in the database based on the provided context.
// It supports both full updates (PUT) and partial updates (PATCH).
// For partial updates, only the provided fields are updated, leaving others unchanged.
// It retrieves the database object, checks if it exists in the database,
// parses the request body to update the object, and executes the updates
// on the database. It also calls the OnBeforeUpdate and ValidateUpdate methods
// if they are implemented by the object to perform any necessary operations
// OnBefore and OnAfter the update. Finally, it sets the updated object as the response
// data in the context.
func (h Handler) Update(context *Context) *Error {
	var dbo = context.GetDBO()
	object := context.CreateIndirectObject()
	if !context.RestPermission(PermissionUpdate, object) {
		return ErrorPermissionDenied
	}
	ptr := object.Addr().Interface()

	// Load existing record first
	key, httpErr := context.FindByPrimaryKey(ptr)
	if httpErr != nil {
		return httpErr
	}
	if !key {
		return ErrorObjectNotExist
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "update_start",
		"resource":  context.Action.Resource.Table,
		"method":    context.Request.Method(),
	})

	// Determine if this is a partial update (PATCH) or full update (PUT)
	isPartialUpdate := context.Request.Method() == "PATCH"

	if isPartialUpdate {
		return h.handlePartialUpdate(context, dbo, object, ptr)
	} else {
		return h.handleFullUpdate(context, dbo, object, ptr)
	}
}

// handlePartialUpdate handles PATCH operations for partial updates
func (h Handler) handlePartialUpdate(context *Context, dbo *gorm.DB, object reflect.Value, existingPtr interface{}) *Error {
	// Create a new object to hold only the fields to update
	updateObject := context.CreateIndirectObject()
	updatePtr := updateObject.Addr().Interface()

	// Parse only the provided fields from request body
	err := context.Request.BodyParser(updatePtr)
	if err != nil {
		return context.Error(err, StatusBadRequest)
	}

	LogError(nil, LogLevelDebug, map[string]interface{}{
		"operation": "partial_update_parsed",
		"resource":  context.Action.Resource.Table,
	})

	httpError := callBeforeUpdateHook(updatePtr, context)
	if httpError != nil {
		LogError(httpError, LogLevelError, map[string]interface{}{
			"operation": "partial_update_before_hook",
			"resource":  context.Action.Resource.Table,
		})
		return httpError
	}

	context.applyOverrides(updateObject)

	// Use Updates() for partial update - only updates non-zero fields
	if err := dbo.Model(existingPtr).Omit(clause.Associations).Updates(updatePtr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "partial_update_db_operation",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(err, StatusInternalServerError)
	}

	// Reload the updated record to get the complete object
	if err := dbo.First(existingPtr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "partial_update_reload",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(err, StatusInternalServerError)
	}

	httpError = callAfterUpdateHook(existingPtr, context)
	if httpError != nil {
		LogError(httpError, LogLevelError, map[string]interface{}{
			"operation": "partial_update_after_hook",
			"resource":  context.Action.Resource.Table,
		})
		return httpError
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "partial_update_complete",
		"resource":  context.Action.Resource.Table,
	})

	context.Response.Data = existingPtr
	return nil
}

// handleFullUpdate handles PUT operations for full updates
func (h Handler) handleFullUpdate(context *Context, dbo *gorm.DB, object reflect.Value, ptr interface{}) *Error {
	// Parse the complete object from request body
	err := context.Request.BodyParser(ptr)
	if err != nil {
		return context.Error(err, StatusBadRequest)
	}

	LogError(nil, LogLevelDebug, map[string]interface{}{
		"operation": "full_update_parsed",
		"resource":  context.Action.Resource.Table,
	})

	httpError := callBeforeUpdateHook(ptr, context)
	if httpError != nil {
		LogError(httpError, LogLevelError, map[string]interface{}{
			"operation": "full_update_before_hook",
			"resource":  context.Action.Resource.Table,
		})
		return httpError
	}

	context.applyOverrides(object)

	// Use Save() for full update - replaces the entire record
	if err := dbo.Omit(clause.Associations).Save(ptr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "full_update_db_operation",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(err, StatusInternalServerError)
	}

	httpError = callAfterUpdateHook(ptr, context)
	if httpError != nil {
		LogError(httpError, LogLevelError, map[string]interface{}{
			"operation": "full_update_after_hook",
			"resource":  context.Action.Resource.Table,
		})
		return httpError
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "full_update_complete",
		"resource":  context.Action.Resource.Table,
	})

	context.Response.Data = ptr
	return nil
}

func (h Handler) BatchUpdate(context *Context) *Error {
	if !context.RestPermission(PermissionBatchUpdate, context.CreateIndirectObject()) {
		return ErrorPermissionDenied
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
		return context.Error(err, StatusInternalServerError)
	}

	if context.Request.Query("unsafe").String() == "" {
		stmt := query.Statement
		if stmt != nil && stmt.Clauses["WHERE"].Expression == nil {
			return ErrorUnsafe
		}
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "batch_update_start",
		"resource":  context.Action.Resource.Table,
	})

	httpError := callBeforeUpdateHook(ptr, context)
	if httpError != nil {
		LogError(httpError, LogLevelError, map[string]interface{}{
			"operation": "batch_update_before_hook",
			"resource":  context.Action.Resource.Table,
		})
		return httpError
	}

	context.applyOverrides(object)

	// Execute the batch update
	if err := query.Omit(clause.Associations).Where("1=1").Updates(ptr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "batch_update_db_operation",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(err, StatusInternalServerError)
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "batch_update_complete",
		"resource":  context.Action.Resource.Table,
	})

	// Handle optional return data with memory optimization
	if context.Request.Query("return").String() != "" {
		return h.handleBatchUpdateResponse(context, query)
	}

	context.Response.Data = ptr
	return nil
}

// handleBatchUpdateResponse handles the optional return data for batch update with chunked processing
func (h Handler) handleBatchUpdateResponse(context *Context, query *gorm.DB) *Error {
	slice := context.CreateIndirectSlice()
	ptr := slice.Addr().Interface()

	// Load updated records
	if err := query.Find(ptr).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "batch_update_load_response",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(err, StatusInternalServerError)
	}

	totalItems := slice.Len()
	LogError(nil, LogLevelDebug, map[string]interface{}{
		"operation":   "batch_update_response_loaded",
		"resource":    context.Action.Resource.Table,
		"total_items": totalItems,
	})

	// Process hooks in chunks to optimize memory usage
	const chunkSize = 100
	for chunkStart := 0; chunkStart < totalItems; chunkStart += chunkSize {
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > totalItems {
			chunkEnd = totalItems
		}

		for i := chunkStart; i < chunkEnd; i++ {
			v := slice.Index(i).Addr().Interface()

			if httpError := callAfterUpdateHook(v, context); httpError != nil {
				LogError(httpError, LogLevelError, map[string]interface{}{
					"operation":  "batch_update_after_hook",
					"resource":   context.Action.Resource.Table,
					"item_index": i,
				})
				return httpError
			}

			if httpError := callAfterGetHook(v, context); httpError != nil {
				return httpError
			}
		}

		LogError(nil, LogLevelDebug, map[string]interface{}{
			"operation":   "batch_update_hooks_chunk_complete",
			"resource":    context.Action.Resource.Table,
			"chunk_start": chunkStart,
			"chunk_end":   chunkEnd,
		})
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
		return ErrorPermissionDenied
	}
	ptr := object.Addr().Interface()
	key, httpErr := context.FindByPrimaryKey(ptr)
	if httpErr != nil {
		return httpErr
	}

	if !key {
		return ErrorObjectNotExist
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
		return ErrorPermissionDenied
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

	context.Response.Data = ptr
	context.SetResponse(ptr)
	return nil
}

// Paginate applies pagination to a database query based on the context provided.
// It modifies the context's response object with the paginated data.
func (Handler) Paginate(context *Context) *Error {
	obj := context.CreateIndirectObject()
	if !context.RestPermission(PermissionViewPagination, obj) {
		return ErrorPermissionDenied
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
		return ErrorPermissionDenied
	}

	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	exists, err := context.FindByPrimaryKey(ptr)
	if err != nil {
		return err
	}
	if !exists {
		return ErrorObjectNotExist
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
		return ErrorPermissionDenied
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
			return ErrorUnsafe
		}
	}

	query.Omit(clause.Associations).Delete(ptr)

	return nil
}

// Set updates the collection by creating new items that don't already exist
// and removing any items that are not present in the provided list.
func (h Handler) Set(context *Context) *Error {
	if !context.RestPermission(PermissionSet, context.CreateIndirectObject()) {
		return ErrorPermissionDenied
	}

	// Parse and validate input
	input, loader, httpErr := h.parseSetInput(context)
	if httpErr != nil {
		return httpErr
	}

	// Load existing items with filters
	query, httpErr := h.buildSetQuery(context, loader)
	if httpErr != nil {
		return httpErr
	}

	// Process deletions for items not in input
	if httpErr := h.processSetDeletions(context, input, loader); httpErr != nil {
		return httpErr
	}

	// Process creations for new items
	if httpErr := h.processSetCreations(context, input, loader); httpErr != nil {
		return httpErr
	}

	// Handle optional return data
	return h.handleSetResponse(context, query, loader)
}

// parseSetInput parses the request body and creates necessary data structures
func (h Handler) parseSetInput(context *Context) (input, loader reflect.Value, httpErr *Error) {
	input = context.CreateIndirectSlice()
	loader = context.CreateIndirectSlice()
	inputPtr := input.Addr().Interface()

	err := context.Request.BodyParser(inputPtr)
	if err != nil {
		return input, loader, context.Error(err, StatusBadRequest)
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":   "set_parse_input",
		"items_count": input.Len(),
		"resource":    context.Action.Resource.Table,
	})

	return input, loader, nil
}

// buildSetQuery builds and executes the query to load existing items
func (h Handler) buildSetQuery(context *Context, loader reflect.Value) (*gorm.DB, *Error) {
	loaderPtr := loader.Addr().Interface()
	query := context.GetDBO().Model(loaderPtr)

	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		return nil, httpErr
	}

	if context.Request.Query("unsafe").String() == "" {
		stmt := query.Statement
		if stmt != nil && stmt.Clauses["WHERE"].Expression == nil {
			return nil, ErrorUnsafe
		}
	}

	query.Unscoped().Find(loaderPtr)

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":      "set_load_existing",
		"existing_count": loader.Len(),
		"resource":       context.Action.Resource.Table,
	})

	return query, nil
}

// processSetDeletions handles deletion of items not present in input
func (h Handler) processSetDeletions(context *Context, input, loader reflect.Value) *Error {
	dbo := context.GetDBO()
	deletedCount := 0

	for j := 0; j < loader.Len(); j++ {
		loaderItem := loader.Index(j)
		if !h.itemExistsInSlice(loaderItem, input) {
			ptr := loaderItem.Addr().Interface()

			if httpError := callBeforeDeleteHook(ptr, context); httpError != nil {
				return httpError
			}

			if err := dbo.Unscoped().Delete(ptr).Error; err != nil {
				LogError(err, LogLevelError, map[string]interface{}{
					"operation": "set_delete_item",
					"resource":  context.Action.Resource.Table,
				})
				return context.Error(err, StatusInternalServerError)
			}

			if httpError := callAfterDeleteHook(ptr, context); httpError != nil {
				return httpError
			}

			deletedCount++
		}
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":     "set_deletions_complete",
		"deleted_count": deletedCount,
		"resource":      context.Action.Resource.Table,
	})

	return nil
}

// processSetCreations handles creation of new items not present in existing data
func (h Handler) processSetCreations(context *Context, input, loader reflect.Value) *Error {
	dbo := context.GetDBO()
	createdCount := 0

	for i := 0; i < input.Len(); i++ {
		inputItem := input.Index(i)
		if !h.itemExistsInSlice(inputItem, loader) {
			ptr := inputItem.Addr().Interface()

			if httpError := callBeforeCreateHook(ptr, context); httpError != nil {
				return httpError
			}

			if obj, ok := ptr.(interface{ ValidateCreate(context *Context) error }); ok {
				if err := obj.ValidateCreate(context); err != nil {
					return context.Error(err, StatusBadRequest)
				}
			}

			context.applyOverrides(inputItem)
			if err := dbo.Create(inputItem.Addr().Interface()).Error; err != nil {
				LogError(err, LogLevelError, map[string]interface{}{
					"operation": "set_create_item",
					"resource":  context.Action.Resource.Table,
				})
				return context.Error(err, StatusInternalServerError)
			}

			if httpError := callAfterCreateHook(ptr, context); httpError != nil {
				return httpError
			}

			createdCount++
		}
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":     "set_creations_complete",
		"created_count": createdCount,
		"resource":      context.Action.Resource.Table,
	})

	return nil
}

// handleSetResponse handles the optional return data for Set operation
func (h Handler) handleSetResponse(context *Context, query *gorm.DB, loader reflect.Value) *Error {
	if context.Request.Query("return").String() != "" {
		loaderPtr := loader.Addr().Interface()
		query.Unscoped().Find(loaderPtr)

		for i := 0; i < loader.Len(); i++ {
			v := loader.Index(i).Addr().Interface()
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

// itemExistsInSlice checks if an item exists in a slice using the equal function
func (h Handler) itemExistsInSlice(item reflect.Value, slice reflect.Value) bool {
	for i := 0; i < slice.Len(); i++ {
		if equal(item, slice.Index(i)) {
			return true
		}
	}
	return false
}

var aggregateRegex = regexp.MustCompile(`(?mi)([a-z0-9_*\-]+)\.(count|sum|min|max|avg|first|last)`)

func (h Handler) Aggregate(context *Context) *Error {
	if !context.RestPermission(PermissionAggregate, context.CreateIndirectObject()) {
		return ErrorPermissionDenied
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation": "aggregate_start",
		"resource":  context.Action.Resource.Table,
	})

	// Build base query with filters
	query, httpErr := h.buildAggregateQuery(context)
	if httpErr != nil {
		return httpErr
	}

	// Parse and validate fields parameter
	selectClause, httpErr := h.parseAggregateFields(context)
	if httpErr != nil {
		return httpErr
	}

	// Apply select clause to query
	query = query.Select(selectClause)

	// Execute query and return results
	return h.executeAggregateQuery(context, query, selectClause)
}

// buildAggregateQuery builds the base query with filters for aggregation
func (h Handler) buildAggregateQuery(context *Context) (*gorm.DB, *Error) {
	object := context.CreateIndirectObject()
	ptr := object.Addr().Interface()

	query := context.GetDBO().Model(ptr)
	var httpErr *Error
	query, httpErr = context.ApplyFilters(query)
	if httpErr != nil {
		LogError(httpErr, LogLevelError, map[string]interface{}{
			"operation": "aggregate_build_query",
			"resource":  context.Action.Resource.Table,
		})
		return nil, httpErr
	}

	LogError(nil, LogLevelDebug, map[string]interface{}{
		"operation": "aggregate_query_built",
		"resource":  context.Action.Resource.Table,
	})

	return query, nil
}

// parseAggregateFields parses and validates the fields parameter for aggregation
func (h Handler) parseAggregateFields(context *Context) (string, *Error) {
	fieldsInput := context.Request.Query("fields").String()
	if fieldsInput == "" {
		LogError(fmt.Errorf("fields parameter missing"), LogLevelWarn, map[string]interface{}{
			"operation": "aggregate_parse_fields",
			"resource":  context.Action.Resource.Table,
		})
		return "", &Error{
			Code:    StatusBadRequest,
			Message: "fields parameter is required",
		}
	}

	fields := strings.Split(fieldsInput, ",")
	var selectParts []string

	for _, item := range fields {
		match := aggregateRegex.FindStringSubmatch(item)
		if len(match) == 3 {
			fieldName := match[1]
			funcName := strings.ToUpper(match[2])
			alias := fieldName + "." + strings.ToLower(funcName)

			if fieldName != "*" {
				fieldName = "`" + fieldName + "`"
			}

			selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS `%s`", funcName, fieldName, alias))
		}
	}

	if len(selectParts) == 0 {
		LogError(fmt.Errorf("no valid aggregate functions found"), LogLevelWarn, map[string]interface{}{
			"operation":    "aggregate_parse_fields",
			"resource":     context.Action.Resource.Table,
			"fields_input": fieldsInput,
		})
		return "", &Error{
			Code:    StatusBadRequest,
			Message: "fields parameter should contain aggregate functions field_name.aggregate_function",
		}
	}

	selectClause := strings.Join(selectParts, ",")

	LogError(nil, LogLevelDebug, map[string]interface{}{
		"operation":     "aggregate_fields_parsed",
		"resource":      context.Action.Resource.Table,
		"fields_count":  len(selectParts),
		"select_clause": selectClause,
	})

	return selectClause, nil
}

// executeAggregateQuery executes the aggregate query and handles grouping
func (h Handler) executeAggregateQuery(context *Context, query *gorm.DB, selectClause string) *Error {
	groupByInput := context.Request.Query("group_by").String()

	if columnNameRegex.MatchString(groupByInput) {
		// Execute grouped aggregation
		return h.executeGroupedAggregation(context, query, selectClause, groupByInput)
	} else {
		// Execute simple aggregation
		return h.executeSimpleAggregation(context, query)
	}
}

// executeGroupedAggregation executes aggregation with GROUP BY clause
func (h Handler) executeGroupedAggregation(context *Context, query *gorm.DB, selectClause, groupBy string) *Error {
	query = query.Group(groupBy).Select(selectClause, groupBy)

	var result []map[string]interface{}
	if err := query.Scan(&result).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "aggregate_grouped_execution",
			"resource":  context.Action.Resource.Table,
			"group_by":  groupBy,
		})
		return context.Error(fmt.Errorf("unable to execute aggregate query"), StatusInternalServerError)
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":    "aggregate_grouped_complete",
		"resource":     context.Action.Resource.Table,
		"group_by":     groupBy,
		"result_count": len(result),
	})

	context.Response.Data = result
	return nil
}

// executeSimpleAggregation executes aggregation without GROUP BY clause
func (h Handler) executeSimpleAggregation(context *Context, query *gorm.DB) *Error {
	var result map[string]interface{}
	if err := query.Scan(&result).Error; err != nil {
		LogError(err, LogLevelError, map[string]interface{}{
			"operation": "aggregate_simple_execution",
			"resource":  context.Action.Resource.Table,
		})
		return context.Error(fmt.Errorf("unable to execute aggregate query"), StatusInternalServerError)
	}

	LogError(nil, LogLevelInfo, map[string]interface{}{
		"operation":     "aggregate_simple_complete",
		"resource":      context.Action.Resource.Table,
		"result_fields": len(result),
	})

	context.Response.Data = result
	return nil
}
