package app

import (
	"net/http"
	"os"
	"strings"

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
  	  	if err := c.BindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
        	c.JSON(400, gin.H{"error": "bad json"})
        		return
    		}

    		if err := auth.Register(req.Email, req.Password); err != nil {
        		c.JSON(400, gin.H{"error": err.Error()})
        		return
    		}

    		// E-posta doğrulama linki gönderildi
    		c.JSON(200, gin.H{"ok": true})
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
 		// Doğrulama başarılı → anasayfaya / ürünler sayfasına
    		c.Redirect(http.StatusFound, "/")
	})

	r.POST("/api/auth/login", func(c *gin.Context) {
		var req struct{ Email, Password string }
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}
		tok, err := auth.Login(req.Email, req.Password)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		// cookie + JSON token
		c.SetCookie("session", tok, 7*24*3600, "/", "", true, true)
		c.JSON(200, gin.H{"ok": true, "token": tok, "token_type": "Bearer"})
	})

	r.POST("/api/auth/logout", func(c *gin.Context) {
		c.SetCookie("session", "", -1, "/", "", true, true)
		c.JSON(200, gin.H{"ok": true})
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
