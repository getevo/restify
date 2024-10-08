## Context

Context helps developers take control over the actions of Restify endpoints by providing mechanisms to enforce conditions and override data before it is submitted to the database.

### Forced Conditions

Conditions allow you to append custom conditions to `select`, `update`, and `delete` queries, forcing specific criteria to be met.

### Override

The `Override` function lets you modify data before it is submitted to the database, ensuring that certain fields or values are set according to your application's logic.


### Error Handling

The `Error` function lets you throw an error as endpoint response.


### Example

Here’s an example of how to use `RestPermission` to manage permissions, conditions, and overrides:

```golang
func (article *Article) RestPermission(permissions restify.Permissions, context *restify.Context) bool {

    var user, err = GetUser(context.Request) // retrieve user from basic auth

    // Don't let user do anything if the user is not logged in
    if err != nil {
        context.Error(err, http.StatusUnauthorized)
        return false
    }

    // Enable delete only for admin users
    if !user.IsAdmin && permissions.Has("DELETE") {
        return false
    }

    // Automatically set user_id in context for VIEW, UPDATE, DELETE, BATCH operations to the current user
    if permissions.Has("VIEW", "UPDATE", "DELETE", "BATCH") {
        context.SetCondition("user_id", "=", user.UserID)
    }

    // Override user_id in context for CREATE, UPDATE, DELETE, SET operations to the current user only
    if permissions.Has("CREATE", "UPDATE", "DELETE", "SET") {
        context.Override(Article{
            UserID: user.UserID,
        })
    }

    return true
}
```


### Custom Validation Errors

The `AddValidationErrors` function lets you throw validation errors as endpoint response.

```golang
func (app App) Register() error {
    //global hooks
    restify.OnBeforeSave(func(obj any, c *restify.Context) error {
        if user, ok := obj.(*User); ok {
			if user.Username == "unallowed" {
                c.AddValidationErrors(fmt.Errorf("this username is not allowed"))
                return fmt.Errorf("validation error")
            }
        }
		
        return nil
    })
	
    return nil
}
```