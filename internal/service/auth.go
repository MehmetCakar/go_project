package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
	"time"

	"example.com/ecom-go/internal/model"
)

var (
	//    ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotVerified = errors.New("email not verified")
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
	mailer    EmailSender
	codeTTL   time.Duration
}

func NewAuthService(db *gorm.DB, jwtSecret []byte, mailer EmailSender, codeTTL time.Duration) *AuthService {
	return &AuthService{db: db, jwtSecret: jwtSecret, mailer: mailer, codeTTL: codeTTL}
}

func randomCode() (string, error) {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	code := 100000 + (n % 900000)
	return fmt.Sprintf("%06d", code), nil
}

func (a *AuthService) Register(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		return errors.New("email and password required")
	}

	var u model.User
	err := a.db.WithContext(ctx).Where("email = ?", email).First(&u).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// yeni kullanc
		u = model.User{Email: email, Verified: false}
	case err != nil:
		return err
	default:
		// kullanc bulundu
		if u.Verified {
			return ErrEmailInUse
		}
		// verified=false ise devam edip kodu ve ifreyi gncelleyeceiz
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	code, err := randomCode()
	if err != nil {
		return err
	}
	exp := time.Now().Add(a.codeTTL)

	u.PasswordHash = string(hash)
	u.VerifyCode = code
	u.VerifyExpires = &exp
	u.Verified = false

	if u.ID == 0 {
		if err := a.db.WithContext(ctx).Create(&u).Error; err != nil {
			return err
		}
	} else {
		if err := a.db.WithContext(ctx).Model(&u).Updates(map[string]any{
			"password_hash":  u.PasswordHash,
			"verify_code":    u.VerifyCode,
			"verify_expires": u.VerifyExpires,
			"verified":       false,
		}).Error; err != nil {
			return err
		}
	}

	if a.mailer != nil {
		body := fmt.Sprintf("<p>Dorulama kodunuz: <b>%s</b></p>", code)
		if err := a.mailer.Send(email, "Dorulama Kodunuz", body); err != nil {
			return fmt.Errorf("send mail: %w", err)
		}
	}
	return nil
}
func (s *AuthService) HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, error) {
	var u model.User
	if err := s.db.WithContext(ctx).
		Where("email = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&u).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	// ifre dorulama
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Dorulanmam kullancya izin vermek istemiyorsanz:
	if !u.Verified {
		return nil, ErrNotVerified
	}

	return &u, nil
}

func (a *AuthService) Resend(ctx context.Context, email string) error {
	var u model.User
	if err := a.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user not found")
		}
		return err
	}
	if u.Verified {
		return ErrEmailInUse
	}

	code, err := randomCode()
	if err != nil {
		return err
	}
	exp := time.Now().Add(a.codeTTL)

	if err := a.db.WithContext(ctx).
		Model(&u).
		Updates(map[string]any{
			"verify_code":    code,
			"verify_expires": &exp,
		}).Error; err != nil {
		return err
	}

	if a.mailer != nil {
		body := fmt.Sprintf("<p>Dorulama kodunuz: <b>%s</b></p>", code)
		if err := a.mailer.Send(email, "Dorulama Kodunuz", body); err != nil {
			return fmt.Errorf("send mail: %w", err)
		}
	}
	return nil
}

func (a *AuthService) Verify(ctx context.Context, email, code string) error {
	var u model.User
	if err := a.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return err
	}
	if u.VerifyCode == "" || u.VerifyCode != code {
		return fmt.Errorf("invalid code")
	}
	if u.VerifyExpires != nil && time.Now().After(*u.VerifyExpires) {
		return fmt.Errorf("code expired")
	}
	return a.db.WithContext(ctx).Model(&u).Updates(map[string]any{
		"verified":       true,
		"verify_code":    "",
		"verify_expires": nil,
	}).Error
}

func (a *AuthService) IssueJWT(email string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": email,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(a.jwtSecret)
}

func (a *AuthService) ParseJWT(token string) (string, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("bad alg")
		}
		return a.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return "", fmt.Errorf("invalid token")
	}
	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
	}
	return "", fmt.Errorf("bad claims")
}

func (a *AuthService) RandomPassword() string {
	buf := make([]byte, 18)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}
