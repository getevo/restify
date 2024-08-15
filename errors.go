package restify

type Error struct {
	Code    int
	Message string
}

func NewError(message string, code int) Error {
	return Error{code, message}
}

// ErrorObjectNotExist represents an error indicating that the object does not exist.
var ErrorObjectNotExist = NewError("object does not exists", 404)

// ErrorColumnNotExist represents an error indicating that a column does not exist.
var ErrorColumnNotExist = NewError("column does not exists", 500)

var ErrorPermissionDenied = NewError("permission denied", 403)

var ErrorUnauthorized = NewError("unauthorized", 403)

var ErrorHandlerNotFound = NewError("handler not found", 404)

var ErrorUnsafe = NewError("unsafe request", 400)
