package app

import (
	"net/http"
	"os"
	"strings"
	"log"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"example.com/ecom-go/internal/model"
	"example.com/ecom-go/internal/service"
)

func NewServer(cfg Config) (*gin.Engine, func(), error) {
	// --- DB bağlan ---
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	// --- Migrations (burayı genişlettik) ---
	if err := db.AutoMigrate(
		&model.Product{},
		&model.User{},
		&model.CartItem{},
		&model.Order{},
		&model.OrderItem{},
	); err != nil {
		return nil, nil, err
	}

	// --- Gin ---
	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// sayfalar
	r.GET("/", func(c *gin.Context) { c.File("./web/index.html") })
	r.GET("/auth", func(c *gin.Context) { c.File("./web/auth.html") })
	r.GET("/cart", func(c *gin.Context) { c.File("./web/cart.html") })

	// statik dosyalar: /assets altında servis et
	r.Static("/assets", "./web/assets")

	// cache kapatma (debug)
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})

	// --- Servisler ---
	emailSvc := service.NewEmailService()
	auth := service.NewAuthService(db)
	cart := service.NewCartService(db)
	checkout := service.NewCheckoutService(db, emailSvc)

	// --- Public rotalar ---
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	// Ürünler
	r.GET("/api/products", func(c *gin.Context) {
		var ps []model.Product
		if err := db.Order("id asc").Find(&ps).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ps)
	})
	r.POST("/api/admin/seed", func(c *gin.Context) {
		// Order ve Cart tablosu da varsa onları da temizleyelim ve ID’leri sıfırlayalım
		db.Exec("TRUNCATE TABLE order_items, orders, cart_items, products RESTART IDENTITY CASCADE")

		data := []model.Product{
			{Name: "Blue T-Shirt", PriceCents: 1999, ImageURL: "https://picsum.photos/seed/blue/600/400"},
			{Name: "Red Hoodie", PriceCents: 4599, ImageURL: "https://picsum.photos/seed/red/600/400"},
			{Name: "Sneakers", PriceCents: 6999, ImageURL: "https://picsum.photos/seed/shoes/600/400"},
		}
		for _, p := range data {
			db.Create(&p)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// --- Auth ---

	r.POST("/api/auth/register", func(c *gin.Context) {
		        var req struct {
        		Email    string `json:"email"`
        		Password string `json:"password"`
    		}
    		if err := c.ShouldBindJSON(&req); err != nil {
    		    	c.JSON(400, gin.H{"error": "invalid payload"})
        		return
    		}

    		// 🔐 Kullanıcı kaydı dene
		err := auth.Register(req.Email, req.Password)

		switch {
		case err == nil || errors.Is(err, service.ErrExistsUnverified):
    			c.JSON(http.StatusOK, gin.H{"ok": true})
    			return
		case errors.Is(err, service.ErrExistsVerified):
    			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
  			  return
		default:
    			log.Printf("register unexpected: %v", err)
    			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
    			return
		}

	})

	r.POST("/api/auth/login", func(c *gin.Context) {
		type reqBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		var req reqBody
		if err := c.BindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		tok, err := auth.Login(req.Email, req.Password)
		if err != nil {
			// Not: service.Login zaten Verified=false ise kabul etmiyor.
			// Dışarıya nedeni yansıtma (invalid creds de).
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		// Güvenli cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "session",
			Value:    tok,
			Path:     "/",
			MaxAge:   7 * 24 * 3600,
			HttpOnly: true,
			Secure:   true,                   // sadece HTTPS
			SameSite: http.SameSiteLaxMode,   // form login için ideal
		})
		// JSON cevabı (opsiyonel: token’ı döndürme)
		c.JSON(http.StatusOK, gin.H{
			"ok":         true,
			// "token":   tok,          // gerekliyse aç
			// "token_type": "Bearer",  // gerekliyse aç
		})
	})

	r.POST("/api/auth/logout", func(c *gin.Context) {
    		http.SetCookie(c.Writer, &http.Cookie{
    	    Name:     "session",
 	       Value:    "",
       		 Path:     "/",
      		  MaxAge:   -1,
     		   HttpOnly: true,
       			Secure:   true,
       			 SameSite: http.SameSiteLaxMode,
    		})
    		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// KOD DOĞRULAMA

	r.POST("/api/auth/verify-code", func(c *gin.Context) {
	    var req struct {
	        Email string `json:"email"`
        	Code  string `json:"code"`
    	    }
  	  if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
  	  }
 	   if err := auth.VerifyCode(req.Email, req.Code); err != nil {
        // istersen sabitle: "invalid or expired code"
        	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
       		 return
   		 }
  		  c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/api/auth/resend-code", func(c *gin.Context) {
    		var req struct{ Email string `json:"email"` }
    		if err := c.BindJSON(&req); err != nil || req.Email == "" {
    		    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
    			return
    		}
  	  _ = auth.ResendCode(req.Email) // enumeration önleme
 	   c.JSON(http.StatusOK, gin.H{"ok": true})
		})

	r.GET("/api/auth/verify", func(c *gin.Context) { 
    		t := c.Query("token")
   		 if t == "" {
        		c.JSON(400, gin.H{"error": "missing token"})
        		return
    		}
    		if err := auth.VerifyEmail(t); err != nil {
    		    c.JSON(400, gin.H{"error": err.Error()})
       			 return
   	 }
    	// ✅ Doğrulama başarılı → kullanıcıyı ana sayfaya yönlendir
   	 c.Redirect(http.StatusFound, "/")
})


	// --- Auth middleware (userID'yi context'e koyar) ---
	authMW := func(c *gin.Context) {
		var tok string
		if ah := c.GetHeader("Authorization"); strings.HasPrefix(ah, "Bearer ") {
			tok = strings.TrimPrefix(ah, "Bearer ")
		}
		if tok == "" {
			if v, err := c.Cookie("session"); err == nil {
				tok = v
			}
		}
		if tok == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "login required"})
			return
		}
		uid, err := auth.ParseToken(tok)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid session"})
			return
		}
		c.Set("userID", uid)
		c.Next()
	}

	// --- Sepet + Checkout ---
	r.POST("/api/cart/add", authMW, func(c *gin.Context) {
		var req struct {
			ProductID uint `json:"product_id"`
			Qty       int  `json:"qty"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}
		uid := c.GetUint("userID")
		if err := cart.Add(uid, req.ProductID, req.Qty); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	r.GET("/api/cart", authMW, func(c *gin.Context) {
		uid := c.GetUint("userID")
		items, err := cart.Get(uid)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, items)
	})

	r.POST("/api/checkout", authMW, func(c *gin.Context) {
		uid := c.GetUint("userID")
		order, err := checkout.Checkout(uid)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, order)
	})

	// --- cleanup ---
	cleanup := func() {
		if s, err := db.DB(); err == nil {
			_ = s.Close()
		}
	}
	// (opsiyonel SPA fallback)
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/assets/") {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.File("./web/index.html")
	})

	return r, cleanup, nil
}
