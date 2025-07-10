package restify

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/db/schema"
	"github.com/getevo/postman"
	"gorm.io/gorm"
	"math"
)

// Prefix defines the base URL path for all REST API endpoints.
// Default value is "/admin/rest" but can be changed using SetPrefix().
// All generated endpoints will be prefixed with this value.
// Example: if Prefix is "/api/v1", endpoints will be "/api/v1/users", "/api/v1/posts", etc.
var Prefix = "/admin/rest"

// onReady stores callback functions that will be executed when the application is ready.
// These callbacks are registered using the Ready() function and executed during WhenReady().
// Useful for initialization logic that needs to run after all models are registered.
var onReady []func()

// permissionHandler is a global permission handler function that determines access control.
// It's called for every API request to check if the user has permission to perform the action.
// Can be set using SetPermissionHandler() to implement custom authorization logic.
// If not set, model-specific RestPermission methods are used for authorization.
var permissionHandler func(permissions Permissions, context *Context) bool

// collection holds the Postman collection for API documentation generation.
// It's initialized when the app registers and populated with endpoints as models are processed.
// The collection can be accessed via the /postman endpoint if Postman integration is enabled.
var collection *postman.Collection

// App represents the main Restify application that integrates with the EVO framework.
// It implements the EVO application interface (Register, Router, WhenReady, Priority, Name).
// This struct is registered with EVO to initialize the REST API functionality.
type App struct{}

// Register initializes the Restify application during the EVO framework startup.
// This method is called automatically by the EVO framework during application registration.
// It performs the following initialization tasks:
//   - Validates that the database is enabled (required for Restify to function)
//   - Sets up database context preparation for debugging and localization
//   - Initializes the Postman collection for API documentation
//   - Configures authentication for the Postman collection if specified
//
// Returns an error if the database is not enabled, as Restify cannot function without it.
func (app App) Register() error {
	// Ensure database is enabled - Restify requires database functionality
	if !db.Enabled {
		return fmt.Errorf("database is not enabled. restify plugin cannot be registered without a running database. please enable the database in your configuration file")
	}

	// Set up database context preparation hook for enhanced functionality
	db.OnPrepareContext(func(db *gorm.DB, v interface{}) *gorm.DB {
		// Check if the context is a Restify context
		if context, ok := v.(*Context); ok {
			// Enable SQL debugging when ?debug=restify query parameter is present
			// This helps developers see the actual SQL queries being executed
			if context.Request.Query("debug").String() == "restify" {
				db = db.Debug()
			}

			// Set language context for internationalization support
			// Check for language in HTTP header first, then fall back to cookie
			if lang := context.Request.Header("language"); lang != "" {
				db = db.Set("lang", lang)
			} else if lang := context.Request.Cookie("l10n-language"); lang != "" {
				db = db.Set("lang", lang)
			}
		}

		return db
	})

	// Initialize Postman collection for API documentation generation
	collection = postman.NewCollection("Restify", "")

	// Configure authentication for Postman collection if specified
	// This allows the generated Postman collection to include auth configuration
	if postmanAuthType != "none" {
		collection.Auth = &postman.Auth{
			Type: postman.AuthType(postmanAuthType),
		}
	}
	return nil
}

// Router is called by the EVO framework to set up application-specific routes.
// Currently, Restify doesn't register any routes during this phase as all routes
// are dynamically generated based on registered models during the WhenReady phase.
// This method is required by the EVO application interface but has no implementation.
func (app App) Router() error {
	// No routes are registered here - all routes are generated dynamically
	// based on registered models during the WhenReady phase
	return nil
}

// WhenReady is called by the EVO framework after all applications have been registered.
// This is where Restify performs the main initialization and route registration:
//   - Processes all registered GORM models and creates REST endpoints for them
//   - Registers global lifecycle hooks for all models
//   - Sets up the main models endpoint for API introspection
//   - Executes all registered ready callbacks
//   - Registers all generated routes for each resource and action
//   - Optionally registers the Postman collection endpoint
//
// This method ensures all REST endpoints are available before the server starts.
func (app App) WhenReady() error {
	// Process all registered GORM models and create REST resources for them
	// This iterates through the schema.Models slice and calls UseModel for each
	for idx := range schema.Models {
		var model = schema.Models[idx]
		UseModel(model.Sample) // Create REST resource for this model
	}

	// Register global lifecycle hooks that apply to all models
	app.registerHooks()

	// Create controller instance for handling system endpoints
	var controller Controller

	// Register the models endpoint for API introspection
	// This endpoint returns information about all available models and their fields
	evo.Get(Prefix+"/models", controller.ModelsHandler)

	// Execute all registered ready callbacks
	// These are custom initialization functions registered via Ready()
	for _, fn := range onReady {
		fn()
	}

	// Register all generated routes for each resource and action
	// This creates the actual REST endpoints (GET, POST, PUT, DELETE, etc.)
	for idx := range Resources {
		for i := range Resources[idx].Actions {
			Resources[idx].Actions[i].RegisterRouter()
		}
	}

	// Register Postman collection endpoint if Postman integration is enabled
	// This provides access to the generated Postman collection for API testing
	if postmanRegistered {
		evo.Get(Prefix+"/postman", controller.PostmanHandler)
	}
	return nil
}

// Ready registers a callback function to be executed when the application is ready.
// This is useful for custom initialization logic that needs to run after all models
// have been registered but before the server starts accepting requests.
//
// Example usage:
//
//	restify.Ready(func() {
//	    // Custom initialization logic
//	    log.Println("Restify is ready!")
//	})
//
// Multiple callbacks can be registered, and they will be executed in registration order.
func Ready(fn func()) {
	onReady = append(onReady, fn)
}

// Priority returns the application priority for the EVO framework.
// Restify uses the maximum integer value to ensure it runs after all other applications.
// This is important because Restify needs to process models that may be registered
// by other applications during their initialization.
func (app App) Priority() application.Priority {
	return math.MaxInt32
}

// Name returns the application name for the EVO framework.
// This is used for logging and identification purposes within the EVO ecosystem.
func (app App) Name() string {
	return "restify"
}
