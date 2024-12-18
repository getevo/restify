# RESTify API Documentation for Developers

This guide provides comprehensive instructions for interacting with RESTify APIs. RESTify dynamically generates RESTful APIs for data management, making it easy to perform CRUD operations, filtering, sorting, pagination, and association handling.

---

## 1. Acquiring Model Information

The `/admin/rest/<model>` endpoint provides essential metadata about a model, including its fields, their types, relationships, and available operations.

### **Endpoint**
```bash
GET /admin/rest/<model>
```

### **Response**
```json
{
    "data": {
        "name": "Product",
        "id": "product",
        "fields": [
            { "label": "ProductID", "name": "product_id", "type": "int", "pk": true },
            { "label": "Name", "name": "name", "type": "string" },
            { "label": "UnitPrice", "name": "unit_price", "type": "int" },
            { "label": "CreatedAt", "name": "created_at", "type": "Time" },
            { "label": "UpdatedAt", "name": "updated_at", "type": "Time" },
            { "label": "Deleted", "name": "deleted", "type": "bool" },
            { "label": "DeletedAt", "name": "deleted_at", "type": "Time" }
        ],
        "endpoints": [
            { "name": "ModelInfo", "method": "GET", "url": "/admin/rest/product", "description": "Returns model metadata" },
            { "name": "All", "method": "GET", "url": "/admin/rest/product/all", "description": "Retrieve all objects" },
            { "name": "Paginate", "method": "GET", "url": "/admin/rest/product/paginate", "description": "Retrieve paginated data" },
            { "name": "Get", "method": "GET", "url": "/admin/rest/product/:product_id", "description": "Retrieve an object by ID" },
            { "name": "Create", "method": "PUT", "url": "/admin/rest/product", "description": "Create a new object" },
            { "name": "BatchCreate", "method": "PUT", "url": "/admin/rest/product/batch", "description": "Create multiple objects" },
            { "name": "Update", "method": "PATCH", "url": "/admin/rest/product/:product_id", "description": "Update an object by ID" },
            { "name": "BatchUpdate", "method": "PATCH", "url": "/admin/rest/product/batch", "description": "Update multiple objects" },
            { "name": "Delete", "method": "DELETE", "url": "/admin/rest/product/:product_id", "description": "Delete an object by ID" },
            { "name": "BatchDelete", "method": "DELETE", "url": "/admin/rest/product/batch", "description": "Delete multiple objects" }
        ]
    }
}
```

### **Explanation**
- **Fields**:
    - Each field includes:
        - `label`: Human-readable name of the field.
        - `name`: The field's technical name in the database.
        - `type`: Data type (`int`, `string`, `Time`, `bool`, `relation`).
        - `pk: true`: Indicates the field is a primary key.
    - **Relation Fields**:
        - Relation fields link models. For example, a `User` model may have a `Orders` relation to fetch associated orders.
        - Example:
          ```json
          { "label": "Orders", "name": "orders", "type": "relation" }
          ```
          This indicates the model has a relationship with another model, such as fetching all orders belonging to a user.

- **Endpoints**:
    - Describe available operations (CRUD, pagination, etc.) for the model.

- **Composite Primary Keys**:
    - If a model has composite primary keys, endpoints like `Get`, `Update`, and `Delete` use URLs like:
      ```
      /admin/rest/<model>/:pk1/:pk2
      ```
      The order of `pk1` and `pk2` matches the column order in the `fields` array.

---

## 2. Calling Endpoints with Examples

### **Retrieve All Records**
```bash
curl --location 'http://<your-server>/admin/rest/product/all'
```

### **Create a New Record**
```bash
curl --location --request PUT 'http://<your-server>/admin/rest/product' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Milk",
    "unit_price": 50
}'
```

### **Update a Record**
```bash
curl --location --request PATCH 'http://<your-server>/admin/rest/product/1' \
--header 'Content-Type: application/json' \
--data '{
    "unit_price": 55
}'
```

### **Delete a Record**
```bash
curl --location --request DELETE 'http://<your-server>/admin/rest/product/1'
```

### **Batch Create**
```bash
curl --location --request PUT 'http://<your-server>/admin/rest/product/batch' \
--header 'Content-Type: application/json' \
--data '[
    { "name": "Bread", "unit_price": 30 },
    { "name": "Eggs", "unit_price": 60 }
]'
```

### **Batch Update**
```bash
curl --location --request PATCH 'http://<your-server>/admin/rest/product/batch?unit_price[gt]=50' \
--header 'Content-Type: application/json' \
--data '{
    "unit_price": 100
}'
```

---

## 3. Pagination, Sorting, and Filtering

RESTify supports advanced querying to refine data retrieval.

### **Pagination**
```bash
curl --location 'http://<your-server>/admin/rest/product/paginate?page=2&size=10'
```

### **Sorting**
Sort by one or more fields:
```bash
curl --location 'http://<your-server>/admin/rest/product/all?order=name.asc,unit_price.desc'
```

### **Filtering**
RESTify provides flexible filters. Supported operators:

| **Operator**  | **Description**                | **Example**                        |
|---------------|--------------------------------|------------------------------------|
| `eq`          | Equals                        | `name[eq]=Milk`                   |
| `neq`         | Not Equals                    | `name[neq]=Milk`                  |
| `gt`          | Greater Than                  | `unit_price[gt]=50`               |
| `gte`         | Greater Than or Equal To      | `unit_price[gte]=50`              |
| `lt`          | Less Than                     | `unit_price[lt]=100`              |
| `lte`         | Less Than or Equal To         | `unit_price[lte]=100`             |
| `contains`    | Contains                      | `name[contains]=Mil`              |
| `in`          | In List                       | `name[in]=Milk,Bread`             |
| `notnull`     | Is Not Null                   | `deleted_at[notnull]=1`           |
| `isnull`      | Is Null                       | `deleted_at[isnull]=1`            |

### **Example**
```bash
curl --location 'http://<your-server>/admin/rest/product/all?unit_price[gte]=50&name[contains]=Milk'
```

---

## 4. Loading Associations

Load related data using the `associations` query parameter.

### **Single Association**
```bash
curl --location 'http://<your-server>/admin/rest/user/1?associations=Orders'
```

### **Multiple Associations**
```bash
curl --location 'http://<your-server>/admin/rest/user/1?associations=Orders,Payments'
```

### **Paginated Resources with Associations**
```bash
curl --location 'http://<your-server>/admin/rest/user/paginate?page=1&size=5&associations=Orders,Payments'
```

---

## 5. Authentication in RESTify

Backend may require authentication for API calls, depending on your application logic. Check with your backend team for specific authentication methods.

### **API Token**
Include the API key in the request header:
```bash
x-api-key: <your-api-key>
```

### **Bearer Token**
Include the token in the `Authorization` header:
```bash
Authorization: Bearer <token>
```

---

## Validation in RESTify

RESTify automatically validates input data based on the validation rules specified in the model's structure. If the input data fails validation, the API returns an error response with details about the failed validations.



### API Validation Error Responses

When a validation error occurs, the response includes a `validation_error` field containing details about the fields that failed validation and the specific error messages.

### Example: Validation Error for Create API

**Request**:
```bash
curl --location --request PUT 'http://127.0.0.1:8080/admin/rest/user' \
--header 'Content-Type: application/json' \
--data-raw '{
    "username": "",
    "name": "reza",
    "email": "reza@.dev"
}'
```

**Response**:
```json
{
    "data": 0,
    "total": 0,
    "offset": 0,
    "total_pages": 0,
    "current_page": 0,
    "size": 0,
    "success": false,
    "error": "validation failed",
    "type": "",
    "validation_error": [
        { "field": "username", "error": "is required" },
        { "field": "email", "error": "invalid email reza@.dev" }
    ]
}
```

---

### Validation Error Scenarios

### **Create API Validation**
If any field violates the validation rules during the creation of a single record, the API returns a `400 Bad Request` with the validation errors.

### **Batch Create API Validation**
When creating multiple records in a batch, the validation errors for each individual record are returned. The API proceeds with valid records and skips invalid ones.

**Request**:
```bash
curl --location --request PUT 'http://127.0.0.1:8080/admin/rest/user/batch' \
--header 'Content-Type: application/json' \
--data '[
    { "username": "user1", "name": "John", "email": "john@example.com" },
    { "username": "", "name": "Reza", "email": "reza@.dev" }
]'
```

**Response**:
```json
{
    "data": [
        { "username": "user1", "name": "John", "email": "john@example.com" }
    ],
    "success": false,
    "error": "validation failed for some records",
    "validation_error": [
        {
            "record": 2,
            "errors": [
                { "field": "username", "error": "is required" },
                { "field": "email", "error": "invalid email reza@.dev" }
            ]
        }
    ]
}
```

---

### **Update API Validation**
For update operations, the API ensures that the new values meet the validation criteria. Errors are returned for any invalid fields.

**Request**:
```bash
curl --location --request PATCH 'http://127.0.0.1:8080/admin/rest/user/1' \
--header 'Content-Type: application/json' \
--data-raw '{
    "email": "invalid-email"
}'
```

**Response**:
```json
{
    "data": 0,
    "success": false,
    "error": "validation failed",
    "validation_error": [
        { "field": "email", "error": "invalid email invalid-email" }
    ]
}
```

---

### **Batch Update API Validation**
Similar to batch create, validation errors are returned for each invalid record during batch updates.

**Request**:
```bash
curl --location --request PATCH 'http://127.0.0.1:8080/admin/rest/user/batch?name[eq]=John' \
--header 'Content-Type: application/json' \
--data-raw '{
    "email": "not-an-email"
}'
```

**Response**:
```json
{
    "data": 0,
    "success": false,
    "error": "validation failed for some records",
    "validation_error": [
        {
            "field": "email",
            "error": "invalid email not-an-email"
        }
    ]
}
```

---

## Built-In Validators and Error Messages

RESTify supports a wide range of built-in validators. Below is a table of possible validators and their corresponding error messages:

### **Non-Database Validators**

| Validator         | Description                                       | Example Usage              | Possible Error Message                  |
|-------------------|---------------------------------------------------|----------------------------|-----------------------------------------|
| `required`        | Value cannot be empty.                            | `validation:"required"`    | `is required`                          |
| `email`           | Checks for a valid email format.                  | `validation:"email"`       | `invalid email`                        |
| `regex(...)`      | Matches value against a custom regex.             | `validation:"regex(...)"`  | `format is not valid`                  |
| `len<, len>, ...` | Compares string length.                           | `validation:"len>5"`       | `is too long` / `is too short`         |
| `alpha`           | Only alphabetical characters allowed.             | `validation:"alpha"`       | `is not alpha`                         |
| `numeric`         | Only numeric values allowed.                      | `validation:"numeric"`     | `is not numeric`                       |
| `password(...)`   | Checks password complexity.                       | `validation:"password(...)"` | `password is not complex enough`   |

---

### **Database-Related Validators**

| Validator         | Description                                       | Example Usage              | Possible Error Message                  |
|-------------------|---------------------------------------------------|----------------------------|-----------------------------------------|
| `unique`          | Ensures the field value is unique in the table.   | `validation:"unique"`      | `duplicate entry`                      |
| `fk`              | Ensures the field references a valid foreign key. | `validation:"fk"`          | `value does not match foreign key`     |
| `enum`            | Ensures the value matches ENUM values in schema.  | `validation:"enum"`        | `invalid value, expected values are ...` |

---

## Handling Validation Errors in Frontend

1. **Display Validation Messages**:
    - Parse the `validation_error` array in the response.
    - Show user-friendly messages for each invalid field.

2. **Highlight Invalid Fields**:
    - Use the `field` name from the `validation_error` to identify and highlight the input.

3. **Retry with Valid Data**:
    - Correct the invalid fields and retry the API request.

---

By leveraging RESTify's validation capabilities, you can ensure robust data integrity and provide meaningful feedback to users during API interactions.

This documentation provides a comprehensive guide for interacting with RESTify APIs. For further customization, consult your backend team.
