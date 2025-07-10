# Error Handling in Restify

This document provides a comprehensive guide to error handling in Restify, explaining how errors are structured, processed, and returned to API consumers.

## Overview

Restify implements a sophisticated error handling system that provides:
- Structured error responses with consistent formatting
- Multiple error types for different scenarios
- Comprehensive error logging and monitoring
- Automatic error conversion and processing
- Detailed error context for debugging

## Error Architecture

### Core Error Structure

All errors in Restify are based on the `Error` struct:

```go
type Error struct {
    Code      int                    `json:"code"`        // HTTP status code
    Message   string                 `json:"message"`     // Human-readable error message
    ErrorCode string                 `json:"error_code"`  // Machine-readable error code
    Details   map[string]interface{} `json:"details,omitempty"` // Additional error details
    Timestamp time.Time              `json:"timestamp"`   // When the error occurred
    TraceID   string                 `json:"trace_id,omitempty"` // Trace ID for debugging
    Cause     error                  `json:"-"`           // Underlying Go error (not serialized)
}
```

### Error Types

Restify defines several specialized error types:

#### 1. DatabaseError
Used for database operation failures:
```go
type DatabaseError struct {
    *Error
    Query     string `json:"query,omitempty"`     // SQL query that failed
    Operation string `json:"operation,omitempty"` // Database operation type
}
```

#### 2. PermissionError
Used for authorization failures:
```go
type PermissionError struct {
    *Error
    Resource string `json:"resource,omitempty"` // Resource being accessed
    Action   string `json:"action,omitempty"`   // Action being performed
    UserID   string `json:"user_id,omitempty"`  // User attempting the action
}
```

#### 3. AuthenticationError
Used for authentication failures:
```go
type AuthenticationError struct {
    *Error
    Reason string `json:"reason,omitempty"` // Specific authentication failure reason
}
```

#### 4. ValidationError
Used for field-level validation failures:
```go
type ValidationError struct {
    Field string      `json:"field"`           // Field name that failed validation
    Error string      `json:"error"`           // Validation error message
    Value interface{} `json:"value,omitempty"` // Value that failed validation
    Rule  string      `json:"rule,omitempty"`  // Validation rule that was violated
}
```

## Error Flow

### 1. Handler Level
All REST endpoint handlers return `*Error` types:

```go
func (Handler) Create(context *Context) *Error {
    // Parse input
    if err := context.Request.BodyParser(ptr); err != nil {
        return context.Error(err, StatusBadRequest)
    }
    
    // Check permissions
    if !context.RestPermission(PermissionCreate, object) {
        return ErrorPermissionDenied
    }
    
    // Execute hooks
    if httpError := callBeforeCreateHook(ptr, context); httpError != nil {
        return httpError
    }
    
    // Database operations
    if err := dbo.Create(ptr).Error; err != nil {
        return context.Error(err, StatusInternalServerError)
    }
    
    return nil // Success
}
```

### 2. Request Processing
The main request handler processes errors:

```go
func (action *Endpoint) handler(request *evo.Request) interface{} {
    context := &Context{...}
    
    // Execute handler and process any returned error
    context.HandleError(action.Handler(context))
    
    // Set HTTP status code
    if context.Code == 0 {
        request.Status(200)
    } else {
        request.Status(context.Code)
    }
    
    return request.JSON(response)
}
```

### 3. Error Processing
The `HandleError` method processes structured errors:

```go
func (context *Context) HandleError(error *Error) {
    if error != nil {
        context.Response.Error = error.Message
        context.Response.Success = false
        context.Code = error.Code
        
        // Log error with context
        LogError(error, LogLevelError, map[string]interface{}{
            "http_code":  error.Code,
            "error_code": error.ErrorCode,
            "trace_id":   error.TraceID,
            "endpoint":   context.Request.Path(),
            "method":     context.Request.Method(),
        })
    }
}
```

## Error Creation

### Creating Structured Errors

```go
// Basic error
err := NewError("Something went wrong", 500)

// Structured error with error code
err := NewStructuredError("Invalid input", 400, "VALIDATION_ERROR")

// Wrap existing error
err := WrapError(originalErr, "Database operation failed", 500, "DATABASE_ERROR")

// Specialized errors
dbErr := NewDatabaseError("Query failed", "SELECT", originalErr)
permErr := NewPermissionError("Access denied", "users", "create")
authErr := NewAuthenticationError("Invalid token", "token_expired")
```

### Converting Go Errors

The `Context.Error` method converts standard Go errors:

```go
func (context *Context) Error(err error, code int) *Error {
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
```

## Lifecycle Hook Errors

Lifecycle hooks can return regular Go errors, which are automatically converted:

```go
// In model hooks
func (u *User) OnBeforeCreate(context *restify.Context) error {
    if u.Age < 18 {
        return fmt.Errorf("age must be at least 18")
    }
    return nil
}

// Internal conversion
func callBeforeCreateHook(obj any, c *Context) *Error {
    err := callHook(obj, c, _onBeforeCreateCallbacks)
    if err != nil {
        return c.Error(err, 500) // Convert to structured error
    }
    return nil
}
```

## Validation Errors

Validation errors are handled separately:

```go
func (context *Context) AddValidationErrors(errs ...error) {
    if len(errs) > 0 {
        context.Response.Success = false
        context.Code = 412 // Precondition Failed
        
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
```

## Error Response Formats

### Single Error Response
```json
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
```

### Validation Error Response
```json
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
```

### Database Error Response
```json
{
    "success": false,
    "error": "Database operation failed",
    "code": 500,
    "error_code": "DATABASE_ERROR",
    "timestamp": "2023-01-01T12:00:00Z",
    "query": "SELECT * FROM users WHERE id = ?",
    "operation": "SELECT"
}
```

### Permission Error Response
```json
{
    "success": false,
    "error": "Access denied",
    "code": 403,
    "error_code": "PERMISSION_ERROR",
    "timestamp": "2023-01-01T12:00:00Z",
    "resource": "users",
    "action": "create",
    "user_id": "123"
}
```

## Error Logging

All errors are automatically logged with comprehensive context:

```go
func LogError(err error, level string, context map[string]interface{}) {
    // Get caller information
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
    case LogLevelInfo:
        log.Infof("Info: %v, Context: %+v", err, context)
    case LogLevelDebug:
        log.Debugf("Debug: %v, Context: %+v", err, context)
    }
}
```

## Panic Recovery

Restify includes panic recovery:

```go
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
```

## Predefined Errors

Restify provides commonly used predefined errors:

```go
var ErrorObjectNotExist = NewStructuredError(MessageObjectNotExist, StatusNotFound, ErrorCodeNotFound)
var ErrorColumnNotExist = NewStructuredError(MessageColumnNotExist, StatusInternalServerError, ErrorCodeInternal)
var ErrorPermissionDenied = NewStructuredError(MessagePermissionDenied, StatusForbidden, ErrorCodePermission)
var ErrorUnauthorized = NewStructuredError(MessageUnauthorized, StatusUnauthorized, ErrorCodeAuthentication)
var ErrorHandlerNotFound = NewStructuredError(MessageHandlerNotFound, StatusNotFound, ErrorCodeNotFound)
var ErrorUnsafe = NewStructuredError(MessageUnsafeRequest, StatusBadRequest, ErrorCodeBadRequest)
```

## Error Codes

Restify uses standardized error codes:

```go
const (
    ErrorCodeValidation      = "VALIDATION_ERROR"
    ErrorCodeDatabase        = "DATABASE_ERROR"
    ErrorCodePermission      = "PERMISSION_ERROR"
    ErrorCodeAuthentication  = "AUTHENTICATION_ERROR"
    ErrorCodeNotFound        = "NOT_FOUND_ERROR"
    ErrorCodeInternal        = "INTERNAL_ERROR"
    ErrorCodeBadRequest      = "BAD_REQUEST_ERROR"
    ErrorCodeUnauthorized    = "UNAUTHORIZED_ERROR"
    ErrorCodeForbidden       = "FORBIDDEN_ERROR"
)
```

## Best Practices

### 1. Use Structured Errors
Always return `*Error` types from handlers:

```go
func (Handler) CustomHandler(context *Context) *Error {
    if someCondition {
        return NewStructuredError("Custom error", 400, "CUSTOM_ERROR")
    }
    return nil
}
```

### 2. Provide Context
Include relevant details in error messages:

```go
err := NewStructuredError("User not found", 404, "NOT_FOUND_ERROR")
err.WithDetails(map[string]interface{}{
    "user_id": userID,
    "resource": "users",
})
```

### 3. Use Appropriate HTTP Codes
Match HTTP status codes to error types:

```go
// 400 for client errors
return context.Error(err, StatusBadRequest)

// 401 for authentication errors
return ErrorUnauthorized

// 403 for permission errors
return ErrorPermissionDenied

// 404 for not found errors
return ErrorObjectNotExist

// 500 for server errors
return context.Error(err, StatusInternalServerError)
```

### 4. Handle Validation Separately
Use `AddValidationErrors` for field-level validation:

```go
if validationErrors := validateInput(input); len(validationErrors) > 0 {
    context.AddValidationErrors(validationErrors...)
    return fmt.Errorf("validation failed")
}
```

### 5. Implement Custom Errors
Create domain-specific error types:

```go
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
```

### 6. Test Error Scenarios
Ensure error handling works correctly:

```go
func TestCreateUserWithInvalidData(t *testing.T) {
    // Test validation errors
    // Test permission errors
    // Test database errors
    // Test custom business logic errors
}
```

## Common Error Scenarios

### 1. Input Validation Errors
```go
func (u *User) OnBeforeCreate(context *restify.Context) error {
    if u.Email == "" {
        return fmt.Errorf("email is required")
    }
    if u.Age < 18 {
        return fmt.Errorf("age must be at least 18")
    }
    return nil
}
```

### 2. Permission Errors
```go
func (u *User) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    if permissions.Has("create") {
        return context.User.IsAdmin()
    }
    return false
}
```

### 3. Database Errors
```go
if err := context.GetDBO().Create(&user).Error; err != nil {
    if isDuplicateKeyError(err) {
        return NewStructuredError("Email already exists", 409, "DUPLICATE_ERROR")
    }
    return context.Error(err, StatusInternalServerError)
}
```

### 4. Business Logic Errors
```go
func (o *Order) OnBeforeCreate(context *restify.Context) error {
    if o.Total < 0 {
        return fmt.Errorf("order total cannot be negative")
    }
    if o.Items == nil || len(o.Items) == 0 {
        return fmt.Errorf("order must have at least one item")
    }
    return nil
}
```

## Debugging Errors

### 1. Enable Debug Mode
Add `?debug=restify` to see SQL queries:
```
GET /api/v1/users?debug=restify
```

### 2. Use Trace IDs
Add trace IDs to errors for tracking:
```go
err := NewStructuredError("Database error", 500, "DATABASE_ERROR")
err.WithTraceID(generateTraceID())
```

### 3. Log Error Context
Include relevant context in error logs:
```go
LogError(err, LogLevelError, map[string]interface{}{
    "user_id": context.User.ID,
    "endpoint": context.Request.Path(),
    "method": context.Request.Method(),
    "request_id": context.Request.Header("X-Request-ID"),
})
```

## Error Monitoring

### 1. Error Metrics
Monitor error rates and types:
- Total error count
- Error rate by endpoint
- Error distribution by type
- Response time for error scenarios

### 2. Alerting
Set up alerts for:
- High error rates
- Specific error types (database errors, permission errors)
- Unusual error patterns
- Critical system errors

### 3. Error Analysis
Regularly analyze:
- Most common error types
- Error trends over time
- User-facing vs system errors
- Error resolution times

This comprehensive error handling system ensures that Restify applications provide consistent, informative error responses while maintaining detailed logging for debugging and monitoring purposes.