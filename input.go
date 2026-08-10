package restify

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/getevo/evo/v2/lib/generic"
)

// Input carries the parameters of a call that did not arrive over HTTP.
//
// Every handler used to read its parameters straight off the HTTP request:
// query values, route parameters and the body. That works for a REST endpoint
// and for nothing else. Input is the second source those reads fall back to,
// so one handler can serve a REST route and an MCP tool call without knowing
// which one it is answering.
//
// A nil Input means the call came over HTTP and the request is authoritative.
type Input struct {
	// Values holds named query parameters — page, size, order, fields,
	// associations, join, group_by, offset, limit, unsafe, return.
	Values map[string]string

	// Filters holds raw filter expressions in the query grammar,
	// for example "status[eq]=paid". They are the only thing filterMapper
	// reads, so they are kept apart from Values and never url-encoded as a
	// whole: the grammar itself uses square brackets.
	Filters []string

	// Params holds route parameters, keyed the way the router names them.
	// Primary keys are keyed by column name, matching the ":id" style
	// placeholders SetAction generates.
	Params map[string]string

	// Body is the raw JSON body the write handlers decode.
	Body []byte

	// Verb is the HTTP verb the call is equivalent to. Update branches on it
	// to tell a partial update from a full one.
	Verb string
}

// Set records a named query parameter, skipping empty values so that an
// absent parameter stays absent rather than becoming an empty string.
func (in *Input) Set(key, value string) {
	if value == "" {
		return
	}
	if in.Values == nil {
		in.Values = map[string]string{}
	}
	in.Values[key] = value
}

// AddFilter records one filter expression in the query grammar. The value is
// percent-encoded because the grammar accepts a restricted character set and
// filterMapper unescapes what it matches.
func (in *Input) AddFilter(column, condition, value string) {
	if condition == IsNullOperator || condition == NotNullOperator {
		in.Filters = append(in.Filters, fmt.Sprintf("%s[%s]", column, condition))
		return
	}
	in.Filters = append(in.Filters, fmt.Sprintf("%s[%s]=%s", column, condition, escapeFilterValue(value)))
}

// escapeFilterValue percent-encodes a filter value so that it survives
// filterRegEx, whose value group accepts only letters, digits, whitespace and
// a handful of punctuation marks. Percent is one of the accepted characters,
// which is what makes the encoding round trip: filterMapper unescapes the
// value before using it.
func escapeFilterValue(value string) string {
	escaped := url.QueryEscape(value)
	// QueryEscape writes a space as "+", which the grammar does not accept,
	// and leaves "~" alone, which it also does not accept.
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "~", "%7E")
	return escaped
}

// Query returns a named query parameter, from Input when the call did not
// arrive over HTTP.
func (context *Context) Query(key string) generic.Value {
	if context.Input != nil {
		return generic.Parse(context.Input.Values[key])
	}
	return context.Request.Query(key)
}

// QueryString returns the string filters are parsed from. For a non-HTTP call
// that is the filter expressions alone: the other parameters are read by name
// through Query and would only risk a false match against the filter grammar.
func (context *Context) QueryString() string {
	if context.Input != nil {
		return strings.Join(context.Input.Filters, "&")
	}
	return context.Request.QueryString()
}

// Param returns a route parameter, from Input when the call did not arrive
// over HTTP.
func (context *Context) Param(key string) generic.Value {
	if context.Input != nil {
		return generic.Parse(context.Input.Params[key])
	}
	return context.Request.Param(key)
}

// ParseBody decodes the request body into dst. A non-HTTP call carries JSON
// only, so the content type negotiation of BodyParser does not apply.
func (context *Context) ParseBody(dst any) error {
	if context.Input != nil {
		if len(context.Input.Body) == 0 {
			return fmt.Errorf("empty body")
		}
		return json.Unmarshal(context.Input.Body, dst)
	}
	return context.Request.BodyParser(dst)
}

// Method returns the HTTP verb of the call, or the verb a non-HTTP call is
// equivalent to.
func (context *Context) Method() string {
	if context.Input != nil {
		return context.Input.Verb
	}
	return context.Request.Method()
}

// Path returns the path of the call, used for logging. A non-HTTP call reports
// the URI of the endpoint it is equivalent to.
func (context *Context) Path() string {
	if context.Input != nil {
		if context.Action != nil {
			return context.Action.AbsoluteURI
		}
		return ""
	}
	return context.Request.Path()
}
