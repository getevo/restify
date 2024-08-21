package restify

import (
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

type AuthType string

const (
	AuthTypeNone     AuthType = "none"
	AuthTypeBasic    AuthType = "basic"
	AuthTypeBearer   AuthType = "bearer"
	AuthTypeDigest   AuthType = "digest"
	AuthTypeEdgeGrid AuthType = "edgegrid"
	AuthTypeHawk     AuthType = "hawk"
	AuthTypeOAuth1   AuthType = "oauth1"
	AuthTypeOAuth2   AuthType = "oauth2"
	AuthTypeNTLM     AuthType = "ntlm"
	AuthTypeHeader   AuthType = "header"
)

var postmanRegistered = false
var postmanAuthType = AuthTypeNone
var postmanHeaders = make(map[string]string)

func EnablePostman() {
	postmanRegistered = true
}

func ModelDataFaker(schema *schema.Schema) interface{} {
	var m = make(map[string]interface{})
	for idx, _ := range schema.Fields {
		field := schema.Fields[idx]
		if field.AutoIncrement {
			continue
		}
		var clone = reflect.New(field.FieldType)
		var jsonField = field.Tag.Get("json")
		if jsonField == "-" {
			continue
		}
		fieldName := strings.Split(jsonField, ",")[0]
		if fieldName == "" {
			fieldName = field.DBName
		}
		m[fieldName] = clone.Interface()
	}
	return m
}

func SetPostmanAuthorization(_type AuthType, _value ...string) {
	if _type == AuthTypeHeader {
		postmanAuthType = "none"
		for _, item := range _value {
			postmanHeaders[item] = ""
		}
		return
	}
	postmanAuthType = _type
}
