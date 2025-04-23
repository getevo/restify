package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/db/schema"
	"reflect"
)

type Entity struct {
	Schema  *schema.Model
	Context *Context
}

func (e *Entity) Preload(s ...string) *Entity {
	for _, preload := range s {
		e.Context.DBO = e.Context.DBO.Preload(preload)
	}
	return e
}

func (e *Entity) InnerJoins(query string, args ...interface{}) *Entity {
	e.Context.DBO = e.Context.DBO.InnerJoins(query, args...)
	return e
}

func (e *Entity) Joins(query string, args ...interface{}) *Entity {
	e.Context.DBO = e.Context.DBO.Joins(query, args...)
	return e
}

func (e *Entity) Order(value interface{}) *Entity {
	e.Context.DBO = e.Context.DBO.Order(value)
	return e
}

func (e *Entity) Where(query interface{}, args ...interface{}) *Entity {
	e.Context.DBO = e.Context.DBO.Where(query, args)
	return e
}

func (e *Entity) Limit(limit int) *Entity {
	e.Context.DBO = e.Context.DBO.Limit(limit)
	return e
}

func (e *Entity) Offset(offset int) *Entity {
	e.Context.DBO = e.Context.DBO.Offset(offset)
	return e
}

func (e *Entity) Group(name string) *Entity {
	e.Context.DBO = e.Context.DBO.Group(name)
	return e
}

func (e *Entity) Having(query interface{}, args ...interface{}) *Entity {
	e.Context.DBO = e.Context.DBO.Having(query, args...)
	return e
}

func (e *Entity) Debug() *Entity {
	e.Context.DBO = e.Context.DBO.Debug()
	return e
}

func NewEntity(obj interface{}, request *evo.Request) (*Entity, error) {
	ref := reflect.ValueOf(obj)
	for ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	if ref.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid entity type %T", obj)
	}

	find := schema.Find(obj)
	if find == nil {
		return nil, fmt.Errorf("invalid entity %v", obj)
	}
	var entity = &Entity{Schema: find}
	entity.Context = &Context{
		Action:  nil,
		Request: request,
		Object:  find.Value,
		Response: &Pagination{
			TotalPages: 1,
			Total:      1,
			Page:       1,
			Size:       1,
			Success:    true,
		},
	}
	entity.Context.DBO = db.GetContext(entity.Context, entity.Context.Request)
	entity.Context.DBO = entity.Context.DBO.Model(entity.Context.Object)

	return entity, nil
}

func (e *Entity) Load(v interface{}) error {
	var httpErr *Error
	e.Context.DBO, httpErr = e.Context.ApplyFilters(e.Context.DBO)
	if httpErr != nil {
		return fmt.Errorf(httpErr.Message)
	}
	err := e.Context.DBO.Find(v).Error
	if err != nil {
		return err
	}
	ref := reflect.ValueOf(v)
	for ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	if ref.Kind() == reflect.Struct {
		e.Context.Response.Total = 1
		if httpError := callAfterGetHook(ref.Addr().Interface(), e.Context); httpError != nil {
			return fmt.Errorf(httpError.Message)
		}
	}
	if ref.Kind() == reflect.Slice {
		e.Context.Response.Total = int64(ref.Len())
		e.Context.Response.Size = ref.Len()
		for i := 0; i < ref.Len(); i++ {
			if httpError := callAfterGetHook(ref.Index(i).Addr().Interface(), e.Context); httpError != nil {
				return fmt.Errorf(httpError.Message)
			}
		}
	}
	e.Context.Response.Success = true
	return nil
}
