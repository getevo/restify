# Restify

Restify is an application built for the [EVO Framework](https://github.com/getevo/evo) that allows you to generate RESTful APIs on the fly without the need to write any code. These APIs can be used for various purposes, including data management, data entry processes, dashboards, or communication with third-party applications.

## Features

- **No Code Required:** Automatically generate RESTful APIs without writing a single line of code.
- **Data Management:** Use the generated APIs for managing data in your applications.
- **Data Entry:** Facilitate data entry processes through RESTful endpoints.
- **Dashboard Integration:** Seamlessly integrate with dashboards for real-time data management.
- **Third-Party Communication:** Use the APIs to interact with external services or applications.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- EVO Framework: [Installation Guide](https://github.com/getevo/evo)
- Golang (version 1.22 or higher)

### Usage

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


### Endpoints

After registering a model, Restify will expose several default endpoints. These endpoints support both `application/x-www-form-urlencoded` and `application/json` body content types.

| **Endpoint**     | **Description**                                                         | **Example `curl` Command**                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
|------------------|-------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Model Info**   | Return model structure and all endpoints information                    | `bash curl --location --request GET '/admin/rest/:model'`                                                                                                                                                                                                                                                                                                                                                                                                                          |
| **Create**       | Create a new resource using the `PUT` method.                           | `bash curl --location --request PUT '/admin/rest/:model' --header 'Content-Type: application/json' --data '{"field1": "field 1 value", "field2": "field 2 value", "field3": 3}'`                                                                                                                                                                                                                                                                                                   |
| **Batch Create** | Create multiple resources using the `PUT` method.                       | `bash curl --location --request PUT '/admin/rest/:model/batch' --header 'Content-Type: application/json' --data '[{"field1": "field 1 value", "field2": "field 2 value", "field3": 3},{"field1": "field 1 value", "field2": "field 2 value", "field3": 3},...]'`                                                                                                                                                                                                                   |
| **List All**     | Retrieve a list of resources with optional filters and pagination.      | `bash curl --location --request GET '/admin/rest/:model/all?field1[contains]=value'`                                                                                                                                                                                                                                                                                                                                                                                               |
| **Paginate**     | Retrieve a paginated list of resources with optional filters.           | `bash curl --location --request GET '/admin/rest/:model/paginate?field1[contains]=value&page=2&size=20'`                                                                                                                                                                                                                                                                                                                                                                           |
| **Retrieve**     | Retrieve a specific resource by ID.                                     | `bash curl --location --request GET '/admin/rest/:model/{id}'`                                                                                                                                                                                                                                                                                                                                                                                                                     |
| **Update**       | Update an existing resource by ID using the `PATCH` method.             | `bash curl --location --request PATCH '/admin/rest/:model/{id}' --header 'Content-Type: application/json' --data '{"field1": "updated value"}'`                                                                                                                                                                                                                                                                                                                                    |
| **Batch Update** | Update multiple resources based on conditions using the `PATCH` method. | `bash curl --location --request PATCH '/admin/rest/:model/batch?field1[gte]=value' --header 'Content-Type: application/json' --data '{"field2": "updated value"}'`                                                                                                                                                                                                                                                                                                                 |
| **Set**          | The `Set` endpoint compares existing rows in the database with the user input based on given criteria. It automatically removes rows from the database that aren't included in the user's request and creates any missing ones.            | `bash curl --location --request POST '/admin/rest/:model/set?field1[eq]=value' --header 'Content-Type: application/json' --data '[{"field2": "v1"},{"field2": "v2"},...]'`                                                                                                                                      |
| **Delete**       | Delete a specific resource by ID.                                       | `bash curl --location --request DELETE '/admin/rest/:model/{id}'`                                                                                                                                                                                                                                                                                                                                                                                                                  |
| **Batch Delete** | Delete multiple resources based on conditions.                          | `bash curl --location --request DELETE '/admin/rest/:model/batch?field1[eq]=value&field2[isnull]'`                                                                                                                                                                                                                                                                                                                                                                                 |

### Notes
- By default, if no criteria are given to the `batch delete` and `set` endpoints, they return an `unsafe request` error to prevent unwanted data loss. If you want to bypass this error, you can pass `unsafe=1` in the query string.
- In case of  `batch update` and `set`, if `"return=1"` is added to the query string, it will return all affected rows.


#### Query Parameters Explanation

The query parameters in the URL can be used to filter and manipulate the database query. The format `field1[operator]=value` allows you to compare a database field using the specified operator. Multiple query parameters can be mixed to refine your search.

| **Operator**  | **Description**                            | **Example**                       |
|---------------|--------------------------------------------|-----------------------------------|
| `eq`          | Equals (`=`)                               | `field1[eq]=value`                |
| `neq`         | Not equal (`!=`)                           | `field1[neq]=value`               |
| `gt`          | Greater than (`>`)                         | `field1[gt]=value`                |
| `lt`          | Less than (`<`)                            | `field1[lt]=value`                |
| `gte`         | Greater than or equal to (`>=`)            | `field1[gte]=value`               |
| `lte`         | Less than or equal to (`<=`)               | `field1[lte]=value`               |
| `in`          | In list (`IN`)                             | `field1[in]=value1,value2,value3` |
| `contains`    | Contains (`LIKE`)                          | `field1[contains]=partial_value`  |
| `isnull`      | Is null (`IS NULL`)                        | `field1[isnull]=1`                |
| `notnull`     | Is not null (`IS NOT NULL`)                | `field1[notnull]=1`               |

For the pagination API, you can identify the page number using `page=n` and set the result size using `size=m`.

#### Loading Associations

For `all`, `pagination`, and `retrieve` endpoints, it is possible to load associations. Assuming this is our model:

```golang
type User struct {
   UserID   int    `gorm:"primaryKey;autoIncrement"`
   UserName string `gorm:"column:username;index;size:255"`
   Name     string `gorm:"column:name;size:255"`
   Orders   []Order
   restify.API  
}

type Order struct{
   OrderID   int    `gorm:"primaryKey;autoIncrement"`
   UserID    int    `gorm:"fk;users"`
   Title     string
   Price     int
   ProductID int    `gorm:"fk:product_id"`
   Product   Product
}

type Product struct{
   ProductID int    `gorm:"primaryKey;autoIncrement"`
   Title     string
   Price     int
}
```
- To load `Orders` association:
    ```bash
    curl --location --request GET '/rest/api/users/:id?associations=Orders'
    ```

- To load an association of an association (`Orders.Product`):
    ```bash
    curl --location --request GET '/rest/api/users/:id?associations=Orders.Product'
    ```

- To load all associations:
    ```bash
    curl --location --request GET '/rest/api/users/:id?associations=*'
    ```

#### Order

You may order the "all" and "pagination" responses by passing `order=field.asc|desc` in the query string.

**Example:**

```bash
curl --location --request GET '/admin/rest/:model/all?order=username.asc'
```
This example retrieves all records from the model and orders them by the UserName field in ascending order.

#### Offset and Limit
You may set a data offset and limit for the "all" API by passing offset=n and limit=m in the query string.
```bash
curl --location --request GET '/admin/rest/:model/all?offset=10&limit=20'
```
This example retrieves records starting from the 10th record and limits the response to 20 records.


#### Select Specific Fields
It is possible to select specific fields in the "all" and "pagination" APIs by passing a list of comma-separated fields like `fields=field1,field2`.
```bash
curl --location --request GET '/admin/rest/:model/all?fields=username,name'
```
This example retrieves only the username and name fields from all records in the model.


#### Additional Notes

- **Composite Primary Key:** In case the model has a composite primary key, the URL will become: `/admin/rest/{key1}/{key2}/...`

- **Model Name:** The `{model}` variable is equal to the table name of the model.

- **Model Information:** Using `GET /admin/rest/models`, it is possible to get information about all available models.



### Customization

#### Hooks

Using hooks, you can modify the data before or after database actions. It is also possible to validate or interrupt the action before it runs. Below is a list of available hooks and their descriptions:

| **Hook Name**    | **Description**                                | **Supported Endpoints** |
|------------------|------------------------------------------------|-------------------------|
| `OnBeforeCreate` | Runs before creating an object                 | Create/Batch Create/Set |
| `OnBeforeUpdate` | Runs before updating an object                 | Update                  |
| `OnBeforeDelete` | Runs before deleting an object                 | Delete/Set              |
| `ValidateCreate` | Validates before creating an object            | Create/Batch Create     |
| `ValidateUpdate` | Validates before updating an object            | Update/Batch Update     |
| `OnAfterCreate`  | Runs after creating an object                  | Create/Set              |
| `OnAfterUpdate`  | Runs after updating an object                  | Update                  |
| `OnAfterDelete`  | Runs after deleting an object                  | Delete/Set              |
| `OnAfterGet`     | Runs after loading an object from the database | All/Paginate/Update/Set |

### Warning

Manipulation of values in `OnBeforeCreate` and `ValidateCreate` could cause the `set` endpoint to break, as Restify may fail to compare user input with the new values set by these hooks.

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




#### Validation

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



#### Features

You may turn on or off some Restify endpoints. The following table outlines the available options:

| **Feature**             | **Description**                           |
|-------------------------|-------------------------------------------|
| `restify.API`           | Enable Restify Endpoint API               |
| `restify.DisableCreate` | Disable create and batch create endpoints |
| `restify.DisableUpdate` | Disable update and batch update endpoints |
| `restify.DisableSet`    | Disable set endpoint                      |
| `restify.DisableList`   | Disable data listing API endpoints        |
| `restify.DisableDelete` | Disable single and batch delete endpoints |

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


### Base Path

The Restify API base path is set to `"/admin/restify"` by default. However, you can modify this path before calling `WhenReady` by using the `restify.SetPrefix` method.

#### Example

```golang
func (App) Register() {
    restify.SetPrefix("/api/rest") // all Restify endpoints will start with /api/rest
}
```
