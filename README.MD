# Restify

Restify is an application built for the [EVO Framework](https://github.com/getevo/evo) that allows you to generate RESTful APIs on the fly without the need to write any code. These APIs can be used for various purposes, including data management, data entry processes, dashboards, or communication with third-party applications.

## Features

- **No Code Required:** Automatically generate RESTful APIs without writing a single line of code.
- **Data Management:** Use the generated APIs for managing data in your applications.
- **Data Entry:** Facilitate data entry processes through RESTful endpoints.
- **Dashboard Integration:** Seamlessly integrate with dashboards for real-time data management.
- **Third-Party Communication:** Use the APIs to interact with external services or applications.

---

## Table of contents
- **[Getting Started](https://github.com/getevo/restify?tab=readme-ov-file#getting-started)**
- **[Endpoints](./docs/endpoints.md)**
  - [Endpoints](./docs/endpoints.md#endpoints)
  - [Query Parameters Explanation](./docs/endpoints.md#query-parameters-explanation)
  - [Loading Associations](./docs/endpoints.md#loading-associations)
  - [Offset and Limit](./docs/endpoints.md#offset-and-limit)
  - [Select Specific Fields](./docs/endpoints.md#select-specific-fields)
  - [Postman Collection Generator](./docs/endpoints.md#postman-collection-generator)
  - [Example Postman Collection](./docs/endpoints.md#example-postman-collection)
- **[Customization](./docs/customization.md)**
  - [Hooks](./docs/customization.md#hooks)
  - [Validation](./docs/customization.md#validation)
  - [Features](./docs/customization.md#features)
  - [Base Path](./docs/customization.md#base-path)
  - [Soft Delete](./docs/customization.md#soft-delete)
- **[Permissions](./docs/permissions.md)**
  - [Default Permission Handler](./docs/permissions.md#default-permission-handler)
  - [Model Rest Permission Handler](./docs/permissions.md#model-rest-permission-handler)
  - [Using `permissions.Has`](./docs/permissions.md#using-permissionshas)
- **[Context](./docs/context.md)**
  - [Forced Conditions](./docs/context.md#forced-conditions)
  - [Override](./docs/context.md#override)
  - [Error Handling](./docs/context.md#error-handling)
- **[Example](./example)**

---

### Getting Started

#### Prerequisites

Before you begin, ensure you have the following installed:

- EVO Framework: [Installation Guide](https://github.com/getevo/evo)
- Golang (version 1.22 or higher)

#### Usage

1. Import Restify into your EVO application:

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




