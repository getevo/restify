/*
Package restify provides a comprehensive REST API framework built on top of the EVO framework.
It offers automatic CRUD operations, authentication, permissions, and API documentation generation
for GORM models with minimal configuration.

# Overview

Restify is designed to rapidly develop REST APIs by automatically generating endpoints for your
GORM models. It provides a complete set of CRUD operations, advanced filtering, pagination,
batch operations, and comprehensive error handling out of the box.

# Key Features

  - Automatic CRUD endpoint generation for GORM models
  - Advanced filtering and search capabilities
  - Built-in pagination with customizable page sizes
  - Batch operations for create, update, and delete
  - Comprehensive permission system with fine-grained control
  - Input validation and sanitization
  - Structured error handling with detailed error information
  - Postman collection generation for API documentation
  - Lifecycle hooks for custom business logic
  - Multi-language support
  - Performance optimizations with database query debugging

# Quick Start

To get started with Restify, you need to:

1. Define your GORM models with the restify.API embedded struct
2. Register your models using UseModel()
3. Configure the framework in your EVO application

Example model definition:

	type User struct {
		ID        uint      `gorm:"primaryKey" json:"id"`
		Name      string    `gorm:"size:100;not null" json:"name" validation:"required"`
		Email     string    `gorm:"size:255;unique" json:"email" validation:"required,email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		restify.API         // Enables REST endpoints
	}

	func (User) TableName() string {
		return "users"
	}

Example application setup:

	package main

	import (
		"github.com/getevo/evo/v2"
		"github.com/getevo/restify"
		"your-project/models"
	)

	type App struct{}

	func (app App) Register() error {
		// Configure restify
		restify.SetPrefix("/api/v1")
		restify.EnablePostman()

		// Register models
		db.UseModel(models.User{})
		evo.GetDBO().AutoMigrate(&models.User{})
		return nil
	}

	func main() {
		evo.Setup()
		evo.Register(restify.App{}, App{})
		evo.Run()
	}

# Generated Endpoints

For each registered model, Restify automatically generates the following endpoints:

  - GET    /api/v1/{model}           - List all records with pagination
  - GET    /api/v1/{model}/{id}      - Get a specific record by ID
  - POST   /api/v1/{model}           - Create a new record
  - PUT    /api/v1/{model}/{id}      - Update a specific record (full update)
  - PATCH  /api/v1/{model}/{id}      - Update a specific record (partial update)
  - DELETE /api/v1/{model}/{id}      - Delete a specific record
  - POST   /api/v1/{model}/batch     - Batch create multiple records
  - PUT    /api/v1/{model}/batch     - Batch update multiple records
  - DELETE /api/v1/{model}/batch     - Batch delete multiple records
  - POST   /api/v1/{model}/set       - Set operation (replace collection)
  - GET    /api/v1/{model}/aggregate - Aggregate operations (count, sum, avg, etc.)

# Advanced Filtering

Restify supports advanced filtering through query parameters:

	// Basic filtering
	GET /api/v1/users?name=John&email=john@example.com

	// Comparison operators
	GET /api/v1/users?age__gte=18&age__lt=65

	// Text search
	GET /api/v1/users?name__contains=John&email__startswith=john

	// Date filtering
	GET /api/v1/users?created_at__gte=2023-01-01&created_at__lt=2024-01-01

	// Ordering
	GET /api/v1/users?order_by=name&order_direction=asc

	// Pagination
	GET /api/v1/users?page=2&page_size=20

# Lifecycle Hooks

Implement lifecycle hooks in your models for custom business logic:

	func (u *User) OnBeforeCreate(context *restify.Context) error {
		// Hash password, validate business rules, etc.
		u.Password = hashPassword(u.Password)
		return nil
	}

	func (u *User) OnAfterCreate(context *restify.Context) error {
		// Send welcome email, log activity, etc.
		sendWelcomeEmail(u.Email)
		return nil
	}

	func (u *User) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
		// Custom permission logic
		return context.User.IsAdmin() || context.User.ID == u.ID
	}

# Global Hooks

Register global hooks that apply to all models:

	restify.OnBeforeSave(func(obj any, c *restify.Context) error {
		// Global validation, auditing, etc.
		return nil
	})

	restify.OnAfterDelete(func(obj any, c *restify.Context) error {
		// Global cleanup, logging, etc.
		return nil
	})

# Permission System

Restify provides a comprehensive permission system:

	// Set global permission handler
	restify.SetPermissionHandler(func(permissions restify.Permissions, context *restify.Context) bool {
		return context.User.HasPermission(permissions.Action, permissions.Resource)
	})

	// Model-specific permissions
	func (u *User) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
		switch permissions.Action {
		case restify.ActionRead:
			return true // Anyone can read
		case restify.ActionCreate, restify.ActionUpdate, restify.ActionDelete:
			return context.User.IsAdmin() || context.User.ID == u.ID
		}
		return false
	}

# Error Handling

Restify provides a comprehensive, structured error handling system designed to provide detailed
error information for both developers and API consumers. The error handling system operates
at multiple levels and supports various error types with consistent formatting and logging.

## Error Types and Structure

Restify defines several specialized error types, all based on the core Error struct:

### Core Error Structure
The base Error struct provides structured error information:

	type Error struct {
		Code      int                    `json:"code"`        // HTTP status code
		Message   string                 `json:"message"`     // Human-readable error message
		ErrorCode string                 `json:"error_code"`  // Machine-readable error code
		Details   map[string]interface{} `json:"details,omitempty"` // Additional error details
		Timestamp time.Time              `json:"timestamp"`   // When the error occurred
		TraceID   string                 `json:"trace_id,omitempty"` // Trace ID for debugging
		Cause     error                  `json:"-"`           // Underlying Go error (not serialized)
	}

### Specialized Error Types

 1. **DatabaseError**: For database operation failures
    type DatabaseError struct {
    *Error
    Query     string `json:"query,omitempty"`     // SQL query that failed
    Operation string `json:"operation,omitempty"` // Database operation type
    }

 2. **PermissionError**: For authorization failures
    type PermissionError struct {
    *Error
    Resource string `json:"resource,omitempty"` // Resource being accessed
    Action   string `json:"action,omitempty"`   // Action being performed
    UserID   string `json:"user_id,omitempty"`  // User attempting the action
    }

 3. **AuthenticationError**: For authentication failures
    type AuthenticationError struct {
    *Error
    Reason string `json:"reason,omitempty"` // Specific authentication failure reason
    }

 4. **ValidationError**: For field-level validation failures
    type ValidationError struct {
    Field string      `json:"field"`           // Field name that failed validation
    Error string      `json:"error"`           // Validation error message
    Value interface{} `json:"value,omitempty"` // Value that failed validation
    Rule  string      `json:"rule,omitempty"`  // Validation rule that was violated
    }

## Error Flow and Processing

### 1. Handler Error Flow
All REST endpoint handlers return *Error types. The error flow follows this pattern:

	func (Handler) Create(context *Context) *Error {
		// 1. Parse and validate input
		if err := context.Request.BodyParser(ptr); err != nil {
			return context.Error(err, StatusBadRequest) // Convert to structured error
		}

		// 2. Check permissions
		if !context.RestPermission(PermissionCreate, object) {
			return ErrorPermissionDenied // Return predefined error
		}

		// 3. Execute lifecycle hooks (may return errors)
		if httpError := callBeforeCreateHook(ptr, context); httpError != nil {
			return httpError // Return structured error from hooks
		}

		// 4. Perform database operations
		if err := dbo.Create(ptr).Error; err != nil {
			return context.Error(err, StatusInternalServerError) // Convert DB error
		}

		return nil // Success - no error
	}

### 2. Error Processing in Request Handler
The main request handler processes errors through HandleError:

	func (action *Endpoint) handler(request *evo.Request) interface{} {
		context := &Context{...}

		// Execute handler and process any returned error
		context.HandleError(action.Handler(context))

		// Set appropriate HTTP status code
		if context.Code == 0 {
			request.Status(200)
		} else {
			request.Status(context.Code) // Use error's HTTP code
		}

		return request.JSON(response)
	}

### 3. HandleError Method
The HandleError method processes structured errors:

	func (context *Context) HandleError(error *Error) {
		if error != nil {
			context.Response.Error = error.Message    // Set error message in response
			context.Response.Success = false          // Mark response as failed
			context.Code = error.Code                 // Set HTTP status code

			// Log error with comprehensive context
			LogError(error, LogLevelError, map[string]interface{}{
				"http_code":  error.Code,
				"error_code": error.ErrorCode,
				"trace_id":   error.TraceID,
				"endpoint":   context.Request.Path(),
				"method":     context.Request.Method(),
			})
		}
	}

## Error Creation and Conversion

### Creating Structured Errors
Restify provides several functions for creating structured errors:

	// Basic structured error
	err := NewError("Something went wrong", 500)

	// Structured error with error code
	err := NewStructuredError("Invalid input", 400, "VALIDATION_ERROR")

	// Wrap existing error with context
	err := WrapError(originalErr, "Database operation failed", 500, "DATABASE_ERROR")

	// Specialized error types
	dbErr := NewDatabaseError("Query failed", "SELECT", originalErr)
	permErr := NewPermissionError("Access denied", "users", "create")
	authErr := NewAuthenticationError("Invalid token", "token_expired")

### Converting Go Errors to Structured Errors
The Context.Error method converts standard Go errors to structured errors:

	func (context *Context) Error(err error, code int) *Error {
		// Determine error code based on HTTP status
		var errorCode string
		switch code {
		case StatusBadRequest:
			errorCode = ErrorCodeBadRequest
		case StatusUnauthorized:
			errorCode = ErrorCodeUnauthorized
		case StatusForbidden:
			errorCode = ErrorCodeForbidden
		case StatusNotFound:
			errorCode = ErrorCodeNotFound
		case StatusInternalServerError:
			errorCode = ErrorCodeInternal
		default:
			errorCode = ErrorCodeInternal
		}

		return WrapError(err, err.Error(), code, errorCode)
	}

## Lifecycle Hook Error Handling

Lifecycle hooks can return regular Go errors, which are automatically converted to structured errors:

	// In model lifecycle hooks
	func (u *User) OnBeforeCreate(context *restify.Context) error {
		if u.Age < 18 {
			return fmt.Errorf("age must be at least 18") // Regular Go error
		}
		return nil
	}

	// Hook error conversion (internal)
	func callBeforeCreateHook(obj any, c *Context) *Error {
		err := callHook(obj, c, _onBeforeCreateCallbacks)
		if err != nil {
			return c.Error(err, 500) // Convert to structured error
		}
		return nil
	}

## Validation Error Handling

Validation errors are handled separately and collected into the response:

	func (context *Context) AddValidationErrors(errs ...error) {
		if len(errs) > 0 {
			context.Response.Success = false
			context.Code = 412 // Precondition Failed

			for _, item := range errs {
				// Parse error message to extract field name
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

## Error Response Format

Errors are returned in a consistent JSON format:

### Single Error Response

	{
		"success": false,
		"error": "User not found",
		"code": 404,
		"error_code": "NOT_FOUND_ERROR",
		"timestamp": "2023-01-01T12:00:00Z",
		"trace_id": "abc123",
		"details": {
			"resource": "users",
			"id": "123"
		}
	}

### Validation Error Response

	{
		"success": false,
		"code": 412,
		"validation_error": [
			{
				"field": "email",
				"error": "must be a valid email address",
				"value": "invalid-email",
				"rule": "email"
			},
			{
				"field": "age",
				"error": "must be at least 18",
				"value": 16,
				"rule": "min"
			}
		]
	}

## Error Logging and Monitoring

All errors are automatically logged with comprehensive context:

	func LogError(err error, level string, context map[string]interface{}) {
		// Get caller information for debugging
		_, file, line, ok := runtime.Caller(1)
		if ok {
			context["file"] = file
			context["line"] = line
		}

		// Log with appropriate level
		switch level {
		case LogLevelError:
			log.Errorf("Error: %v, Context: %+v", err, context)
		case LogLevelWarn:
			log.Warningf("Warning: %v, Context: %+v", err, context)
		// ... other levels
		}
	}

## Panic Recovery

Restify includes panic recovery to convert panics to structured errors:

	func RecoverFromPanic() *Error {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic recovered: %v", r)
			LogError(err, LogLevelError, map[string]interface{}{
				"panic_value": r,
			})
			return NewStructuredError(MessageInternalError, StatusInternalServerError, ErrorCodeInternal)
		}
		return nil
	}

## Predefined Errors

Restify provides commonly used predefined errors:

	var ErrorObjectNotExist = NewStructuredError(MessageObjectNotExist, StatusNotFound, ErrorCodeNotFound)
	var ErrorColumnNotExist = NewStructuredError(MessageColumnNotExist, StatusInternalServerError, ErrorCodeInternal)
	var ErrorPermissionDenied = NewStructuredError(MessagePermissionDenied, StatusForbidden, ErrorCodePermission)
	var ErrorUnauthorized = NewStructuredError(MessageUnauthorized, StatusUnauthorized, ErrorCodeAuthentication)
	var ErrorHandlerNotFound = NewStructuredError(MessageHandlerNotFound, StatusNotFound, ErrorCodeNotFound)
	var ErrorUnsafe = NewStructuredError(MessageUnsafeRequest, StatusBadRequest, ErrorCodeBadRequest)

## Best Practices for Error Handling

1. **Use Structured Errors**: Always return *Error types from handlers
2. **Provide Context**: Include relevant details in error messages and context
3. **Use Appropriate HTTP Codes**: Match HTTP status codes to error types
4. **Log Comprehensively**: Include request context in error logs
5. **Handle Validation Separately**: Use AddValidationErrors for field-level validation
6. **Implement Custom Errors**: Create domain-specific error types when needed
7. **Test Error Scenarios**: Ensure error handling works correctly in all cases

## Custom Error Implementation

For custom business logic errors:

	// Custom error in model hooks
	func (u *User) OnBeforeCreate(context *restify.Context) error {
		if u.Email == "" {
			return restify.NewValidationError("email", "is required", u.Email)
		}

		// Check for duplicate email
		var count int64
		context.GetDBO().Model(&User{}).Where("email = ?", u.Email).Count(&count)
		if count > 0 {
			return fmt.Errorf("email already exists")
		}

		return nil
	}

	// Custom error types for specific domains
	type BusinessLogicError struct {
		*restify.Error
		BusinessRule string `json:"business_rule,omitempty"`
	}

	func NewBusinessLogicError(message, rule string) *BusinessLogicError {
		return &BusinessLogicError{
			Error: restify.NewStructuredError(message, 422, "BUSINESS_LOGIC_ERROR"),
			BusinessRule: rule,
		}
	}

# Performance Considerations

  - Use database indexes on frequently filtered fields
  - Implement proper pagination for large datasets
  - Use Preload() for eager loading relationships in hooks
  - Enable query debugging in development: ?debug=restify
  - Monitor slow queries with database configuration
  - Consider caching for read-heavy operations
  - Use batch operations for bulk data manipulation

# Configuration

Configure Restify in your EVO application:

	// Set API prefix (default: "/admin/rest")
	restify.SetPrefix("/api/v1")

	// Enable Postman collection generation
	restify.EnablePostman()

	// Set Postman authentication
	restify.SetPostmanAuthorization(restify.AuthTypeBearer)

	// Register ready callbacks
	restify.Ready(func() {
		// Custom initialization logic
	})

# Database Configuration

Configure your database in config.yml:

	Database:
	  Type: sqlite          # sqlite, mysql, sqlserver
	  Database: "app.db"    # database name/file
	  Debug: "3"            # enable SQL logging
	  Enabled: true
	  MaxIdleConns: "10"
	  MaxOpenConns: "100"
	  ConnMaxLifTime: 1h

# Security Best Practices

  - Always implement proper authentication and authorization
  - Use RestPermission methods for fine-grained access control
  - Validate all input data using validation tags
  - Sanitize user input in hooks to prevent XSS and SQL injection
  - Use HTTPS in production environments
  - Implement rate limiting for API endpoints
  - Log security events and monitor for suspicious activity

# Migration and Versioning

When updating Restify versions:

  - Review breaking changes in release notes
  - Test all endpoints after updates
  - Update model validation tags if needed
  - Check for deprecated methods and replace them
  - Verify permission logic still works as expected
  - Update API documentation and client SDKs

# Troubleshooting

Common issues and solutions:

  - Database not enabled: Ensure database is configured in config.yml
  - Permission denied: Check RestPermission methods and global permission handler
  - Validation errors: Verify validation tags and custom validation logic
  - Query performance: Add database indexes and use query debugging
  - Memory issues: Implement proper pagination and avoid loading large datasets

# Examples

See the example/ directory for complete working examples including:
  - Basic CRUD operations
  - Advanced filtering and search
  - Custom validation and hooks
  - Permission implementation
  - Postman collection usage

For more detailed documentation, see the docs/ directory.
*/
package restify
