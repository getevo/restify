# Restify

Restify is an application built for the [EVO Framework](https://github.com/getevo/evo) that allows you to generate RESTful APIs on the fly without the need to write any code. These APIs can be used for various purposes, including data management, data entry processes, dashboards, or communication with third-party applications.

## Features

- **No Code Required:** Automatically generate RESTful APIs without writing a single line of code.
- **Data Management:** Use the generated APIs for managing data in your applications.
- **Data Entry:** Facilitate data entry processes through RESTful endpoints.
- **Dashboard Integration:** Seamlessly integrate with dashboards for real-time data management.
- **Third-Party Communication:** Use the APIs to interact with external services or applications.
- **Comprehensive Error Handling:** Structured error responses with detailed information, multiple error types, automatic logging, and consistent formatting for better debugging and monitoring.

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
- **[Developers Integration Guide](./docs/developer.md)**
- **[Customization](./docs/customization.md)**
  - [Model Hooks](./docs/customization.md#model-hooks)
  - [Global Hooks](./docs/customization.md#global-hooks)
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
  - [Custom Validation Errors](./docs/context.md#custom-validation-errors)
- **[Error Handling](./docs/error-handling.md)**
  - [Error Architecture](./docs/error-handling.md#error-architecture)
  - [Error Types](./docs/error-handling.md#error-types)
  - [Error Flow](./docs/error-handling.md#error-flow)
  - [Error Creation](./docs/error-handling.md#error-creation)
  - [Error Response Formats](./docs/error-handling.md#error-response-formats)
  - [Best Practices](./docs/error-handling.md#best-practices)
- **[Advanced Features](./docs/advanced.md)**
  - [Ready Function](./docs/advanced.md#ready-function)
  - [Debug Mode](./docs/advanced.md#debug-mode)
  - [Language Support](./docs/advanced.md#language-support)
  - [Custom Database Context](./docs/advanced.md#custom-database-context)
  - [Performance Tips](./docs/advanced.md#performance-tips)
  - [Security Best Practices](./docs/advanced.md#security-best-practices)
- **[Example](./example)**

---

### Getting Started

#### Prerequisites

Before you begin, ensure you have the following installed:

- EVO Framework: [Installation Guide](https://github.com/getevo/evo)
- Golang (version 1.22 or higher)

#### Quick Start

Follow these steps to get Restify up and running in your EVO application:

##### 1. Install Dependencies

```bash
go mod init your-project
go get github.com/getevo/evo/v2
go get github.com/getevo/restify
```

##### 2. Create Your Main Application

Create a `main.go` file:

```golang
package main

import (
    "github.com/getevo/evo/v2"
    "github.com/getevo/restify"
    "your-project/models"
)

func main() {
    evo.Setup()
    evo.Register(restify.App{}, models.App{})
    evo.Run()
}
```

##### 3. Define Your Models

Create a `models/models.go` file:

```golang
package models

import (
    "github.com/getevo/evo/v2/lib/model"
    "github.com/getevo/restify"
)

type User struct {
    UserID   int    `gorm:"primaryKey;autoIncrement" json:"user_id"`
    Username string `gorm:"column:username;uniqueIndex;size:255" validation:"required" json:"username"`
    Email    string `gorm:"column:email;uniqueIndex;size:255" validation:"required,email" json:"email"`
    Name     string `gorm:"column:name;size:255" validation:"required" json:"name"`
    IsActive bool   `gorm:"column:is_active;default:true" json:"is_active"`
    model.CreatedAt
    model.UpdatedAt
    model.DeletedAt // Enables soft delete
    restify.API     // This enables Restify endpoints
}

type Product struct {
    ProductID   int     `gorm:"primaryKey;autoIncrement" json:"product_id"`
    Name        string  `gorm:"column:name;size:255" validation:"required" json:"name"`
    Description string  `gorm:"column:description;type:text" json:"description"`
    Price       float64 `gorm:"column:price" validation:"required,+float" json:"price"`
    Stock       int     `gorm:"column:stock;default:0" json:"stock"`
    CategoryID  int     `gorm:"column:category_id;fk:category" json:"category_id"`
    model.CreatedAt
    model.UpdatedAt
    restify.API
}

type Category struct {
    CategoryID int    `gorm:"primaryKey;autoIncrement" json:"category_id"`
    Name       string `gorm:"column:name;size:255;uniqueIndex" validation:"required" json:"name"`
    model.CreatedAt
    model.UpdatedAt
    restify.API
}
```

##### 4. Create Your App Registration

Create a `models/app.go` file:

```golang
package models

import (
    "github.com/getevo/evo/v2"
    "github.com/getevo/evo/v2/lib/application"
    "github.com/getevo/evo/v2/lib/db"
    "github.com/getevo/restify"
)

type App struct{}

func (app App) Register() error {
    // Configure Restify
    restify.SetPrefix("/api/v1")
    restify.EnablePostman()

    // Register models with the database
    db.UseModel(User{}, Product{}, Category{})

    // Auto-migrate database schema
    evo.GetDBO().AutoMigrate(&User{}, &Product{}, &Category{})

    return nil
}

func (app App) Router() error {
    return nil
}

func (app App) WhenReady() error {
    return nil
}

func (app App) Priority() application.Priority {
    return application.DEFAULT
}

func (app App) Name() string {
    return "models"
}
```

##### 5. Create Configuration File

Create a `config.yml` file:

```yaml
Database:
  Driver: "sqlite"
  DSN: "./app.db"

Server:
  Host: "localhost"
  Port: 8080

Debug: "1"
```

##### 6. Run Your Application

```bash
go run main.go
```

Your Restify API will be available at `http://localhost:8080/api/v1/`

##### 7. Test Your API

Once your application is running, you can test the automatically generated endpoints:

```bash
# Get all users
curl --location 'http://localhost:8080/api/v1/user/all'

# Create a new user
curl --location --request PUT 'http://localhost:8080/api/v1/user' \
--header 'Content-Type: application/json' \
--data '{
    "username": "john_doe",
    "email": "john@example.com",
    "name": "John Doe"
}'

# Get user by ID
curl --location 'http://localhost:8080/api/v1/user/1'

# Update user
curl --location --request PATCH 'http://localhost:8080/api/v1/user/1' \
--header 'Content-Type: application/json' \
--data '{
    "name": "John Smith"
}'

# Get paginated users
curl --location 'http://localhost:8080/api/v1/user/paginate?page=1&size=10'

# Filter users
curl --location 'http://localhost:8080/api/v1/user/all?name[contains]=john&is_active[eq]=true'
```

##### 8. Download Postman Collection

Get a complete Postman collection for your API:

```bash
curl -o "api-collection.json" "http://localhost:8080/api/v1/postman"
```

---

#### Advanced Usage Examples

##### Model with Hooks

```golang
type Order struct {
    OrderID    int     `gorm:"primaryKey;autoIncrement" json:"order_id"`
    UserID     int     `gorm:"column:user_id;fk:user" json:"user_id"`
    ProductID  int     `gorm:"column:product_id;fk:product" json:"product_id"`
    Quantity   int     `gorm:"column:quantity" validation:"required,+int" json:"quantity"`
    TotalPrice float64 `gorm:"column:total_price" json:"total_price"`
    Status     string  `gorm:"column:status;default:'pending'" json:"status"`
    model.CreatedAt
    model.UpdatedAt
    restify.API
}

// Calculate total price before creating order
func (order *Order) OnBeforeCreate(context *restify.Context) error {
    var product Product
    if err := context.GetDBO().First(&product, order.ProductID).Error; err != nil {
        context.AddValidationErrors(fmt.Errorf("product not found"))
        return fmt.Errorf("validation error")
    }

    order.TotalPrice = product.Price * float64(order.Quantity)
    return nil
}

// Update product stock after order creation
func (order *Order) OnAfterCreate(context *restify.Context) error {
    return context.GetDBO().Model(&Product{}).
        Where("product_id = ?", order.ProductID).
        Update("stock", gorm.Expr("stock - ?", order.Quantity)).Error
}
```

##### Model with Custom Permissions

```golang
type Invoice struct {
    InvoiceID int    `gorm:"primaryKey;autoIncrement" json:"invoice_id"`
    UserID    int    `gorm:"column:user_id;fk:user" json:"user_id"`
    Amount    float64 `gorm:"column:amount" json:"amount"`
    Status    string  `gorm:"column:status;default:'draft'" json:"status"`
    model.CreatedAt
    model.UpdatedAt
    restify.API
}

func (invoice *Invoice) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    user := getCurrentUser(context.Request)
    if user == nil {
        context.Error(fmt.Errorf("authentication required"), http.StatusUnauthorized)
        return false
    }

    // Users can only see their own invoices
    if permissions.Has("VIEW", "UPDATE", "DELETE") {
        context.SetCondition("user_id", "=", user.UserID)
    }

    // Only allow users to create invoices for themselves
    if permissions.Has("CREATE") {
        context.Override(Invoice{UserID: user.UserID})
    }

    // Only admins can delete invoices
    if permissions.Has("DELETE") && !user.IsAdmin {
        context.Error(fmt.Errorf("insufficient permissions"), http.StatusForbidden)
        return false
    }

    return true
}
```

##### Model with Disabled Endpoints

```golang
type AuditLog struct {
    LogID     int    `gorm:"primaryKey;autoIncrement" json:"log_id"`
    UserID    int    `gorm:"column:user_id" json:"user_id"`
    Action    string `gorm:"column:action;size:255" json:"action"`
    Details   string `gorm:"column:details;type:text" json:"details"`
    IPAddress string `gorm:"column:ip_address;size:45" json:"ip_address"`
    model.CreatedAt
    restify.API
    restify.DisableUpdate // Audit logs should not be updated
    restify.DisableDelete // Audit logs should not be deleted
    restify.DisableSet    // Disable set operations
}
```
