package restify

import (
	"github.com/getevo/evo/v2/lib/db"
	"path/filepath"
	"reflect"
	"strings"
)

func SetPrefix(prefix string) {
	Prefix = prefix
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
	}
	if !features.DisableUpdate {
		resource.SetAction(&Endpoint{
			Name:        "UPDATE",
			Method:      MethodPOST,
			URL:         "/",
			PKUrl:       true,
			Handler:     handler.Update,
			Description: "update single object select using primary key",
		})
	}
	if !features.DisableDelete {
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
