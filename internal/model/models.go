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
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex"`
	Password  string
	Verified  bool
	// ---
	VerifyCode    string     // 6 haneli kod (dilersen hash’leyebilirsin)
	VerifyExpires *time.Time // son kullanma (örn. 15 dk)
	// timestamps …
}
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
