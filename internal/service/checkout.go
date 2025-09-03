package service

import (
	"fmt"

	"gorm.io/gorm"

	"example.com/ecom-go/internal/model"
)

type CheckoutService interface {
	Checkout(userID uint) (model.Order, error)
}

type checkoutService struct{ db *gorm.DB; email EmailService }

func NewCheckoutService(db *gorm.DB, email EmailService) CheckoutService {
	return &checkoutService{db: db, email: email}
}

func (s *checkoutService) Checkout(userID uint) (model.Order, error) {
	// sepeti yükle
	var items []model.CartItem
	if err := s.db.Preload("Product").Where("user_id = ?", userID).Find(&items).Error; err != nil {
		return model.Order{}, err
	}
	if len(items) == 0 { return model.Order{}, fmt.Errorf("cart empty") }

	// sipariş oluştur
	var total int64
	var oitems []model.OrderItem
	for _, it := range items {
		total += it.Product.PriceCents * int64(it.Qty)
		oitems = append(oitems, model.OrderItem{
			ProductID:  it.ProductID,
			Name:       it.Product.Name,
			PriceCents: it.Product.PriceCents,
			Qty:        it.Qty,
		})
	}
	order := model.Order{UserID: userID, TotalCents: total}
	if err := s.db.Create(&order).Error; err != nil { return model.Order{}, err }
	for i := range oitems { oitems[i].OrderID = order.ID }
	if err := s.db.Create(&oitems).Error; err != nil { return model.Order{}, err }

	// sepeti temizle
	_ = s.db.Where("user_id = ?", userID).Delete(&model.CartItem{}).Error

	// mail (best-effort)
	var u model.User
	_ = s.db.First(&u, userID).Error
	_ = s.email.Send(u.Email, "Order confirmation",
		fmt.Sprintf("Thanks! Your order #%d total %.2f received.", order.ID, float64(order.TotalCents)/100.0))

	return order, nil
}
