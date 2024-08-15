# Customization
## Hooks

Using hooks, you can modify the data before or after database actions. It is also possible to validate or interrupt the action before it runs. Below is a list of available hooks and their descriptions:

| **Hook Name**     | **Description**                                | **Supported Endpoints** |
|-------------------|------------------------------------------------|-------------------------|
| `OnBeforeCreate`  | Runs before creating an object                 | Create/Batch Create/Set |
| `OnBeforeUpdate`  | Runs before updating an object                 | Update                  |
| `OnBeforeDelete`  | Runs before deleting an object                 | Delete/Set              |
| `ValidateCreate`  | Validates before creating an object            | Create/Batch Create     |
| `ValidateUpdate`  | Validates before updating an object            | Update/Batch Update     |
| `OnAfterCreate`   | Runs after creating an object                  | Create/Set              |
| `OnAfterUpdate`   | Runs after updating an object                  | Update                  |
| `OnAfterDelete`   | Runs after deleting an object                  | Delete/Set              |
| `OnAfterGet`      | Runs after loading an object from the database | All/Paginate/Update/Set |
| `RestPermissions` | Check if request is eligible to be processed   | Every endpoint          |

### Warning

Manipulation of values in `OnBeforeCreate` and `ValidateCreate` could cause the `set` endpoint to break, as Restify may fail to compare user input with the new values set by these hooks.

##### Example

Here’s an example of how to use hooks in your model:

```golang
func (user *User) OnBeforeCreate(context *restify.Context) error {
    user.Username = uuid.New().String()
    return nil
}

func (user *User) OnBeforeUpdate(context *restify.Context) error {
    if len(user.Name) == "" {
        return fmt.Errorf("invalid name")
    }
    return nil
}
```


---

## Validation

Using validation tags, you can automatically validate your data before submitting it to the database. This feature leverages the `"github.com/getevo/evo/v2/lib/validation"` library. For more detailed information, please refer to the [validation documentation](https://github.com/getevo/evo/blob/master/docs/validation.md).

##### Example

```golang
type User struct {
    UserID   int    `gorm:"primaryKey;autoIncrement"`
    UserName string `gorm:"column:username;index;size:255" validation:"required"`      // username will be required
    Name     string `gorm:"column:name;size:255" validation:"alpha,required"`          // name will be required and can contain only alpha characters
    restify.API     // this signature will enable Restify
}
```


---
## Features

You may turn on or off some Restify endpoints. The following table outlines the available options:

| **Feature**             | **Description**                           |
|-------------------------|-------------------------------------------|
| `restify.API`           | Enable Restify Endpoint API               |
| `restify.DisableCreate` | Disable create and batch create endpoints |
| `restify.DisableUpdate` | Disable update and batch update endpoints |
| `restify.DisableSet`    | Disable set endpoint                      |
| `restify.DisableList`   | Disable data listing API endpoints        |
| `restify.DisableDelete` | Disable single and batch delete endpoints |

##### Example

```golang
type User struct {
    UserID   int    `gorm:"primaryKey;autoIncrement"`
    UserName string `gorm:"column:username;index;size:255" validation:"required"`      // username will be required
    Name     string `gorm:"column:name;size:255" validation:"alpha,required"`          // name will be required and can contain only alpha characters
    restify.API     // this signature will enable Restify
    restify.DisableDelete // disable delete APIs
}
```

---
## Base Path

The Restify API base path is set to `"/admin/restify"` by default. However, you can modify this path before calling `WhenReady` by using the `restify.SetPrefix` method.

#### Example

```golang
func (App) Register() {
    restify.SetPrefix("/api/rest") // all Restify endpoints will start with /api/rest
}
```
---
## Soft Delete

If the represented model has a `Delete` method, Restify will call this method by default, effectively performing a soft delete. If the `Delete` method does not exist, Restify will perform a hard delete instead. However, it is highly recommended to use `model.DeletedAt` included in `"github.com/getevo/evo/v2/lib/model"` for handling soft deletes.

```golang
type User struct {
    UserID   int    `gorm:"primaryKey;autoIncrement"`
    UserName string `gorm:"column:username;index;size:255"`    
    Name     string `gorm:"column:name;size:255"`         
    IsDeleted bool
    restify.API    
}

func (user *User)Delete()  {
    user.IsDeleted = true
}
```