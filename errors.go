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

// ErrorInvalidColumn is returned when order or fields name something that is not a column of the model.
var ErrorInvalidColumn = NewError("invalid column in order or fields", 400)

var ErrorPermissionDenied = NewError("permission denied", 403)

var ErrorUnauthorized = NewError("unauthorized", 403)

var ErrorHandlerNotFound = NewError("handler not found", 404)

var ErrorUnsafe = NewError("unsafe request", 400)
