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
	ParseToken(token string) (uint, error)        // returns userID
}

type authService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) AuthService { return &authService{db: db} }

func jwtSecret() []byte { return []byte(os.Getenv("JWT_SECRET")) }

// Kamuya açık base URL (Nginx ile yayınlanan host)
func publicBase() string {
	if v := os.Getenv("PUBLIC_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// ---------------------------------------------------
// Register
// ---------------------------------------------------

func (a *authService) Register(email, password string) error {
	// Mail zaten var mı?
	var existed model.User
	err := a.db.Where("email = ?", email).First(&existed).Error
	if err == nil {
		// Kullanıcı zaten var
		if !existed.Verified {
			// Doğrulanmamış ise tekrar doğrulama maili gönder
			if err := a.sendVerifyMail(existed.ID, existed.Email); err != nil {
				return err
			}
			// Burada özel bir hata döndürebilirsiniz; şimdilik bilgi amaçlı error
			return errors.New("email already exists but not verified — verification re-sent")
		}
		return errors.New("email already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Yeni kullanıcı
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u := model.User{
		Email:    email,
		Password: string(hash),
		Verified: false,
	}
	if err := a.db.Create(&u).Error; err != nil {
		return err
	}

	// Doğrulama maili gönder
	return a.sendVerifyMail(u.ID, u.Email)
}

func (a *authService) sendVerifyMail(userID uint, to string) error {
	// 24 saatlik verify token
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"typ": "verify",
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	token, err := t.SignedString(jwtSecret())
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/api/auth/verify?token=%s", publicBase(), url.QueryEscape(token))
	body := "Merhaba,\n\nHesabını doğrulamak için aşağıdaki bağlantıya tıkla:\n" + link + "\n\nTeşekkürler."

	return NewEmailService().Send(to, "Hesabını Doğrula", body)
}

// ---------------------------------------------------
// VerifyEmail
// ---------------------------------------------------

func (a *authService) VerifyEmail(token string) error {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return err
	}
	if claims["typ"] != "verify" {
		return errors.New("invalid token type")
	}
	idFloat, ok := claims["sub"].(float64)
	if !ok {
		return errors.New("invalid sub")
	}
	return a.db.Model(&model.User{}).
		Where("id = ?", uint(idFloat)).
		Update("verified", true).Error
}

// ---------------------------------------------------
// Login
// ---------------------------------------------------

func (a *authService) Login(email, password string) (string, error) {
	var u model.User
	if err := a.db.Where("email = ?", email).First(&u).Error; err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}
	if !u.Verified {
		return "", errors.New("email not verified")
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID,
		"typ": "session",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return t.SignedString(jwtSecret())
}

// ---------------------------------------------------
// ParseToken
// ---------------------------------------------------

func (a *authService) ParseToken(token string) (uint, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return 0, err
	}
	if claims["typ"] != "session" {
		return 0, errors.New("invalid token type")
	}
	idFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid sub")
	}
	return uint(idFloat), nil
}
