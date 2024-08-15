package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/db"
	"path/filepath"
	"reflect"
	"strings"
)

func SetPrefix(prefix string) {
	Prefix = prefix
}

func SetDefaultPermissionHandler(handler func(permissions Permissions, context *Context) bool) {
	permissionHandler = handler
}

func UseModel(model any) *Resource {
	var features = GetFeatures(model)
	ref := reflect.ValueOf(model)
	for ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	model = ref.Interface()

	typ := reflect.TypeOf(model)
	stmt := db.Model(model).Statement
	_ = stmt.Parse(model)
	var resource = Resource{
		PrimaryFieldDBNames: stmt.Schema.PrimaryFieldDBNames,
		Feature:             features,
		Schema:              stmt.Schema,
		Table:               stmt.Table,
		Ref:                 ref,
		Instance:            model,
		Type:                typ,
		Name:                filepath.Base(ref.Type().PkgPath()) + "." + typ.Name(),
	}
	if !features.API {
		return &resource
	}
	var handler = Handler{}
	resource.SetAction(&Endpoint{
		Name:        "MODEL INFO",
		Method:      MethodGET,
		URL:         "/",
		Handler:     handler.ModelInfo,
		Description: "return information of the model",
	})

	if !features.DisableSet {
		resource.SetAction(&Endpoint{
			Name:        "SET",
			Method:      MethodPOST,
			URL:         "/set",
			PKUrl:       false,
			Handler:     handler.Set,
			Description: "set objects in database",
		})
	}

	if !features.DisableList {
		resource.SetAction(&Endpoint{
			Name:        "ALL",
			Method:      MethodGET,
			URL:         "/all",
			Handler:     handler.All,
			Description: "return all objects in one call",
		})

		resource.SetAction(&Endpoint{
			Name:        "PAGINATE",
			Method:      MethodGET,
			URL:         "/paginate",
			Handler:     handler.Paginate,
			Description: "paginate objects",
		})

		resource.SetAction(&Endpoint{
			Name:        "GET",
			Method:      MethodGET,
			URL:         "/",
			PKUrl:       true,
			Handler:     handler.Get,
			Description: "get single object using primary key",
		})
	}
	if !features.DisableCreate {
		resource.SetAction(&Endpoint{
			Name:        "CREATE",
			Method:      MethodPUT,
			URL:         "/",
			Handler:     handler.Create,
			Description: "create an object using given values",
		})
		resource.SetAction(&Endpoint{
			Name:        "BATCH.CREATE",
			Method:      MethodPUT,
			URL:         "/batch",
			PKUrl:       false,
			Handler:     handler.BatchCreate,
			Description: "create a batch of objects",
		})
	}
	if !features.DisableUpdate {
		resource.SetAction(&Endpoint{
			Name:        "BATCH.UPDATE",
			Method:      MethodPatch,
			URL:         "/batch",
			PKUrl:       false,
			Handler:     handler.BatchUpdate,
			Description: "update batch objects",
		})
		resource.SetAction(&Endpoint{
			Name:        "UPDATE",
			Method:      MethodPatch,
			URL:         "/",
			PKUrl:       true,
			Handler:     handler.Update,
			Description: "update single object select using primary key",
		})

	}

	if !features.DisableDelete {
		resource.SetAction(&Endpoint{
			Name:        "BATCH.DELETE",
			Method:      MethodDELETE,
			URL:         "/batch",
			PKUrl:       false,
			Handler:     handler.BatchDelete,
			Description: "batch delete objects",
		})
		resource.SetAction(&Endpoint{
			Name:        "DELETE",
			Method:      MethodDELETE,
			URL:         "/",
			PKUrl:       true,
			Handler:     handler.Delete,
			Description: "delete existing object using primary key",
		})
	}

	resources[resource.Table] = &resource

	return &resource
}

var _true = reflect.ValueOf(true)

func GetFeatures(v interface{}) Feature {
	var features = Feature{}
	var inputType = reflect.ValueOf(v)
	for inputType.Kind() == reflect.Ptr {
		inputType = inputType.Elem()
	}
	var featureValue = reflect.ValueOf(&features).Elem()
	var featureType = reflect.TypeOf(features)

	for i := 0; i < featureValue.NumField(); i++ {

		for j := 0; j < inputType.NumField(); j++ {
			t := inputType.Field(j).Type().String()
			var chunks = strings.Split(t, ".")
			if len(chunks) == 2 && chunks[1] == featureType.Field(i).Name {
				featureValue.Field(i).Set(_true)
				break
			}
		}

	}

	return features
}

func equal(val1, val2 reflect.Value) bool {

	// Ensure both values are structs or pointers to structs
	for val1.Kind() == reflect.Ptr {
		val1 = val1.Elem()
	}
	for val2.Kind() == reflect.Ptr {
		val2 = val2.Elem()
	}

	// Both values must be structs
	if val1.Kind() != reflect.Struct || val2.Kind() != reflect.Struct {
		return false
	}

	// Iterate through fields of the first struct
	for i := 0; i < val1.NumField(); i++ {
		field1 := val1.Field(i)
		field2 := val2.Field(i)
		// Skip zero fields
		if field2.IsZero() || field1.IsZero() {
			continue
		}
		// Dereference pointers if applicable
		if field1.Kind() == reflect.Ptr {
			if field1.IsNil() || field2.IsNil() {
				if field1.IsNil() != field2.IsNil() {
					return false
				}
				continue // Both are nil, consider them equal
			}
			field1 = field1.Elem()
		}
		for field2.Kind() == reflect.Ptr {
			field2 = field2.Elem()
		}
		for field1.Kind() == reflect.Ptr {
			field1 = field1.Elem()
		}
		// Skip struct fields
		if field1.Kind() == reflect.Struct {
			continue
		}

		// Compare non-struct fields
		if fmt.Sprint(field2.Interface()) != fmt.Sprint(field1.Interface()) {
			return false
		}

	}

	return true
}
