package service

import (
	"errors"

	"gorm.io/gorm"

	"example.com/ecom-go/internal/model"
)

type CartService interface {
	Add(userID uint, productID uint, qty int) error
	Get(userID uint) ([]model.CartItem, error)
	Clear(userID uint) error
}

type cartService struct{ db *gorm.DB }

func NewCartService(db *gorm.DB) CartService { return &cartService{db: db} }

func (s *cartService) Add(userID uint, productID uint, qty int) error {
	if qty <= 0 { return errors.New("qty must be > 0") }

	var it model.CartItem
	err := s.db.Where("user_id = ? AND product_id = ?", userID, productID).First(&it).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		it = model.CartItem{UserID: userID, ProductID: productID, Qty: qty}
		return s.db.Create(&it).Error
	} else if err != nil {
		return err
	}
	it.Qty += qty
	return s.db.Save(&it).Error
}

func (s *cartService) Get(userID uint) ([]model.CartItem, error) {
	var items []model.CartItem
	return items, s.db.Preload("Product").Where("user_id = ?", userID).Order("id asc").Find(&items).Error
}

func (s *cartService) Clear(userID uint) error {
	return s.db.Where("user_id = ?", userID).Delete(&model.CartItem{}).Error
}
