## Permissions

In Restify, you can control access to your API endpoints using the RestPermission hook. This hook is model-specific and has higher priority than the Default Permission Handler. If the RestPermission hook is defined for a model, the default permission handler will not be executed.
### Default Permission Handler

A simple way to apply general limits on all Restify endpoints is through the `Default Permission Handler`. This function intercepts every single call to the endpoints before they execute, allowing you to block or modify requests at this stage.

```golang
func handler(permissions Permissions, context *Context) bool {
    // Example: Block all actions for anonymous users
    if context.Request.User.Anonymous {
        return false
    }
    return true
}

func (App) Register() {
    restify.SetDefaultPermissionHandler(handler)
}
```
___

### Model Rest Permission Handler
Another way to control Restify endpoints is through the `RestPermission` hook. This hook has a higher priority than the `Default Permission Handler`. If `RestPermission` is defined for a model, the default permission handler will not be executed. This allows you to implement model-specific permission logic that can override general permissions set at a higher level.

```golang
type Article struct {
	ArticleID int    `gorm:"column:article_id;primaryKey;autoIncrement" json:"article_id"`
	UserID    int    `gorm:"column:user_id;fk:user" json:"user_id"`
	Body      string `gorm:"column:body;type:text"  json:"body"`
	model.CreatedAt
	model.UpdatedAt
	model.DeletedAt
	restify.API
}

func (*Article) TableName() string {
	return "article"
}

func (article *Article) RestPermission(permissions restify.Permissions, context *restify.Context) bool {

	var user, err = GetUser(context.Request) // retrieve user from basic auth

	//Dont let user do anything if the user is not logged in
	if err != nil {
		context.Error(err, http.StatusUnauthorized)
		return false
	}

	// enable delete only for admin users
	if !user.IsAdmin && permissions.Has("DELETE") {
		return false
	}

	// automatically set user_id in context for VIEW, UPDATE, DELETE, BATCH operations to the current
	if permissions.Has("VIEW", "UPDATE", "DELETE", "BATCH") {
		context.SetCondition("user_id", "=", user.UserID)
	}

	// override user_id in context for CREATE, UPDATE, DELETE,SET operations to the current user only
	if permissions.Has("CREATE", "UPDATE", "DELETE", "SET") {
		context.Override(Article{
			UserID: user.UserID,
		})
	}
	return true
}
```

### Explanation

**User Authentication:** The `RestPermission` function first checks if the user is authenticated by retrieving the user from the request using `GetUser`. If the user is not authenticated, the function returns an unauthorized error and blocks the action.

**Delete Permission:** The function checks if the user has permission to delete records. If the user is not an admin and the requested action is a delete (`permissions.Has("DELETE")`), the delete action is blocked by returning `false`.

**Setting Conditions:** For `VIEW`, `UPDATE`, `DELETE`, and `BATCH` operations, the function automatically sets the `user_id` in the context to match the current user's ID. This ensures that the action is restricted to data belonging to the current user.

**Overriding Data:** For `CREATE`, `UPDATE`, `DELETE`, and `SET` operations, the function overrides the `user_id` field in the context to the current user's ID. This ensures that any changes or new records are correctly attributed to the current user.

### Using `permissions.Has`

The `permissions.Has` method is used to check whether the current action being performed on the model matches a specific permission. This is useful for restricting or allowing access to specific actions based on the user's role or other conditions.

**Common Permission Checks:**

- `permissions.Has("VIEW")`: Checks if the user has permission to view records.
- `permissions.Has("UPDATE")`: Checks if the user has permission to update records.
- `permissions.Has("DELETE")`: Checks if the user has permission to delete records.
- `permissions.Has("BATCH")`: Checks if the user has permission to perform batch operations.
- `permissions.Has("CREATE")`: Checks if the user has permission to create new records.
- `permissions.Has("SET")`: Checks if the user has permission to use the set operation.

By using these checks, you can enforce fine-grained access control within your Restify application, ensuring that users can only perform actions that they are authorized to do.


