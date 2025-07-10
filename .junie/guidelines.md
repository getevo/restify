# Restify Development Guidelines

## Overview
Restify is a Go REST API framework built on top of the EVO framework. It provides automatic CRUD operations, authentication, permissions, and API documentation generation for GORM models.

## Build/Configuration Instructions

### Prerequisites
- Go 1.23.5 or later
- Database (SQLite, MySQL, or SQL Server)

### Project Setup
1. **Initialize Go Module**:
   ```bash
   go mod init your-project-name
   go mod tidy
   ```

2. **Configuration File**: Create a `config.yml` file with database and HTTP settings:
   ```yaml
   Database:
     Cache: "false"
     ConnMaxLifTime: 1h
     Database: "your_database_name"
     Debug: "3"
     Enabled: true
     MaxIdleConns: "10"
     MaxOpenConns: "100"
     Type: sqlite  # or mysql, sqlserver
     Username: root
     Password: ""
     Server: ""  # for remote databases

   HTTP:
     Host: 0.0.0.0
     Port: 8080
     BodyLimit: 10mb
     ReadTimeout: 1s
     WriteTimeout: 5s
   ```

3. **Main Application Structure**:
   ```go
   package main

   import (
       "github.com/getevo/evo/v2"
       "github.com/getevo/restify"
       "your-project/apps/your-app"
   )

   func main() {
       evo.Setup()
       evo.Register(restify.App{}, yourapp.App{})
       evo.Run()
   }
   ```

4. **Application Registration**: Each app must implement the EVO application interface:
   ```go
   type App struct{}

   func (app App) Register() error {
       // Configure restify
       restify.SetPrefix("/admin/rest")
       restify.EnablePostman()

       // Register models
       db.UseModel(YourModel{})
       evo.GetDBO().AutoMigrate(&YourModel{})
       return nil
   }

   func (app App) Router() error { return nil }
   func (app App) WhenReady() error { return nil }
   func (app App) Priority() application.Priority { return application.DEFAULT }
   func (app App) Name() string { return "your-app-name" }
   ```

### Running the Application
```bash
# Development
go run main.go

# Build
go build -o app main.go

# Run with specific config
EVO_CONFIG=config.yml go run main.go
```

## Testing Information

### Test Configuration
Create a separate test configuration file (`test_config.yml`) with a test database:
```yaml
Database:
  Database: "test.sqlite"
  Type: sqlite
  Enabled: true
  Debug: "3"
```

### Running Tests
```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestName

# Run tests with test config
EVO_CONFIG=test_config.yml go test -v

# Run tests with coverage
go test -v -cover
```

### Writing Tests
Tests should focus on:
1. **Unit Tests**: Test individual functions and methods
2. **Configuration Tests**: Test prefix settings and error handling
3. **Model Tests**: Test model validation and hooks (when possible without full framework setup)

Example test structure:
```go
package restify

import "testing"

func TestPrefix(t *testing.T) {
    expectedPrefix := "/admin/rest"
    if Prefix != expectedPrefix {
        t.Errorf("Expected prefix '%s', got '%s'", expectedPrefix, Prefix)
    }
}

func TestErrorMessages(t *testing.T) {
    err := NewError("Bad Request", 400)
    if err.Code != 400 {
        t.Errorf("Expected error code 400, got %d", err.Code)
    }
}
```

### Adding New Tests
1. Create `*_test.go` files in the same package
2. Use descriptive test function names starting with `Test`
3. Test both success and failure scenarios
4. Clean up any test data created during tests
5. Use table-driven tests for multiple test cases

## Development Information

### Code Style Guidelines

#### Naming Conventions
- **Structs**: PascalCase (`User`, `OrderItem`)
- **Functions/Methods**: PascalCase for exported, camelCase for private
- **Variables**: camelCase (`userID`, `orderTotal`)
- **Constants**: PascalCase or UPPER_SNAKE_CASE
- **Interfaces**: PascalCase, often ending with -er (`Handler`, `Validator`)

#### Model Definition Patterns
```go
type YourModel struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Name        string    `gorm:"size:100;not null" json:"name" validation:"required"`
    Email       string    `gorm:"size:255;unique" json:"email" validation:"required,email"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    restify.API           // Enables REST endpoints
}

func (YourModel) TableName() string {
    return "your_models"
}
```

#### Hook Methods
Implement lifecycle hooks for custom business logic:
```go
func (m *YourModel) OnBeforeCreate(context *restify.Context) error {
    // Validation, data transformation
    return nil
}

func (m *YourModel) OnAfterCreate(context *restify.Context) error {
    // Logging, notifications
    return nil
}

func (m *YourModel) RestPermission(permissions restify.Permissions, context *restify.Context) bool {
    // Custom permission logic
    return true
}
```

#### Error Handling
- Use `restify.NewError(message string, code int)` for HTTP errors
- Return `*restify.Error` from handlers and hooks
- Use predefined errors when appropriate: `ErrorObjectNotExist`, `ErrorPermissionDenied`, etc.

#### Global Hooks
Register global hooks in your app's Register method:
```go
restify.OnBeforeSave(func(obj any, c *restify.Context) error {
    // Global validation logic
    return nil
})
```

### Debugging

#### Enable Debug Mode
1. Set database debug in config: `Debug: "3"`
2. Use query parameter: `?debug=restify`
3. Chain `.Debug()` on GORM queries

#### Common Debug Patterns
```go
// Enable SQL logging
context.DBO = context.DBO.Debug()

// Custom logging in hooks
func (m *Model) OnBeforeCreate(context *restify.Context) error {
    fmt.Printf("Creating model: %+v\n", m)
    return nil
}
```

#### Postman Integration
- Enable with `restify.EnablePostman()`
- Access collection at `/admin/rest/postman`
- Set authentication: `restify.SetPostmanAuthorization(restify.AuthTypeBasic)`

### Key Dependencies
- **EVO Framework**: Core application framework
- **Fiber**: HTTP web framework
- **GORM**: ORM for database operations
- **Postman**: API documentation generation

### Project Structure Best Practices
```
project/
├── main.go                 # Application entry point
├── config.yml             # Configuration file
├── apps/                  # Application modules
│   └── user/
│       ├── app.go         # App registration
│       └── models.go      # Model definitions
├── docs/                  # Documentation
└── example/               # Usage examples
```

### Performance Considerations
- Use database indexes on frequently queried fields
- Implement pagination for large datasets
- Use `Preload()` for eager loading relationships
- Consider caching for read-heavy operations
- Monitor slow queries with `SlowQueryThreshold` setting

### Security Best Practices
- Implement proper authentication and authorization
- Use `RestPermission` methods for fine-grained access control
- Validate all input data using validation tags
- Sanitize user input in hooks
- Use HTTPS in production
- Implement rate limiting for API endpoints

### Error Handling Best Practices
- Always return `*Error` types from REST handlers
- Use structured errors with appropriate HTTP status codes
- Provide detailed error context for debugging
- Log errors with comprehensive information
- Handle validation errors separately using `AddValidationErrors`
- Implement custom error types for domain-specific scenarios
- Use predefined errors for common scenarios
- Include trace IDs for error tracking and debugging
- Test error scenarios thoroughly in your applications
