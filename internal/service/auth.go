package service

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"example.com/ecom-go/internal/model"
)

type AuthService interface {
	Register(email, password string) error
	VerifyEmail(token string) error
	Login(email, password string) (string, error) // returns JWT
	ParseToken(token string) (uint, error)        // userID
}

type authService struct{
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) AuthService { return &authService{db: db} }

func jwtSecret() []byte { return []byte(os.Getenv("JWT_SECRET")) }
func publicBase() string {
	if v := os.Getenv("PUBLIC_BASE_URL"); v != "" { return v }
	return "http://localhost:8080"
}

func (a *authService) Register(email, password string) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u := model.User{Email: email, Password: string(hash), Verified:false}
	if err := a.db.Create(&u).Error; err != nil { return err }

	// verify token (24h)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID, "typ":"verify", "exp": time.Now().Add(24*time.Hour).Unix(),
	})
	token, err := t.SignedString(jwtSecret()); if err != nil { return err }
	link := fmt.Sprintf("%s/api/auth/verify?token=%s", publicBase(), url.QueryEscape(token))
	return NewEmailService().Send(u.Email, "Verify your account",
		"Hi,\nPlease verify your account:\n"+link)
}

func (a *authService) VerifyEmail(token string) error {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token)(interface{},error){ return jwtSecret(), nil })
	if err != nil { return err }
	if claims["typ"] != "verify" { return errors.New("invalid token type") }
	id, ok := claims["sub"].(float64); if !ok { return errors.New("invalid sub") }
	return a.db.Model(&model.User{}).Where("id = ?", uint(id)).Update("verified", true).Error
}

func (a *authService) Login(email, password string) (string, error) {
	var u model.User
	if err := a.db.Where("email = ?", email).First(&u).Error; err != nil { return "", err }
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil { return "", err }
	if !u.Verified { return "", errors.New("email not verified") }
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID, "typ":"session", "exp": time.Now().Add(7*24*time.Hour).Unix(),
	})
	return t.SignedString(jwtSecret())
}

func (a *authService) ParseToken(token string) (uint, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token)(interface{},error){ return jwtSecret(), nil })
	if err != nil { return 0, err }
	if claims["typ"] != "session" { return 0, errors.New("invalid token type") }
	id, ok := claims["sub"].(float64); if !ok { return 0, errors.New("invalid sub") }
	return uint(id), nil
}
