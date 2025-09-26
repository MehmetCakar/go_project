package model
import "time"

type Product struct {
  ID         uint      `gorm:"primaryKey"`
  Name       string
  ImageURL   string
  PriceCents int64
  CreatedAt  time.Time
  UpdatedAt  time.Time
}


type User struct {
	ID               uint       `gorm:"primaryKey"`
	Email            string     `gorm:"uniqueIndex;not null"`
	Password         string     `gorm:"column:password"`        // İstersen tutma ama map’li olsun
	PasswordHash     string     `gorm:"column:password_hash"`   // BUNA bakacağız
	Verified         bool       `gorm:"column:verified;not null;default:false"`
	VerifiedAt       *time.Time `gorm:"column:verified_at"`
	VerifyCode       *string    `gorm:"column:verify_code"`
	VerifyExpiresAt  *time.Time `gorm:"column:verify_expires_at"`
}

func (User) TableName() string { return "users" }

type CartItem struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index"`
	ProductID uint
	Qty       int
	CreatedAt time.Time
	UpdatedAt time.Time
	Product   Product
}

type Order struct {
	ID         uint `gorm:"primaryKey"`
	UserID     uint `gorm:"index"`
	TotalCents int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Items      []OrderItem
}

type OrderItem struct {
	ID         uint `gorm:"primaryKey"`
	OrderID    uint `gorm:"index"`
	ProductID  uint
	Name       string
	PriceCents int64
	Qty        int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
