## Context

The Context object is a powerful tool in Restify that provides developers with fine-grained control over API endpoint behavior. It allows you to enforce conditions, override data, handle errors, and manage validation throughout the request lifecycle.

---

## Context Methods Overview

| **Method**              | **Description**                                                    | **Use Case**                                    |
|-------------------------|--------------------------------------------------------------------|-------------------------------------------------|
| `SetCondition`          | Adds WHERE conditions to database queries                          | Filtering data based on user permissions       |
| `Override`              | Modifies data before database operations                           | Setting default values or enforcing ownership  |
| `Error`                 | Returns custom error responses                                     | Authentication failures, permission denials    |
| `AddValidationErrors`   | Adds custom validation errors                                      | Business logic validation                       |
| `GetDBO`                | Gets the database connection with applied conditions               | Custom database operations                      |

---

## Forced Conditions

The `SetCondition` method allows you to append custom WHERE conditions to `select`, `update`, and `delete` queries. This is particularly useful for implementing row-level security and ensuring users can only access data they're authorized to see.

### Method Signature
```golang
func (context *Context) SetCondition(field string, operator string, value interface{})
```

### Parameters
- **field**: The database column name
- **operator**: SQL operator (`=`, `!=`, `>`, `<`, `>=`, `<=`, `IN`, `LIKE`, etc.)
- **value**: The value to compare against

### Examples

#### Basic Condition
```golang
// Only allow users to see their own records
context.SetCondition("user_id", "=", currentUser.ID)
```

#### Multiple Conditions
```golang
// Users can only see active records they own
context.SetCondition("user_id", "=", currentUser.ID)
context.SetCondition("status", "=", "active")
```

#### Advanced Conditions
```golang
// Allow access to records created in the last 30 days
context.SetCondition("created_at", ">=", time.Now().AddDate(0, 0, -30))

// Allow access to specific categories
context.SetCondition("category_id", "IN", []int{1, 2, 3})
```

---

## Override

The `Override` method allows you to modify or set specific field values before data is submitted to the database. This ensures data integrity and enforces business rules automatically.

### Method Signature
```golang
func (context *Context) Override(data interface{})
```

### Examples

#### Setting Ownership
```golang
// Automatically set the current user as the owner
context.Override(Article{
    UserID: currentUser.ID,
})
```

#### Setting Timestamps and Status
```golang
// Set creation timestamp and default status
context.Override(Order{
    CreatedBy: currentUser.ID,
    Status:    "pending",
    CreatedAt: time.Now(),
})
```

#### Conditional Override
```golang
func (article *Article) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    user, err := GetUser(context.Request)
    if err != nil {
        context.Error(err, http.StatusUnauthorized)
        return false
    }

    // For create operations, always set the current user as owner
    if permissions.Has("CREATE") {
        context.Override(Article{
            UserID:    user.UserID,
            Status:    "draft",
            CreatedAt: time.Now(),
        })
    }

    // For update operations, prevent changing ownership
    if permissions.Has("UPDATE") {
        context.Override(Article{
            UserID: user.UserID, // Ensure ownership cannot be changed
        })
    }

    return true
}
```

---

## Error Handling

The `Error` method allows you to return custom error responses with specific HTTP status codes. This is essential for proper API error handling and user feedback.

### Method Signature
```golang
func (context *Context) Error(err error, statusCode int)
```

### Examples

#### Authentication Errors
```golang
if !isAuthenticated {
    context.Error(fmt.Errorf("authentication required"), http.StatusUnauthorized)
    return false
}
```

#### Permission Errors
```golang
if !user.HasPermission("admin") {
    context.Error(fmt.Errorf("insufficient permissions"), http.StatusForbidden)
    return false
}
```

#### Business Logic Errors
```golang
if order.Status == "completed" {
    context.Error(fmt.Errorf("cannot modify completed orders"), http.StatusBadRequest)
    return false
}
```

---

## Custom Validation Errors

The `AddValidationErrors` method allows you to add custom validation errors that will be returned in the API response. This is useful for implementing business logic validation beyond basic field validation.

### Method Signature
```golang
func (context *Context) AddValidationErrors(err error)
```

### Examples

#### Single Validation Error
```golang
func (app App) Register() error {
    restify.OnBeforeSave(func(obj any, c *restify.Context) error {
        if user, ok := obj.(*User); ok {
            if user.Username == "admin" && !user.IsAdmin {
                c.AddValidationErrors(fmt.Errorf("username 'admin' is reserved for administrators"))
                return fmt.Errorf("validation error")
            }
        }
        return nil
    })
    return nil
}
```

#### Multiple Validation Errors
```golang
func (product *Product) ValidateCreate(context *restify.Context) error {
    if product.Price <= 0 {
        context.AddValidationErrors(fmt.Errorf("price must be greater than zero"))
    }

    if len(product.Name) < 3 {
        context.AddValidationErrors(fmt.Errorf("product name must be at least 3 characters"))
    }

    if product.CategoryID == 0 {
        context.AddValidationErrors(fmt.Errorf("category is required"))
    }

    // Return error if any validation failed
    if len(context.ValidationErrors) > 0 {
        return fmt.Errorf("validation failed")
    }

    return nil
}
```

---

## Complete Example: Multi-Tenant Application

Here's a comprehensive example showing how to use Context methods together in a multi-tenant application:

```golang
type Document struct {
    DocumentID int    `gorm:"primaryKey;autoIncrement"`
    TenantID   int    `gorm:"column:tenant_id;not null"`
    UserID     int    `gorm:"column:user_id;not null"`
    Title      string `gorm:"column:title;size:255"`
    Content    string `gorm:"column:content;type:text"`
    Status     string `gorm:"column:status;default:'draft'"`
    IsPublic   bool   `gorm:"column:is_public;default:false"`
    model.CreatedAt
    model.UpdatedAt
    restify.API
}

func (doc *Document) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    user, err := GetCurrentUser(context.Request)
    if err != nil {
        context.Error(fmt.Errorf("authentication required"), http.StatusUnauthorized)
        return false
    }

    // Enforce tenant isolation for all operations
    context.SetCondition("tenant_id", "=", user.TenantID)

    // For viewing operations, users can see their own documents or public ones
    if permissions.Has("VIEW") {
        if !user.IsAdmin {
            // Non-admin users can only see their own documents or public ones
            context.SetCondition("user_id", "=", user.UserID)
            // Note: This is a simplified example. In practice, you might need more complex logic for public documents
        }
    }

    // For create operations, set ownership and tenant
    if permissions.Has("CREATE") {
        context.Override(Document{
            TenantID: user.TenantID,
            UserID:   user.UserID,
            Status:   "draft",
        })
    }

    // For update operations, ensure ownership cannot be changed
    if permissions.Has("UPDATE") {
        context.SetCondition("user_id", "=", user.UserID)
        context.Override(Document{
            TenantID: user.TenantID, // Prevent tenant switching
            UserID:   user.UserID,   // Prevent ownership transfer
        })
    }

    // For delete operations, only allow owners or admins
    if permissions.Has("DELETE") {
        if !user.IsAdmin {
            context.SetCondition("user_id", "=", user.UserID)
        }
    }

    return true
}

func (doc *Document) ValidateCreate(context *restify.Context) error {
    if len(doc.Title) < 5 {
        context.AddValidationErrors(fmt.Errorf("title must be at least 5 characters long"))
    }

    if doc.Status != "" && !isValidStatus(doc.Status) {
        context.AddValidationErrors(fmt.Errorf("invalid status: %s", doc.Status))
    }

    return nil
}

func (doc *Document) OnBeforeCreate(context *restify.Context) error {
    // Set default values
    if doc.Status == "" {
        doc.Status = "draft"
    }
    return nil
}

func isValidStatus(status string) bool {
    validStatuses := []string{"draft", "published", "archived"}
    for _, v := range validStatuses {
        if v == status {
            return true
        }
    }
    return false
}
```

---

## Debugging Context Operations

You can enable debug mode to see the SQL queries generated by Restify, including the conditions and overrides applied through the Context:

```bash
# Add debug=restify to any endpoint to see SQL queries
curl --location 'http://localhost:8080/admin/rest/document/all?debug=restify'
```

This will output the generated SQL queries to the console, helping you understand how your Context operations are being applied.
