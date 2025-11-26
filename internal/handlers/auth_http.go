package handlers

import (
	"encoding/json"
	"net/http"
	"time"
//	"github.com/gin-gonic/gin"
	"example.com/ecom-go/internal/service"
)

type AuthHTTP struct {
	S service.AuthService
}

type loginReq struct {
    Email    string `json:"email" binding:"required"`
    Password string `json:"password" binding:"required"`
}


func NewAuthHTTP(s service.AuthService) *AuthHTTP { return &AuthHTTP{S: s} }

type jsonMap map[string]any

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *AuthHTTP) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Password2 string `json:"password2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, jsonMap{"error":"Geçersiz JSON"}); return
	}
	if in.Password != in.Password2 {
		writeJSON(w, 400, jsonMap{"error":"Şifreler aynı değil"}); return
	}
	if err := h.S.Register(in.Email, in.Password); err != nil {
		// exists durumlarını kullanıcı dostu döndür
		if err == service.ErrExistsUnverified {
			writeJSON(w, 200, jsonMap{"ok": true, "info":"Doğrulama kodu tekrar gönderildi"}); return
		}
		if err == service.ErrExistsVerified {
			writeJSON(w, 409, jsonMap{"error":"Bu e-posta zaten kayıtlı"}); return
		}
		writeJSON(w, 500, jsonMap{"error":"Kayıt başarısız"}); return
	}
	writeJSON(w, 200, jsonMap{"ok": true})
}

func (h *AuthHTTP) Verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, jsonMap{"error":"Geçersiz JSON"}); return
	}
	if err := h.S.VerifyCode(in.Email, in.Code); err != nil {
		writeJSON(w, 400, jsonMap{"error": err.Error()}); return
	}
	writeJSON(w, 200, jsonMap{"ok": true})
}

func (h *AuthHTTP) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, jsonMap{"error":"Geçersiz JSON"}); return
	}

	// 👉 AuthService.Login JWT üretir
	token, err := h.S.Login(in.Email, in.Password)
	if err != nil {
		writeJSON(w, 401, jsonMap{"error":"E-posta/şifre hatalı veya doğrulanmamış hesap"}); return
	}

	// 👉 BURASI: JWT’yi HttpOnly cookie olarak yaz
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true, // HTTPS var (nginx ile)
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	writeJSON(w, 200, jsonMap{"ok": true})
}

func (h *AuthHTTP) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		MaxAge:   -1, // sil
	})
	writeJSON(w, 200, jsonMap{"ok": true})
}

func (h *AuthHTTP) Me(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("auth")
	if err != nil || c.Value == "" {
		writeJSON(w, 401, jsonMap{"error":"Giriş gerekli"}); return
	}
	uid, err := h.S.ParseToken(c.Value)
	if err != nil || uid == 0 {
		writeJSON(w, 401, jsonMap{"error":"Giriş gerekli"}); return
	}
	// İstersen burada DB’den e-posta da çekip döndürebilirsin
	writeJSON(w, 200, jsonMap{"id": uid})


}
func (h *AuthHTTP) Resend(w http.ResponseWriter, r *http.Request) {
    var in struct{ Email string `json:"email"` }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" {
        writeJSON(w, 400, jsonMap{"error": "Geçersiz JSON"}); return
    }
    // Kullanıcı yoksa bile service sessiz döner (enumeration engeli)
    if err := h.S.ResendCode(in.Email); err != nil {
        writeJSON(w, 500, jsonMap{"error": "Gönderilemedi"}); return
    }
    writeJSON(w, 200, jsonMap{"ok": true})
}
