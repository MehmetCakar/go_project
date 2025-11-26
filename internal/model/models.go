package model

import "time"

// IMPORTANT: Uygulama kodu AutoMigrate/DDL ÇALIŞTIRMAZ.
// Var olan şemaya uygun alanlar.

type User struct {
	ID            uint       `gorm:"primaryKey"`
	Email         string     `gorm:"column:email"`
	PasswordHash  string     `gorm:"not null"`
	Verified      bool       `gorm:"column:verified"`
	VerifyCode    string     `gorm:"column:verify_code"`
	VerifyExpires *time.Time `gorm:"column:verify_expires"`
	CreatedAt     *time.Time `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func (User) TableName() string { return "users" }

type Product struct {
	ID         uint       `gorm:"primaryKey"`
	Name       string     `gorm:"column:name"`
	ImageURL   string     `gorm:"column:image_url"`
	PriceCents int64      `gorm:"column:price_cents"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (Product) TableName() string { return "products" }

type Order struct {
	ID         uint       `gorm:"primaryKey"`
	UserID     uint       `gorm:"column:user_id"`
	Status     string     `gorm:"column:status"`
	TotalCents int64      `gorm:"column:total_cents"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (Order) TableName() string { return "orders" }
