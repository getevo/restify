package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/log"
	"runtime"
	"time"
)

// Error represents a structured error with detailed information
type Error struct {
	Code      int                    `json:"code"`
	Message   string                 `json:"message"`
	ErrorCode string                 `json:"error_code"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Cause     error                  `json:"-"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error for error wrapping
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetails adds additional details to the error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	e.Details = details
	return e
}

// WithTraceID adds a trace ID to the error for debugging
func (e *Error) WithTraceID(traceID string) *Error {
	e.TraceID = traceID
	return e
}

// NewError creates a new structured error
func NewError(message string, code int) Error {
	return Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewStructuredError creates a new structured error with error code
func NewStructuredError(message string, httpCode int, errorCode string) *Error {
	return &Error{
		Code:      httpCode,
		Message:   message,
		ErrorCode: errorCode,
		Timestamp: time.Now(),
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, message string, httpCode int, errorCode string) *Error {
	return &Error{
		Code:      httpCode,
		Message:   message,
		ErrorCode: errorCode,
		Timestamp: time.Now(),
		Cause:     err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Field: field,
		Error: message,
	}
}

// DatabaseError represents database-specific errors
type DatabaseError struct {
	*Error
	Query     string `json:"query,omitempty"`
	Operation string `json:"operation,omitempty"`
}

// NewDatabaseError creates a new database error
func NewDatabaseError(message string, operation string, cause error) *DatabaseError {
	return &DatabaseError{
		Error:     WrapError(cause, message, StatusInternalServerError, ErrorCodeDatabase),
		Operation: operation,
	}
}

// PermissionError represents permission-specific errors
type PermissionError struct {
	*Error
	Resource string `json:"resource,omitempty"`
	Action   string `json:"action,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

// NewPermissionError creates a new permission error
func NewPermissionError(message, resource, action string) *PermissionError {
	return &PermissionError{
		Error:    NewStructuredError(message, StatusForbidden, ErrorCodePermission),
		Resource: resource,
		Action:   action,
	}
}

// AuthenticationError represents authentication-specific errors
type AuthenticationError struct {
	*Error
	Reason string `json:"reason,omitempty"`
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message, reason string) *AuthenticationError {
	return &AuthenticationError{
		Error:  NewStructuredError(message, StatusUnauthorized, ErrorCodeAuthentication),
		Reason: reason,
	}
}

// LogError logs an error with appropriate level and context
func LogError(err error, level string, context map[string]interface{}) {
	// Get caller information for better debugging
	_, file, line, ok := runtime.Caller(1)
	if ok {
		if context == nil {
			context = make(map[string]interface{})
		}
		context["file"] = file
		context["line"] = line
	}

	switch level {
	case LogLevelError:
		log.Errorf("Error: %v, Context: %+v", err, context)
	case LogLevelWarn:
		log.Warningf("Warning: %v, Context: %+v", err, context)
	case LogLevelInfo:
		log.Infof("Info: %v, Context: %+v", err, context)
	case LogLevelDebug:
		log.Debugf("Debug: %v, Context: %+v", err, context)
	default:
		log.Errorf("Error: %v, Context: %+v", err, context)
	}
}

// RecoverFromPanic recovers from panics and converts them to errors
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

// ErrorObjectNotExist Predefined errors with structured format
var ErrorObjectNotExist = NewStructuredError(MessageObjectNotExist, StatusNotFound, ErrorCodeNotFound)
var ErrorColumnNotExist = NewStructuredError(MessageColumnNotExist, StatusInternalServerError, ErrorCodeInternal)
var ErrorPermissionDenied = NewStructuredError(MessagePermissionDenied, StatusForbidden, ErrorCodePermission)
var ErrorUnauthorized = NewStructuredError(MessageUnauthorized, StatusUnauthorized, ErrorCodeAuthentication)
var ErrorHandlerNotFound = NewStructuredError(MessageHandlerNotFound, StatusNotFound, ErrorCodeNotFound)
var ErrorUnsafe = NewStructuredError(MessageUnsafeRequest, StatusBadRequest, ErrorCodeBadRequest)
