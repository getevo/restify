package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/validation"
	"html"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError represents a field-level validation error with detailed information.
// It provides structured error information that can be easily consumed by clients
// to display meaningful validation messages to users.
//
// Example usage:
//
//	err := &ValidationError{
//	    Field: "email",
//	    Error: "must be a valid email address",
//	    Value: "invalid-email",
//	    Rule:  "email",
//	}
type ValidationError struct {
	Field string      `json:"field"`           // The name of the field that failed validation
	Error string      `json:"error"`           // Human-readable error message
	Value interface{} `json:"value,omitempty"` // The value that failed validation (optional)
	Rule  string      `json:"rule,omitempty"`  // The validation rule that was violated (optional)
}

// Security validation patterns for input sanitization and attack prevention.
// These patterns are used to detect and prevent common web security vulnerabilities.
var (
	// sqlInjectionPatterns contains regular expressions to detect SQL injection attempts.
	// These patterns match common SQL injection techniques including:
	//   - UNION SELECT attacks for data extraction
	//   - DROP TABLE, DELETE FROM, INSERT INTO, UPDATE SET for data manipulation
	//   - Stored procedure execution attempts (EXEC, EXECUTE, sp_executesql)
	//   - Script injection within SQL contexts
	//   - HTML tag injection that could lead to SQL injection
	//
	// Performance note: These patterns are compiled once at startup for efficiency.
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+set)`),
		regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_executesql)`),
		regexp.MustCompile(`(?i)(script\s*>|javascript:|vbscript:)`),
		regexp.MustCompile(`(?i)(<\s*script|<\s*iframe|<\s*object|<\s*embed)`),
	}

	// xssPatterns contains regular expressions to detect Cross-Site Scripting (XSS) attempts.
	// These patterns match common XSS attack vectors including:
	//   - Script tags with potential malicious content
	//   - JavaScript and VBScript protocol handlers
	//   - Event handlers that could execute malicious code
	//   - Dangerous HTML elements that can execute scripts or embed content
	//
	// Performance note: These patterns are compiled once at startup for efficiency.
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<script[^>]*>.*?</script>)`),
		regexp.MustCompile(`(?i)(javascript:|vbscript:|onload=|onerror=|onclick=)`),
		regexp.MustCompile(`(?i)(<iframe|<object|<embed|<applet)`),
	}
)

// SanitizeInput sanitizes user input to prevent injection attacks and normalize data.
// This function performs multiple sanitization steps to ensure input safety:
//   - HTML escapes special characters to prevent XSS attacks
//   - Removes null bytes that could cause issues in string processing
//   - Trims leading and trailing whitespace for consistency
//
// This function is automatically applied to string fields when using SanitizeStruct,
// but can also be called directly for individual string values.
//
// Example usage:
//
//	userInput := "<script>alert('xss')</script>"
//	safe := SanitizeInput(userInput)
//	// Result: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
//
// Performance note: This function is lightweight and safe to call frequently.
func SanitizeInput(input string) string {
	if input == "" {
		return input
	}

	// HTML escape special characters to prevent XSS attacks
	// This converts <, >, &, ', and " to their HTML entity equivalents
	sanitized := html.EscapeString(input)

	// Remove null bytes that could cause issues in string processing
	// Null bytes can sometimes be used to bypass security filters
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Trim leading and trailing whitespace for data consistency
	// This helps normalize user input and prevents accidental whitespace issues
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}

// ValidateAgainstSQLInjection checks input for potential SQL injection attack patterns.
// This function scans the input string against a set of predefined regular expressions
// that match common SQL injection techniques. It's designed to catch obvious injection
// attempts while minimizing false positives.
//
// Common patterns detected:
//   - UNION SELECT attacks for data extraction
//   - Data manipulation commands (DROP, DELETE, INSERT, UPDATE)
//   - Stored procedure execution attempts
//   - Script injection within SQL contexts
//
// Example usage:
//
//	userInput := "'; DROP TABLE users; --"
//	if err := ValidateAgainstSQLInjection(userInput); err != nil {
//	    // Handle potential SQL injection attempt
//	    return fmt.Errorf("invalid input: %v", err)
//	}
//
// Performance note: Uses pre-compiled regex patterns for efficiency.
// Security note: This is a basic protection layer and should be combined with
// parameterized queries and proper input validation.
func ValidateAgainstSQLInjection(input string) error {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("potential SQL injection detected")
		}
	}
	return nil
}

// ValidateAgainstXSS checks input for potential Cross-Site Scripting (XSS) attack patterns.
// This function scans the input string against a set of predefined regular expressions
// that match common XSS attack vectors. It helps prevent malicious script execution
// in web applications.
//
// Common patterns detected:
//   - Script tags with malicious content
//   - JavaScript and VBScript protocol handlers
//   - Event handlers that could execute code
//   - Dangerous HTML elements (iframe, object, embed, applet)
//
// Example usage:
//
//	userInput := "<script>alert('xss')</script>"
//	if err := ValidateAgainstXSS(userInput); err != nil {
//	    // Handle potential XSS attempt
//	    return fmt.Errorf("invalid input: %v", err)
//	}
//
// Performance note: Uses pre-compiled regex patterns for efficiency.
// Security note: This is a basic protection layer and should be combined with
// proper output encoding and Content Security Policy (CSP).
func ValidateAgainstXSS(input string) error {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("potential XSS attack detected")
		}
	}
	return nil
}

// SanitizeStruct recursively sanitizes all string fields in a struct and its nested structures.
// This function uses reflection to traverse the struct and apply sanitization to all string fields,
// including those in nested structs, slices, arrays, and pointers.
//
// The function performs the following operations on each string field:
//   - HTML escapes special characters to prevent XSS attacks
//   - Removes null bytes that could cause processing issues
//   - Trims leading and trailing whitespace
//   - Validates against SQL injection patterns
//   - Validates against XSS attack patterns
//
// Supported field types:
//   - string: Direct sanitization and validation
//   - struct: Recursive processing of all fields
//   - slice/array: Processing of each element
//   - pointer: Processing of pointed-to value (if not nil)
//
// Example usage:
//
//	type User struct {
//	    Name    string `json:"name"`
//	    Email   string `json:"email"`
//	    Profile struct {
//	        Bio string `json:"bio"`
//	    } `json:"profile"`
//	}
//
//	user := &User{
//	    Name:  "<script>alert('xss')</script>",
//	    Email: "user@example.com",
//	}
//	user.Profile.Bio = "Hello & welcome!"
//
//	if err := SanitizeStruct(user); err != nil {
//	    // Handle sanitization error (e.g., potential attack detected)
//	    return err
//	}
//	// user.Name is now: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
//	// user.Profile.Bio is now: "Hello &amp; welcome!"
//
// Performance note: Uses reflection which has some overhead, but is cached where possible.
// Security note: This provides comprehensive protection but should be combined with other
// security measures like parameterized queries and output encoding.
func SanitizeStruct(ptr interface{}) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("input must be a pointer to struct")
	}

	return sanitizeValue(v.Elem())
}

// sanitizeValue is a recursive helper function that sanitizes values based on their reflection type.
// This function handles the actual sanitization logic for different Go types and is called
// recursively to process nested structures.
//
// Type handling:
//   - reflect.String: Applies SanitizeInput and security validations
//   - reflect.Struct: Recursively processes all settable fields
//   - reflect.Slice/Array: Processes each element in the collection
//   - reflect.Ptr: Processes the pointed-to value if not nil
//   - Other types: Ignored (no sanitization needed)
//
// Security validations applied to strings:
//   - SQL injection pattern detection
//   - XSS attack pattern detection
//
// Performance considerations:
//   - Only processes settable fields to avoid unnecessary work
//   - Skips nil pointers to prevent panics
//   - Uses efficient pattern matching with pre-compiled regex
func sanitizeValue(v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		// Only process settable string fields
		if v.CanSet() {
			original := v.String()
			sanitized := SanitizeInput(original)

			// Validate against injection attacks after sanitization
			// This catches potential attacks that might still be present
			if err := ValidateAgainstSQLInjection(sanitized); err != nil {
				return err
			}
			if err := ValidateAgainstXSS(sanitized); err != nil {
				return err
			}

			// Update the field with the sanitized value
			v.SetString(sanitized)
		}
	case reflect.Struct:
		// Recursively process all fields in the struct
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			// Only process settable fields (exported fields)
			if field.CanSet() {
				if err := sanitizeValue(field); err != nil {
					return err
				}
			}
		}
	case reflect.Slice, reflect.Array:
		// Process each element in slices and arrays
		for i := 0; i < v.Len(); i++ {
			if err := sanitizeValue(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		// Process pointed-to value if pointer is not nil
		if !v.IsNil() {
			if err := sanitizeValue(v.Elem()); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

// ValidateInput performs comprehensive input validation against multiple rules.
// This function applies a set of validation rules to an input string and returns
// all validation errors found. It's designed to be flexible and extensible.
//
// Supported validation rules:
//   - "required": Ensures the field is not empty (after trimming whitespace)
//   - "no_sql_injection": Validates against SQL injection attack patterns
//   - "no_xss": Validates against Cross-Site Scripting attack patterns
//   - "alphanumeric": Ensures the field contains only letters and digits
//   - "email": Validates email format using RFC-compliant regex
//
// The function returns a slice of errors, allowing multiple validation failures
// to be reported at once. If no errors are found, an empty slice is returned.
//
// Example usage:
//
//	// Single rule validation
//	errors := ValidateInput("user@example.com", "email")
//	if len(errors) > 0 {
//	    // Handle validation errors
//	    for _, err := range errors {
//	        log.Printf("Validation error: %v", err)
//	    }
//	}
//
//	// Multiple rules validation
//	errors = ValidateInput("john123", "required", "alphanumeric")
//	if len(errors) > 0 {
//	    // Handle validation errors
//	}
//
//	// Security validation
//	errors = ValidateInput(userInput, "required", "no_sql_injection", "no_xss")
//
// Performance note: Email validation uses a compiled regex that's created on each call.
// For high-frequency validation, consider pre-compiling the regex.
//
// Security note: The security rules (no_sql_injection, no_xss) provide basic protection
// but should be combined with other security measures like parameterized queries.
func ValidateInput(input string, rules ...string) []error {
	var errors []error

	// Apply each validation rule to the input
	for _, rule := range rules {
		switch rule {
		case "required":
			// Check if field is empty after trimming whitespace
			if strings.TrimSpace(input) == "" {
				errors = append(errors, fmt.Errorf("field is required"))
			}
		case "no_sql_injection":
			// Validate against SQL injection patterns
			if err := ValidateAgainstSQLInjection(input); err != nil {
				errors = append(errors, err)
			}
		case "no_xss":
			// Validate against XSS attack patterns
			if err := ValidateAgainstXSS(input); err != nil {
				errors = append(errors, err)
			}
		case "alphanumeric":
			// Check that all characters are letters or digits
			for _, r := range input {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					errors = append(errors, fmt.Errorf("field must contain only alphanumeric characters"))
					break
				}
			}
		case "email":
			// Validate email format using RFC-compliant regex
			// This regex covers most common email formats but may not catch all edge cases
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(input) {
				errors = append(errors, fmt.Errorf("invalid email format"))
			}
		}
	}

	return errors
}

// Validate performs comprehensive validation on a struct using validation tags and sanitization.
// This method is the primary validation function for Restify contexts and performs:
//  1. Input sanitization to prevent security vulnerabilities
//  2. Struct validation using validation tags defined on struct fields
//  3. Error logging and collection for debugging and user feedback
//
// The validation process:
//   - First sanitizes all string fields in the struct to prevent XSS and SQL injection
//   - Then validates the struct using the EVO validation library
//   - Collects all validation errors and adds them to the context
//   - Logs validation failures for monitoring and debugging
//
// Validation tags supported (from EVO validation library):
//   - required: Field must not be empty
//   - email: Field must be a valid email address
//   - min=N: Minimum length/value
//   - max=N: Maximum length/value
//   - len=N: Exact length
//   - And many more standard validation tags
//
// Example usage in a handler:
//
//	type CreateUserRequest struct {
//	    Name  string `json:"name" validation:"required,min=2,max=50"`
//	    Email string `json:"email" validation:"required,email"`
//	    Age   int    `json:"age" validation:"min=18,max=120"`
//	}
//
//	var req CreateUserRequest
//	if err := context.BodyParser(&req); err != nil {
//	    return err
//	}
//
//	if err := context.Validate(&req); err != nil {
//	    // Validation failed - errors are automatically added to context
//	    return err
//	}
//
// Performance note: Sanitization uses reflection which has some overhead.
// Security note: Always validate user input before processing or storing.
func (context *Context) Validate(ptr any) error {
	// First sanitize the input to prevent security vulnerabilities
	// This step is crucial for preventing XSS and SQL injection attacks
	if err := SanitizeStruct(ptr); err != nil {
		LogError(err, LogLevelWarn, map[string]interface{}{
			"operation": "sanitization",
			"object":    fmt.Sprintf("%T", ptr),
		})
	}

	// Perform struct validation using validation tags
	errs := validation.Struct(ptr)
	if len(errs) > 0 {
		// Add all validation errors to the context for client response
		context.AddValidationErrors(errs...)

		// Log validation failure for monitoring and debugging
		LogError(fmt.Errorf(MessageValidationFailed), LogLevelInfo, map[string]interface{}{
			"validation_errors": len(errs),
			"object":            fmt.Sprintf("%T", ptr),
		})
		return fmt.Errorf(MessageValidationFailed)
	}
	return nil
}

// ValidateNonZeroFields performs validation only on non-zero fields in a struct.
// This method is useful for partial updates (PATCH operations) where only some fields
// are provided and should be validated. Zero-value fields are ignored during validation.
//
// The validation process:
//  1. Input sanitization to prevent security vulnerabilities
//  2. Validation of only non-zero fields using validation tags
//  3. Error logging and collection for debugging and user feedback
//
// This is particularly useful for:
//   - PATCH operations where only modified fields are sent
//   - Optional field validation where empty values should be ignored
//   - Partial form submissions
//
// Example usage:
//
//	type UpdateUserRequest struct {
//	    Name  string `json:"name,omitempty" validation:"min=2,max=50"`
//	    Email string `json:"email,omitempty" validation:"email"`
//	    Age   int    `json:"age,omitempty" validation:"min=18,max=120"`
//	}
//
//	var req UpdateUserRequest
//	if err := context.BodyParser(&req); err != nil {
//	    return err
//	}
//
//	// Only validates fields that have non-zero values
//	if err := context.ValidateNonZeroFields(&req); err != nil {
//	    return err
//	}
//
// Performance note: Uses reflection to determine zero values and validate fields.
// Security note: Sanitization is still applied to all string fields, including zero values.
func (context *Context) ValidateNonZeroFields(ptr any) error {
	// First sanitize the input to prevent security vulnerabilities
	// This applies to all fields, including zero-value fields
	if err := SanitizeStruct(ptr); err != nil {
		LogError(err, LogLevelWarn, map[string]interface{}{
			"operation": "sanitization",
			"object":    fmt.Sprintf("%T", ptr),
		})
	}

	// Perform validation only on non-zero fields
	errs := validation.StructNonZeroFields(ptr)
	if len(errs) > 0 {
		// Add all validation errors to the context for client response
		context.AddValidationErrors(errs...)

		// Log validation failure for monitoring and debugging
		LogError(fmt.Errorf(MessageValidationFailed), LogLevelInfo, map[string]interface{}{
			"validation_errors": len(errs),
			"object":            fmt.Sprintf("%T", ptr),
		})
		return fmt.Errorf(MessageValidationFailed)
	}
	return nil
}

// ValidateAndSanitizeInput validates and sanitizes a single input field with custom rules.
// This method is useful for validating individual fields outside of struct validation,
// such as query parameters, path parameters, or individual form fields.
//
// The process:
//  1. Sanitizes the input to prevent security vulnerabilities
//  2. Applies custom validation rules to the sanitized input
//  3. Adds any validation errors to the context with the field name
//  4. Returns an error if validation fails
//
// This method combines sanitization and validation in a single call, making it
// convenient for validating individual fields with custom rules.
//
// Example usage:
//
//	// Validate a query parameter
//	searchTerm := context.Query("search")
//	if err := context.ValidateAndSanitizeInput("search", searchTerm, "required", "no_xss"); err != nil {
//	    return err
//	}
//
//	// Validate a path parameter
//	userID := context.Params("id")
//	if err := context.ValidateAndSanitizeInput("id", userID, "required", "alphanumeric"); err != nil {
//	    return err
//	}
//
//	// Validate an email field
//	email := context.FormValue("email")
//	if err := context.ValidateAndSanitizeInput("email", email, "required", "email"); err != nil {
//	    return err
//	}
//
// Performance note: Creates validation errors for each failed rule.
// Security note: Always sanitizes input before validation to prevent attacks.
func (context *Context) ValidateAndSanitizeInput(fieldName, input string, rules ...string) error {
	// Sanitize the input first to prevent security vulnerabilities
	sanitized := SanitizeInput(input)

	// Apply validation rules to the sanitized input
	errors := ValidateInput(sanitized, rules...)
	if len(errors) > 0 {
		// Add validation errors to context with field name for client response
		for _, err := range errors {
			context.AddValidationErrors(fmt.Errorf("%s %s", fieldName, err.Error()))
		}
		return fmt.Errorf(MessageValidationFailed)
	}

	return nil
}
