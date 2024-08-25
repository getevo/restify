package restify

var _onBeforeCreateCallbacks []func(obj any, c *Context) error
var _onBeforeUpdateCallbacks []func(obj any, c *Context) error
var _onBeforeSaveCallbacks []func(obj any, c *Context) error
var _onBeforeDeleteCallbacks []func(obj any, c *Context) error
var _onAfterCreateCallbacks []func(obj any, c *Context) error
var _onAfterUpdateCallbacks []func(obj any, c *Context) error
var _onAfterSaveCallbacks []func(obj any, c *Context) error
var _onAfterDeleteCallbacks []func(obj any, c *Context) error
var _onAfterGetCallbacks []func(obj any, c *Context) error

func (app App) registerHooks() {
	OnBeforeCreate(func(obj any, context *Context) error {
		if obj, ok := obj.(interface{ OnBeforeCreate(context *Context) error }); ok {
			err := obj.OnBeforeCreate(context)
			if err != nil {
				return err
			}
		}

		if v, ok := obj.(interface{ ValidateCreate(context *Context) error }); ok {
			if err := v.ValidateCreate(context); err != nil {
				return err
			}
		}

		if err := context.Validate(obj); err != nil {
			return err
		}

		return nil
	})

	OnBeforeSave(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnBeforeSave(context *Context) error }); ok {
			err := v.OnBeforeSave(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnBeforeUpdate(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnBeforeUpdate(context *Context) error }); ok {
			err := v.OnBeforeUpdate(context)
			if err != nil {
				return err
			}
		}

		if v, ok := obj.(interface{ ValidateUpdate(context *Context) error }); ok {
			if err := v.ValidateUpdate(context); err != nil {
				return err
			}
		}

		if err := context.ValidateNonZeroFields(obj); err != nil {
			return err
		}

		return nil
	})

	OnBeforeDelete(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnBeforeDelete(context *Context) error }); ok {
			err := v.OnBeforeDelete(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnAfterCreate(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnAfterCreate(context *Context) error }); ok {
			err := v.OnAfterCreate(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnAfterSave(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnAfterSave(context *Context) error }); ok {
			err := v.OnAfterSave(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnAfterUpdate(func(obj any, context *Context) error {
		if v, ok := obj.(interface{ OnAfterUpdate(context *Context) error }); ok {
			err := v.OnAfterUpdate(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnAfterDelete(func(obj any, context *Context) error {
		if obj, ok := obj.(interface{ OnAfterDelete(context *Context) error }); ok {
			err := obj.OnAfterDelete(context)
			if err != nil {
				return err
			}
		}
		return nil
	})

	OnAfterGet(func(obj any, context *Context) error {
		if obj, ok := obj.(interface{ OnAfterGet(context *Context) error }); ok {
			err := obj.OnAfterGet(context)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func OnBeforeCreate(fn func(obj any, c *Context) error) {
	_onBeforeCreateCallbacks = append(_onBeforeCreateCallbacks, fn)
}

func OnBeforeUpdate(fn func(obj any, c *Context) error) {
	_onBeforeUpdateCallbacks = append(_onBeforeUpdateCallbacks, fn)
}

func OnBeforeSave(fn func(obj any, c *Context) error) {
	_onBeforeSaveCallbacks = append(_onBeforeSaveCallbacks, fn)
}

func OnBeforeDelete(fn func(obj any, c *Context) error) {
	_onBeforeDeleteCallbacks = append(_onBeforeDeleteCallbacks, fn)
}

func OnAfterCreate(fn func(obj any, c *Context) error) {
	_onAfterCreateCallbacks = append(_onAfterCreateCallbacks, fn)
}

func OnAfterUpdate(fn func(obj any, c *Context) error) {
	_onAfterUpdateCallbacks = append(_onAfterUpdateCallbacks, fn)
}

func OnAfterSave(fn func(obj any, c *Context) error) {
	_onAfterSaveCallbacks = append(_onAfterSaveCallbacks, fn)
}

func OnAfterDelete(fn func(obj any, c *Context) error) {
	_onAfterDeleteCallbacks = append(_onAfterDeleteCallbacks, fn)
}

func OnAfterGet(fn func(obj any, c *Context) error) {
	_onAfterGetCallbacks = append(_onAfterGetCallbacks, fn)
}

func callHook(obj any, c *Context, callbackList []func(obj any, c *Context) error) error {
	for _, fn := range callbackList {
		if err := fn(obj, c); err != nil {
			return err
		}
	}
	return nil
}

func callBeforeCreateHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onBeforeCreateCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	err = callHook(obj, c, _onBeforeSaveCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}

	return nil
}

func callBeforeUpdateHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onBeforeUpdateCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	err = callHook(obj, c, _onBeforeSaveCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}

	return nil
}

func callBeforeDeleteHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onBeforeDeleteCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	return nil
}

func callAfterCreateHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onAfterCreateCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	err = callHook(obj, c, _onAfterSaveCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	return nil
}

func callAfterUpdateHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onAfterUpdateCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	err = callHook(obj, c, _onAfterSaveCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	return nil
}

func callAfterDeleteHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onAfterDeleteCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	return nil
}

func callAfterGetHook(obj any, c *Context) *Error {
	err := callHook(obj, c, _onAfterGetCallbacks)
	if err != nil {
		return c.Error(err, 500)
	}
	return nil
}
