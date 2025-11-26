package model

import "time"

type CartItem struct {
	ID        uint   `gorm:"primaryKey"`
	UserEmail string `gorm:"index;not null"`
	ProductID uint   `gorm:"not null"`
	Qty       int    `gorm:"not null;default:1"`

	Product Product `gorm:"foreignKey:ProductID;references:ID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CartItem) TableName() string {
	return "cart_items"
}
