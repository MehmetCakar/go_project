package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/ecom-go/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHTTP struct {
	svc          *service.AuthService
	cookieDomain string
	cookieSecure bool
}

func NewAuthHTTP(svc *service.AuthService, domain string, secure bool) *AuthHTTP {
	return &AuthHTTP{
		svc:          svc,
		cookieDomain: domain,
		cookieSecure: secure,
	}
}

func (h *AuthHTTP) setCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetCookie(
		"access_token",
		token,
		int(ttl.Seconds()),
		"/",
		h.cookieDomain,
		h.cookieSecure,
		true,
	)
}

func (h *AuthHTTP) Register(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Email == "" || in.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email & password required"})
		return
	}

	if len(in.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
		return
	}

	if err := h.svc.Register(c, in.Email, in.Password); err != nil {
		if errors.Is(err, service.ErrEmailInUse) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHTTP) Verify(c *gin.Context) {
	var in struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email & code required"})
		return
	}

	if err := h.svc.Verify(c, in.Email, in.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHTTP) Resend(c *gin.Context) {
	var in struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}

	if err := h.svc.Resend(c, in.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHTTP) Login(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Email == "" || in.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email & password required"})
		return
	}

	_, err := h.svc.Login(c, in.Email, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		case errors.Is(err, service.ErrNotVerified):
			c.JSON(http.StatusConflict, gin.H{"error": "account not verified"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	tok, err := h.svc.IssueJWT(in.Email, 30*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}

	h.setCookie(c, tok, 30*24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHTTP) Logout(c *gin.Context) {
	h.setCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHTTP) Me(c *gin.Context) {
	tok, err := c.Cookie("access_token")
	if err != nil || tok == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no auth"})
		return
	}

	email, err := h.svc.ParseJWT(tok)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": email})
}
