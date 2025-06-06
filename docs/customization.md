# Customization
## Model Hooks

Using `model hooks`, you can modify the data of certain model before or after database actions. It is also possible to validate or interrupt the action before it runs. Below is a list of available hooks and their descriptions:

| **Hook Name**     | **Description**                                | **Supported Endpoints**                                  |
|-------------------|------------------------------------------------|----------------------------------------------------------|
| `OnBeforeCreate`  | Runs before creating an object                 | Create/Batch Create/Set                                  |
| `OnBeforeUpdate`  | Runs before updating an object                 | Update                                                   |
| `OnBeforeSave`    | Runs before create/edit object                 | Create/Update/Set/Batch Create/Batch Update              |
| `OnBeforeDelete`  | Runs before deleting an object                 | Delete/Set                                               |
| `ValidateCreate`  | Validates before creating an object            | Create/Batch Create                                      |
| `ValidateUpdate`  | Validates before updating an object            | Update/Batch Update                                      |
| `OnAfterCreate`   | Runs after creating an object                  | Create/Set                                               |
| `OnAfterUpdate`   | Runs after updating an object                  | Update                                                   |
| `OnAfterSave`     | Runs after create/edit object                  | Create/Update/Set/Batch Create/Batch Update              |
| `OnAfterDelete`   | Runs after deleting an object                  | Delete/Set                                               |
| `OnAfterGet`      | Runs after loading an object from the database | All/Paginate/Update/Set/Create/Batch Create/Batch Update |
| `RestPermissions` | Check if request is eligible to be processed   | Every endpoint                                           |

### Warning

Manipulation of values in `OnBeforeCreate`, `ValidateCreate` and `OnBeforeSave` could cause the `Set` endpoint to break, as Restify may fail to compare user input with the new values set by these hooks.

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


## Global Hooks

With `global hooks`, you have the ability to modify data before or after any database actions across all models. These hooks are invoked for every model, allowing you to apply consistent logic throughout your application. Additionally, you can validate the data or even interrupt the action before it is executed. Below is a list of the available global hooks and their descriptions:

| **Hook Name**     | **Description**                                | **Supported Endpoints**                                  |
|-------------------|------------------------------------------------|----------------------------------------------------------|
| `OnBeforeCreate`  | Runs before creating an object                 | Create/Batch Create/Set                                  |
| `OnBeforeUpdate`  | Runs before updating an object                 | Update                                                   |
| `OnBeforeSave`    | Runs before create/edit object                 | Create/Update/Set/Batch Create/Batch Update              |
| `OnBeforeDelete`  | Runs before deleting an object                 | Delete/Set                                               |
| `OnAfterCreate`   | Runs after creating an object                  | Create/Set                                               |
| `OnAfterUpdate`   | Runs after updating an object                  | Update                                                   |
| `OnAfterSave`     | Runs after create/edit object                  | Create/Update/Set/Batch Create/Batch Update              |
| `OnAfterDelete`   | Runs after deleting an object                  | Delete/Set                                               |
| `OnAfterGet`      | Runs after loading an object from the database | All/Paginate/Update/Set/Create/Batch Create/Batch Update |

### Warning

Manipulation of values in `OnBeforeCreate`, `ValidateCreate` and `OnBeforeSave` could cause the `Set` endpoint to break, as Restify may fail to compare user input with the new values set by these hooks.

##### Example

Here’s an example of how to use global hooks in your model:

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
        
        fmt.Println("Global OnAfterSave Hook Called!")
        return nil
    })
	
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

| **Feature**                | **Description**                           |
|----------------------------|-------------------------------------------|
| `restify.API`              | Enable Restify Endpoint API               |
| `restify.DisableCreate`    | Disable create and batch create endpoints |
| `restify.DisableUpdate`    | Disable update and batch update endpoints |
| `restify.DisableSet`       | Disable set endpoint                      |
| `restify.DisableList`      | Disable data listing API endpoints        |
| `restify.DisableDelete`    | Disable single and batch delete endpoints |
| `restify.DisableAggregate` | Disable aggregate endpoint                |
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