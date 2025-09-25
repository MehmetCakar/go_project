package service

import (
	"strings"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"
	"errors"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"example.com/ecom-go/internal/model"
)

// Not: ErrExistsVerified ve ErrExistsUnverified ayrı errors.go dosyasında varsa
// buradakilerle ÇAKIŞMA olmasın. Eğer yoksa buradakileri kullan.
//var (
/*	ErrExistsVerified   = errors.New("exists-verified")
	ErrExistsUnverified = errors.New("exists-unverified")
)*/

type AuthService interface {
	Register(email, password string) error
	VerifyCode(email, code string) error
	ResendCode(email string) error
	Login(email, password string) (string, error)  // returns JWT
	ParseToken(token string) (uint, error)         // returns userID

	// Eski link tabanlı doğrulama için (kullanmayacağız ama interface kırılmasın)
	VerifyEmail(token string) error
}

type authService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) AuthService { return &authService{db: db} }

func jwtSecret() []byte { return []byte(os.Getenv("JWT_SECRET")) }

// İhtiyaç halinde public base döndürür (şu an kod akışında kullanılmıyor)
func publicBase() string {
	if v := os.Getenv("APP_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("PUBLIC_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// ---------------------------------------------------
// Yardımcılar
// ---------------------------------------------------

// 6 haneli doğrulama kodu (000000–999999), kriptografik rastgele
func gen6() (string, error) {
	max := big.NewInt(1000000) // 0..999999
	n, err := crand.Int(crand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Kullanıcıya yeni kod üretir, DB'ye yazar ve e-posta gönderir.
func (a *authService) generateAndSendCode(u *model.User) error {
	code, err := gen6()
	if err != nil {
		return err
	}
	expires := time.Now().Add(15 * time.Minute)

	// DB'de kodu/süreyi güncelle
	if err := a.db.Model(&model.User{}).
		Where("id = ?", u.ID).
		Updates(map[string]interface{}{
			"verify_code":    code,
			"verify_expires": expires,
		}).Error; err != nil {
		return err
	}

	// Spam-dostu sade içerik
	subject := "E-posta Doğrulama Kodun"
	html := fmt.Sprintf(`
<!doctype html>
<html><body style="font-family:Arial,sans-serif">
  <h2>Doğrulama Kodun</h2>
  <p>Merhaba,</p>
  <p>Aşağıdaki 6 haneli kodu 15 dakika içinde sitedeki doğrulama kutusuna gir:</p>
  <div style="font-size:28px;font-weight:700;letter-spacing:4px;margin:16px 0">%s</div>
  <p>Bu işlemi siz başlatmadıysanız, e-postayı yok sayabilirsiniz.</p>
  <hr>
  <p style="color:#888;font-size:12px">Bu e-posta <a href="https://cakarokko.com">cakarokko.com</a> tarafından gönderildi.</p>
</body></html>`, code)

	if err := NewEmailService().Send(u.Email, subject, html); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------
// Register
// ---------------------------------------------------
func (a *authService) Register(email, password string) error {
	var existed model.User
	err := a.db.Where("email = ?", email).First(&existed).Error

	if err == nil {
		// kullanıcı var
		if !existed.Verified {
			// yeni kod üret ve gönder; hata loglanır, 200 döneriz
			if err := a.generateAndSendCode(&existed); err != nil {
				log.Printf("Kod maili gönderilemedi (yeniden): %v", err)
			}
			return ErrExistsUnverified
		}
		return ErrExistsVerified
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// yeni kullanıcı oluştur
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u := model.User{
		Email:    email,
		Password: string(hash),
		Verified: false,
	}
	if err := a.db.Create(&u).Error; err != nil {
		return err
	}

	// ilk doğrulama kodunu gönder (hata olursa logla ve yut)
	if err := a.generateAndSendCode(&u); err != nil {
		log.Printf("Kod maili gönderilemedi: %v", err)
	}
	return nil
}

// ---------------------------------------------------
// VerifyCode
// ---------------------------------------------------
func (a *authService) VerifyCode(email, code string) error {
	code = strings.TrimSpace(code)

	var u model.User
	if err := a.db.Where("email = ?", email).First(&u).Error; err != nil {
		return err
	}
	if u.Verified {
		return nil // zaten doğrulanmış
	}
	if u.VerifyCode == "" || u.VerifyExpires == nil {
		return errors.New("no active code")
	}
	if time.Now().After(*u.VerifyExpires) {
		return errors.New("code expired")
	}
	if code != u.VerifyCode {
		return errors.New("invalid code")
	}

	// doğrulandı → kodları temizle
	if err := a.db.Model(&model.User{}).
		Where("id = ?", u.ID).
		Updates(map[string]any{
			"verified":       true,
			"verify_code":    nil,
			"verify_expires": nil,
		}).Error; err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------
// ResendCode
// ---------------------------------------------------
func (a *authService) ResendCode(email string) error {
	var u model.User
	if err := a.db.Where("email = ?", email).First(&u).Error; err != nil {
		// enumeration engelle: kullanıcı yoksa sessizce dön
		return nil
	}
	if u.Verified {
		return nil
	}
	return a.generateAndSendCode(&u)
}

// ---------------------------------------------------
// (Legacy) VerifyEmail by JWT — artık kullanılmıyor ama interface dursun
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
