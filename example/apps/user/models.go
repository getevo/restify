package user

import (
	"github.com/getevo/evo/v2/lib/model"
	"github.com/getevo/restify"
	"github.com/google/uuid"
)

type User struct {
	UserID   int     `gorm:"column:user_id;primaryKey;autoIncrement" json:"id"`
	UUID     string  `gorm:"column:uuid;index;size:24" json:"UUID"`
	Username string  `gorm:"column:username;size:255;uniqueIndex" validation:"required" json:"username"`
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
