package restify

import "github.com/getevo/evo/v2/lib/errors"

// ErrorObjectNotExist represents an error indicating that the object does not exist.
var ErrorObjectNotExist = errors.New("object does not exists", 404)

// ErrorColumnNotExist represents an error indicating that a column does not exist.
var ErrorColumnNotExist = errors.New("column does not exists", 500)

var ErrorPermissionDenied = errors.New("permission denied", 403)

var ErrorUnauthorized = errors.New("unauthorized", 403)

var ErrorHandlerNotFound = errors.New("handler not found", 404)
