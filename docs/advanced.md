# Advanced Features

This document covers advanced Restify features and functions that provide additional control and customization options for your REST API.

---

## Ready Function

The `restify.Ready()` function allows you to register callback functions that will be executed when Restify has finished initializing and is ready to handle requests. This is useful for performing setup tasks that depend on Restify being fully configured.

### Method Signature
```golang
func Ready(fn func())
```

### Use Cases
- Performing database seeding after models are registered
- Setting up additional routes that depend on Restify resources
- Initializing external services that integrate with your API
- Running post-initialization validation or setup

### Examples

#### Basic Usage
```golang
func (app App) Register() error {
    // Register models first
    db.UseModel(User{}, Product{}, Order{})

    // Register a callback to run when Restify is ready
    restify.Ready(func() {
        fmt.Println("Restify is ready and all endpoints are registered!")

        // Perform any post-initialization tasks here
        seedDatabase()
        setupMetrics()
    })

    return nil
}

func seedDatabase() {
    // Check if database needs seeding
    var count int64
    db.Model(&User{}).Count(&count)

    if count == 0 {
        // Create default admin user
        adminUser := User{
            Username: "admin",
            Password: "admin123",
            IsAdmin:  true,
            Name:     "Administrator",
            Email:    "admin@example.com",
        }
        db.Create(&adminUser)
        fmt.Println("Default admin user created")
    }
}

func setupMetrics() {
    // Initialize monitoring or analytics
    fmt.Println("Metrics and monitoring initialized")
}
```

#### Multiple Ready Callbacks
```golang
func (app App) Register() error {
    // You can register multiple Ready callbacks
    restify.Ready(func() {
        fmt.Println("First callback: Setting up logging")
        setupLogging()
    })

    restify.Ready(func() {
        fmt.Println("Second callback: Initializing cache")
        initializeCache()
    })

    restify.Ready(func() {
        fmt.Println("Third callback: Starting background jobs")
        startBackgroundJobs()
    })

    return nil
}
```

#### Integration with External Services
```golang
func (app App) Register() error {
    restify.Ready(func() {
        // Initialize external service connections
        initializeRedis()
        setupElasticsearch()
        configureMessageQueue()

        // Register custom middleware that depends on these services
        registerCustomMiddleware()
    })

    return nil
}

func initializeRedis() {
    // Redis connection setup
    fmt.Println("Redis connection established")
}

func setupElasticsearch() {
    // Elasticsearch setup for search functionality
    fmt.Println("Elasticsearch configured")
}

func configureMessageQueue() {
    // Message queue setup for async processing
    fmt.Println("Message queue configured")
}

func registerCustomMiddleware() {
    // Add custom middleware that uses the initialized services
    evo.Use(func(c *fiber.Ctx) error {
        // Custom middleware logic
        return c.Next()
    })
}
```

---

## Debug Mode

Restify provides built-in debugging capabilities that help you understand how your API endpoints are processing requests and generating SQL queries.

### Enabling Debug Mode

Add the `debug=restify` query parameter to any Restify endpoint to enable debug mode for that request:

```bash
# Debug a specific request
curl --location 'http://localhost:8080/admin/rest/user/all?debug=restify'

# Debug with filters
curl --location 'http://localhost:8080/admin/rest/user/all?debug=restify&name[contains]=john'

# Debug pagination
curl --location 'http://localhost:8080/admin/rest/user/paginate?debug=restify&page=1&size=10'
```

### What Debug Mode Shows

When debug mode is enabled, you'll see detailed information in your application logs:

1. **Generated SQL Queries**: The exact SQL queries being executed
2. **Query Parameters**: The values being bound to the SQL queries
3. **Execution Time**: How long each query takes to execute
4. **Applied Conditions**: Any conditions added through Context.SetCondition
5. **Override Values**: Any data overrides applied through Context.Override

### Example Debug Output

```sql
-- Debug output in console
[2024-01-15 10:30:45] [info] [database] SELECT * FROM `user` WHERE `user`.`deleted_at` IS NULL AND ((`name` LIKE '%john%')) ORDER BY `user`.`user_id` LIMIT 10
[2024-01-15 10:30:45] [info] [database] Query took: 2.5ms
[2024-01-15 10:30:45] [info] [restify] Applied conditions: name LIKE %john%
[2024-01-15 10:30:45] [info] [restify] Total records found: 3
```

### Programmatic Debug Mode

You can also enable debug mode programmatically in your hooks or permission handlers:

```golang
func (user *User) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    // Enable debug mode for this specific request
    if context.Request.Query("admin_debug").String() == "true" {
        context.GetDBO().Debug()
    }

    return true
}
```

---

## Language Support

Restify automatically handles language headers for internationalization support. The language information is passed to the database context and can be used for localized queries.

### Supported Headers

Restify checks for language information in the following order:

1. **HTTP Header**: `language: en-US`
2. **Cookie**: `l10n-language=en-US`

### Usage in Database Queries

The language information is automatically set in the database context and can be accessed in your custom queries:

```golang
func (product *Product) OnAfterGet(context *restify.Context) error {
    // Get the language from the database context
    dbo := context.GetDBO()

    // Use language-specific queries if needed
    if lang, ok := dbo.Get("lang"); ok {
        fmt.Printf("Request language: %s\n", lang)

        // Load localized content based on language
        if lang == "es" {
            // Load Spanish content
        } else if lang == "fr" {
            // Load French content
        }
    }

    return nil
}
```

### Example Request with Language

```bash
# Request with language header
curl --location 'http://localhost:8080/admin/rest/product/all' \
--header 'language: es-ES'

# Request with language cookie
curl --location 'http://localhost:8080/admin/rest/product/all' \
--cookie 'l10n-language=fr-FR'
```

---

## Custom Database Context

Restify provides access to the underlying database connection through the Context object, allowing you to perform custom database operations while maintaining the same transaction and context.

### Getting Database Connection

```golang
func (order *Order) OnBeforeCreate(context *restify.Context) error {
    // Get the database connection with applied conditions
    dbo := context.GetDBO()

    // Perform custom database operations
    var productCount int64
    dbo.Model(&Product{}).Where("product_id = ?", order.ProductID).Count(&productCount)

    if productCount == 0 {
        context.AddValidationErrors(fmt.Errorf("product does not exist"))
        return fmt.Errorf("validation error")
    }

    return nil
}
```

### Transaction Support

The database connection obtained through `GetDBO()` automatically participates in Restify's transaction management:

```golang
func (order *Order) OnAfterCreate(context *restify.Context) error {
    dbo := context.GetDBO()

    // This operation will be part of the same transaction
    // If it fails, the entire operation will be rolled back
    err := dbo.Model(&Product{}).
        Where("product_id = ?", order.ProductID).
        Update("stock_quantity", gorm.Expr("stock_quantity - ?", order.Quantity)).Error

    if err != nil {
        return fmt.Errorf("failed to update product stock: %w", err)
    }

    return nil
}
```

---

## Performance Tips

### 1. Use Selective Field Loading

When you don't need all fields, use the `fields` parameter to reduce data transfer:

```bash
# Only load specific fields
curl --location 'http://localhost:8080/admin/rest/user/all?fields=user_id,username,email'
```

### 2. Optimize Association Loading

Be selective about which associations you load:

```bash
# Load specific associations only
curl --location 'http://localhost:8080/admin/rest/user/1?associations=Orders'

# Avoid loading all associations unless necessary
curl --location 'http://localhost:8080/admin/rest/user/1?associations=*'  # Use sparingly
```

### 3. Use Pagination for Large Datasets

Always use pagination for endpoints that might return large amounts of data:

```bash
# Use pagination instead of /all for large datasets
curl --location 'http://localhost:8080/admin/rest/user/paginate?page=1&size=50'
```

### 4. Implement Efficient Conditions

Use database indexes for fields commonly used in conditions:

```golang
func (user *User) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    // Use indexed fields for conditions when possible
    context.SetCondition("tenant_id", "=", currentTenant.ID)  // Make sure tenant_id is indexed
    context.SetCondition("status", "=", "active")             // Make sure status is indexed

    return true
}
```

---

## Custom Endpoints

Restify allows you to add custom endpoints with custom features beyond the default CRUD operations. This is particularly useful when you need specialized business logic, custom filtering, or endpoints that don't follow the standard REST patterns.

### Adding Custom Endpoints

You can add custom endpoints using `restify.Ready()` combined with `resource.SetAction()`. This approach allows you to:

- Create custom business logic endpoints
- Implement specialized filtering and permissions
- Add endpoints with custom URL patterns
- Integrate with external services or complex workflows

### Method Signatures

```golang
// Get a resource for a specific model
func GetResource(model interface{}) (*Resource, error)

// Add a custom endpoint to a resource
func (resource *Resource) SetAction(endpoint *Endpoint)

// Endpoint structure for custom endpoints
type Endpoint struct {
    Name        string                           // Unique name for the endpoint
    Method      string                          // HTTP method (GET, POST, PUT, PATCH, DELETE)
    AbsoluteURI string                          // Custom URL path
    Handler     func(*Context) *Error           // Custom handler function
    Filterable  bool                           // Whether the endpoint supports filtering
    PKUrl       bool                           // Whether the endpoint uses primary key in URL
    Description string                          // Description for documentation
}
```

### Complete Example: Point of Sale (POS) System

Here's a comprehensive example showing how to add custom endpoints for a Point of Sale system:

```golang
func (app App) Register() error {
    // Register models first
    db.UseModel(models.Order{})

    // Add custom endpoints when Restify is ready
    restify.Ready(func() {
        resource, err := restify.GetResource(models.Order{})
        if err != nil {
            log.Fatal(err)
        }

        // Custom endpoint for current user's orders
        resource.SetAction(&restify.Endpoint{
            Name:        "GetSelfOrders",
            Method:      restify.MethodGET,
            AbsoluteURI: "/api/v1/pos/self/orders",
            Handler:     controller.getSelfOrdersHandler,
            Filterable:  true,
            Description: "get order list of the current user",
        })

        // Custom endpoint for ship orders
        resource.SetAction(&restify.Endpoint{
            Name:        "GetShipOrders",
            Method:      restify.MethodGET,
            AbsoluteURI: "/api/v1/pos/ship/orders",
            Handler:     controller.getShipOrdersHandler,
            Filterable:  true,
            Description: "get order list of the current ship",
        })

        // Custom endpoint for store orders
        resource.SetAction(&restify.Endpoint{
            Name:        "GetStoreOrders",
            Method:      restify.MethodGET,
            AbsoluteURI: "/api/v1/pos/store/orders",
            Handler:     controller.getStoreOrdersHandler,
            Filterable:  true,
            Description: "get order list of the sales channel",
        })

        // Custom endpoint for single order with primary key
        resource.SetAction(&restify.Endpoint{
            Name:        "GetSingleOrder",
            Method:      restify.MethodGET,
            PKUrl:       true,
            AbsoluteURI: "/api/v1/pos/order",
            Handler:     controller.getOrderHandler,
            Description: "get single order by id",
        })
    })

    return nil
}
```

### Custom Handler Functions

Each custom endpoint requires a handler function that implements your business logic:

```golang
type Controller struct{}

// Handler for user's own orders
func (c Controller) getSelfOrdersHandler(context *restify.Context) *restify.Error {
    // Check authentication
    if context.Request.User().Anonymous() {
        return &restifyUnauthorized
    }

    // Validate API key
    if context.Request.Header("APIKEY") != settings.Get("APP.API_KEY").String() {
        return &restifyInvalidAPIKey
    }

    // Get current user
    var user = context.Request.User().Interface().(*auth.User)
    if !user.HasPermission(roles.POSSelfReport) {
        return &restifyInsufficientPermission
    }

    // Apply custom filtering
    var nodeID = context.Request.Header("X-NODE-ID")
    context.ApplyFilter(func(context *restify.Context, db *gorm.DB) *gorm.DB {
        return db.Debug().Where("node_id = ? AND cashier_uuid = ?", nodeID, user.UUID())
    })

    // Use built-in pagination handler
    return (restify.Handler{}).Paginate(context)
}

// Handler for ship orders
func (c Controller) getShipOrdersHandler(context *restify.Context) *restify.Error {
    if context.Request.User().Anonymous() {
        return &restifyUnauthorized
    }

    if context.Request.Header("APIKEY") != settings.Get("APP.API_KEY").String() {
        return &restifyInvalidAPIKey
    }

    var user = context.Request.User().Interface().(*auth.User)
    if !user.HasPermission(roles.POSSalesReport) {
        return &restifyInsufficientPermission
    }

    var nodeID = context.Request.Header("X-NODE-ID")
    context.ApplyFilter(func(context *restify.Context, db *gorm.DB) *gorm.DB {
        return db.Debug().Where("node_id = ?", nodeID)
    })

    return (restify.Handler{}).Paginate(context)
}

// Handler for store orders with complex filtering
func (c Controller) getStoreOrdersHandler(context *restify.Context) *restify.Error {
    if context.Request.User().Anonymous() {
        return &restifyUnauthorized
    }

    if context.Request.Header("APIKEY") != settings.Get("APP.API_KEY").String() {
        return &restifyInvalidAPIKey
    }

    var user = context.Request.User().Interface().(*auth.User)

    // Validate required user fields
    if user.SalesChannelID == nil {
        return &restifySalesChannelIDError
    }
    if user.CompanyID == nil {
        return &restifyCompanyIDError
    }

    if !user.HasPermission(roles.POSStoreOrders) {
        return &restifyInsufficientPermission
    }

    // Apply complex filtering with subquery
    context.ApplyFilter(func(context *restify.Context, db *gorm.DB) *gorm.DB {
        return db.Debug().Where("company_id = ? AND order_id IN (SELECT order_id FROM order_item WHERE sales_channel_id = ? )", *user.CompanyID, user.SalesChannelID)
    })

    return (restify.Handler{}).Paginate(context)
}

// Handler for single order retrieval
func (c Controller) getOrderHandler(context *restify.Context) *restify.Error {
    if context.Request.User().Anonymous() {
        return &restifyUnauthorized
    }

    if context.Request.Header("APIKEY") != settings.Get("APP.API_KEY").String() {
        return &restifyInvalidAPIKey
    }

    var user = context.Request.User().Interface().(*auth.User)
    if !user.HasPermission(roles.POSLogin) {
        return &restifyInsufficientPermission
    }

    // Use built-in Get handler
    return (restify.Handler{}).Get(context)
}
```

### Key Components Explained

#### 1. restify.Ready()
- Ensures custom endpoints are added after Restify initialization
- Guarantees that all resources are available when adding custom actions
- Allows access to fully configured Restify resources

#### 2. restify.GetResource()
- Retrieves the Restify resource for a specific model
- Returns an error if the model is not registered with Restify
- Provides access to the resource's configuration and endpoints

#### 3. Endpoint Configuration
- **Name**: Unique identifier for the endpoint
- **Method**: HTTP method (use restify.MethodGET, restify.MethodPOST, etc.)
- **AbsoluteURI**: Custom URL path for the endpoint
- **Handler**: Your custom handler function
- **Filterable**: Enables query parameter filtering
- **PKUrl**: Adds primary key parameter to the URL
- **Description**: Used for API documentation and Postman collections

#### 4. Custom Handler Pattern
- Receive `*restify.Context` parameter
- Return `*restify.Error` for error handling
- Use `context.ApplyFilter()` for custom database filtering
- Leverage built-in handlers like `Paginate()` and `Get()` when appropriate

### Advanced Handler Techniques

#### Using ApplyFilter for Complex Queries

```golang
func (c Controller) getAdvancedOrdersHandler(context *restify.Context) *restify.Error {
    // Apply multiple filters
    context.ApplyFilter(func(context *restify.Context, db *gorm.DB) *gorm.DB {
        query := db.Debug()

        // Join with related tables
        query = query.Joins("LEFT JOIN users ON orders.user_id = users.user_id")
        query = query.Joins("LEFT JOIN products ON orders.product_id = products.product_id")

        // Apply conditional filtering
        if region := context.Request.Query("region").String(); region != "" {
            query = query.Where("users.region = ?", region)
        }

        if category := context.Request.Query("category").String(); category != "" {
            query = query.Where("products.category = ?", category)
        }

        return query
    })

    return (restify.Handler{}).Paginate(context)
}
```

#### Custom Response Formatting

```golang
func (c Controller) getOrderSummaryHandler(context *restify.Context) *restify.Error {
    // Get filtered data
    var orders []models.Order
    dbo := context.GetDBO()

    if err := dbo.Find(&orders).Error; err != nil {
        return &restify.Error{
            Message: "Failed to fetch orders",
            Code:    http.StatusInternalServerError,
        }
    }

    // Create custom response
    summary := map[string]interface{}{
        "total_orders": len(orders),
        "total_amount": calculateTotalAmount(orders),
        "orders":       orders,
    }

    // Return custom response
    context.Response.JSON(summary)
    return nil
}
```

### Error Handling

Define custom error types for consistent error responses:

```golang
var (
    restifyUnauthorized = restify.Error{
        Message: "Authentication required",
        Code:    http.StatusUnauthorized,
    }

    restifyInvalidAPIKey = restify.Error{
        Message: "Invalid API key",
        Code:    http.StatusForbidden,
    }

    restifyInsufficientPermission = restify.Error{
        Message: "Insufficient permissions",
        Code:    http.StatusForbidden,
    }

    restifySalesChannelIDError = restify.Error{
        Message: "Sales channel ID is required",
        Code:    http.StatusBadRequest,
    }

    restifyCompanyIDError = restify.Error{
        Message: "Company ID is required",
        Code:    http.StatusBadRequest,
    }
)
```

### Testing Custom Endpoints

Once your custom endpoints are configured, you can test them:

```bash
# Test self orders endpoint
curl --location 'http://localhost:8080/api/v1/pos/self/orders' \
--header 'APIKEY: your-api-key' \
--header 'X-NODE-ID: node123' \
--header 'Authorization: Bearer your-token'

# Test ship orders with filtering
curl --location 'http://localhost:8080/api/v1/pos/ship/orders?status[eq]=pending' \
--header 'APIKEY: your-api-key' \
--header 'X-NODE-ID: node123'

# Test single order retrieval
curl --location 'http://localhost:8080/api/v1/pos/order/123' \
--header 'APIKEY: your-api-key'
```

---

## Security Best Practices

### 1. Always Validate Input in Hooks

```golang
func (user *User) ValidateCreate(context *restify.Context) error {
    // Validate email format
    if !isValidEmail(user.Email) {
        context.AddValidationErrors(fmt.Errorf("invalid email format"))
    }

    // Validate password strength
    if !isStrongPassword(user.Password) {
        context.AddValidationErrors(fmt.Errorf("password does not meet security requirements"))
    }

    return nil
}
```

### 2. Implement Row-Level Security

```golang
func (document *Document) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    user := getCurrentUser(context)

    // Always enforce tenant isolation
    context.SetCondition("tenant_id", "=", user.TenantID)

    // Enforce ownership for non-admin users
    if !user.IsAdmin {
        context.SetCondition("user_id", "=", user.UserID)
    }

    return true
}
```

### 3. Sanitize Output Data

```golang
func (user *User) OnAfterGet(context *restify.Context) error {
    // Remove sensitive information before sending response
    user.Password = ""
    user.InternalNotes = ""

    return nil
}
```

### 4. Use HTTPS in Production

Always use HTTPS in production environments and consider implementing additional security headers:

```golang
func (app App) Register() error {
    // Add security middleware
    evo.Use(func(c *fiber.Ctx) error {
        c.Set("X-Content-Type-Options", "nosniff")
        c.Set("X-Frame-Options", "DENY")
        c.Set("X-XSS-Protection", "1; mode=block")
        return c.Next()
    })

    return nil
}
```
