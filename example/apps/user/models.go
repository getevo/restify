package user

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/model"
	"github.com/getevo/restify"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type User struct {
	UserID   int     `gorm:"column:user_id;primaryKey;autoIncrement" json:"id"`
	UUID     string  `gorm:"column:uuid;index;size:24" json:"UUID"`
	Username string  `gorm:"column:username;size:255;uniqueIndex" validation:"required" json:"username"`
	Password string  `gorm:"column:password;size:512;not null;DEFAULT:''" json:"password" `
	IsAdmin  bool    `gorm:"column:is_admin" json:"is_admin"`
	Name     string  `gorm:"column:name;size:255" validation:"required,alpha" json:"name"`
	Email    string  `gorm:"column:email;size:255;uniqueIndex" validation:"required,email" json:"email"`
	Orders   []Order `gorm:"foreignKey:UserID" json:"orders,omitempty"`
	model.CreatedAt
	model.UpdatedAt
	model.DeletedAt // enables soft delete functionality
	restify.API
}

func (*User) TableName() string {
	return "user"
}

func (user *User) OnBeforeCreate(context *restify.Context) error {
	user.UUID = uuid.New().String()
	user.Password = GetMD5Hash(user.Password)
	return nil
}

func (user *User) OnBeforeUpdate(context *restify.Context) error {
	fmt.Println("here")
	if user.Password != "" {
		user.Password = GetMD5Hash(user.Password)
	}
	return nil
}

type Order struct {
	RowID     int      `gorm:"column:row_id;primaryKey;autoIncrement" json:"row_id"`
	OrderID   int      `gorm:"column:order_id;index;uniqueIndex:unq" json:"order_id"`
	UserID    int      `gorm:"column:user_id;fk:user;uniqueIndex:unq" json:"user_id"`
	ProductID int      `gorm:"column:product_id;fk:product;uniqueIndex:unq" json:"product_id"`
	Product   *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity  int      `gorm:"column:quantity" json:"quantity"`
	Total     int      `gorm:"column:total"  json:"total"`
	model.CreatedAt
	model.UpdatedAt
	restify.API
	restify.DisableCreate
	restify.DisableUpdate
	restify.DisableDelete
}

func (Order) TableName() string {
	return "order"
}

type Product struct {
	ProductID int    `gorm:"column:product_id;primaryKey;autoIncrement" json:"product_id"`
	Name      string `gorm:"column:name;size:255" json:"name"`
	UnitPrice int    `gorm:"column:unit_price" json:"unit_price"`
	model.CreatedAt
	model.UpdatedAt
	model.DeletedAt
	restify.API
	restify.DisableSet
}

func (Product) TableName() string {
	return "product"
}

type Article struct {
	ArticleID int    `gorm:"column:article_id;primaryKey;autoIncrement" json:"article_id"`
	UserID    int    `gorm:"column:user_id;fk:user" json:"user_id"`
	Body      string `gorm:"column:body;type:text"  json:"body"`
	model.CreatedAt
	model.UpdatedAt
	model.DeletedAt
	restify.API
}

func (*Article) TableName() string {
	return "article"
}

func (article *Article) RestPermission(permissions restify.Permissions, context *restify.Context) bool {

	var user, err = GetUser(context.Request) // retrieve user from basic auth

	//Dont let user do anything if the user is not logged in
	if err != nil {
		context.Error(err, http.StatusUnauthorized)
		return false
	}

	// enable delete only for admin users
	if !user.IsAdmin && permissions.Has("DELETE") {
		return false
	}

	// automatically set user_id in context for VIEW, UPDATE, DELETE, BATCH operations to the current
	if permissions.Has("VIEW", "UPDATE", "DELETE", "BATCH") {
		context.SetCondition("user_id", "=", user.UserID)
	}

	// override user_id in context for CREATE, UPDATE, DELETE,SET operations to the current user only
	if permissions.Has("CREATE", "UPDATE", "DELETE", "SET") {
		context.Override(Article{
			UserID: user.UserID,
		})
	}
	return true
}

func GetUser(request *evo.Request) (*User, error) {
	// Get the value of the "Authorization" header
	auth := request.Header("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("no Authorization header provided")
	}

	// Check if the authorization method is "Basic"
	if !strings.HasPrefix(auth, "Basic ") {
		return nil, fmt.Errorf("authorization header is not Basic")
	}

	// Decode the base64 encoded username:password
	decoded, err := base64.StdEncoding.DecodeString(auth[len("Basic "):])
	if err != nil {
		return nil, fmt.Errorf("failed to decode Authorization header: %v", err)
	}

	// Split the decoded string into username and password
	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return nil, fmt.Errorf("invalid Basic Auth format")
	}

	username := credentials[0]
	password := credentials[1]
	passwordHash := GetMD5Hash(password)
	var user User
	if db.Where("username = ? AND password = ?", username, passwordHash).Take(&user).RowsAffected == 0 {
		return nil, fmt.Errorf("invalid username or password")
	}
	return &user, nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
