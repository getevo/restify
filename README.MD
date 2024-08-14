# Restify

Restify is an application built for the [Evo Framework](https://github.com/getevo/evo) that allows you to generate RESTful APIs on the fly without the need to write any code. These APIs can be used for various purposes, including data management, data entry processes, dashboards, or communication with third-party applications.

## Features

- **No Code Required:** Automatically generate RESTful APIs without writing a single line of code.
- **Data Management:** Use the generated APIs for managing data in your applications.
- **Data Entry:** Facilitate data entry processes through RESTful endpoints.
- **Dashboard Integration:** Seamlessly integrate with dashboards for real-time data management.
- **Third-Party Communication:** Use the APIs to interact with external services or applications.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- Evo Framework: [Installation Guide](https://github.com/getevo/evo)
- Golang (version 1.22 or higher)

### Usage

1. Import Restify into your Evo application:

    ```golang
    import (
        "github.com/getevo/evo/v2"
        "github.com/getevo/restify"
    )

    func main() {
        evo.Setup()
        evo.Register(restify.App{})
        evo.Run()
    }
    ```

2. Add Restify support to your model:

    ```golang
    type User struct {
        UserID   int    `gorm:"primaryKey;autoIncrement"`
        UserName string `gorm:"column:username;index;size:255"`
        Name     string `gorm:"column:name;size:255"`
        restify.API     // this signature enables Restify
    }
    ```

3. Register your model with Restify:

    ```golang
    func (App) Register() {
        db.UseModel(User{})
    }
    ```



### Customization

#### Hooks

Using hooks, you can modify the data before or after database actions. It is also possible to validate or interrupt the action before it runs. Below is a list of available hooks and their descriptions:

| **Hook Name**      | **Description**                                                | **Batch Support** |
|--------------------|----------------------------------------------------------------|-------------------|
| `OnBeforeCreate`   | Runs before creating an object                                  | No                |
| `OnBeforeUpdate`   | Runs before updating an object                                  | No                |
| `OnBeforeDelete`   | Runs before deleting an object                                  | No                |
| `ValidateCreate`   | Validates before creating an object                             | Yes               |
| `ValidateUpdate`   | Validates before updating an object                             | No                |
| `OnAfterCreate`    | Runs after creating an object                                   | No                |
| `OnAfterUpdate`    | Runs after updating an object                                   | No                |
| `OnAfterGet`       | Runs after loading an object from the database                  | Yes (for batch update with `return=1` option) |

#### Example

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