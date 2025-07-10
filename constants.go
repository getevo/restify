package restify

// HTTP Status Codes
const (
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusInternalServerError = 500
)

// Default Values
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
	DefaultPage     = 1
)

// Error Codes
const (
	ErrorCodeValidation     = "VALIDATION_ERROR"
	ErrorCodeDatabase       = "DATABASE_ERROR"
	ErrorCodePermission     = "PERMISSION_ERROR"
	ErrorCodeAuthentication = "AUTHENTICATION_ERROR"
	ErrorCodeNotFound       = "NOT_FOUND_ERROR"
	ErrorCodeInternal       = "INTERNAL_ERROR"
	ErrorCodeBadRequest     = "BAD_REQUEST_ERROR"
	ErrorCodeUnauthorized   = "UNAUTHORIZED_ERROR"
	ErrorCodeForbidden      = "FORBIDDEN_ERROR"
)

// Common Error Messages
const (
	MessageObjectNotExist   = "object does not exist"
	MessageColumnNotExist   = "column does not exist"
	MessagePermissionDenied = "permission denied"
	MessageUnauthorized     = "unauthorized"
	MessageHandlerNotFound  = "handler not found"
	MessageUnsafeRequest    = "unsafe request"
	MessageValidationFailed = "validation failed"
	MessageDatabaseError    = "database operation failed"
	MessageInternalError    = "internal server error"
	MessageBadRequest       = "bad request"
	MessageInvalidInput     = "invalid input provided"
	MessageOperationFailed  = "operation failed"
)

// Log Levels
const (
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
	LogLevelInfo  = "INFO"
	LogLevelDebug = "DEBUG"
)
