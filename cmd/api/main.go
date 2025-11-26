package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/ecom-go/internal/app"
	"example.com/ecom-go/internal/handlers"
	"example.com/ecom-go/internal/model"
	"example.com/ecom-go/internal/service"
)

func main() {
	// .env opsiyonel
	_ = godotenv.Load()

	cfg := app.LoadFromEnv()
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// --- DB ---
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	// SADECE el ile migrate yaptık; burada AutoMigrate ÇALIŞTIRMIYORUZ.
	// (cart_items tablosu zaten var.)

	// --- SMTP / Mail ---
	dialPort := 587
	if cfg.SMTPPort != "" {
		if p, err := strconv.Atoi(cfg.SMTPPort); err == nil {
			dialPort = p
		}
	}

	emailSender := service.NewSMTPSender(
		cfg.SMTPHost,
		dialPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
		cfg.SMTPFromName,
	)

	// --- Auth servis & HTTP ---
	authSvc := service.NewAuthService(db, []byte(cfg.JWTSecret), emailSender, 10*time.Minute)
	authHTTP := handlers.NewAuthHTTP(authSvc, cfg.CookieDomain, cfg.CookieSecure)

	r := gin.New()
	r.Use(gin.Recovery())

	// health
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/api/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// auth
	r.POST("/api/register", authHTTP.Register)
	r.POST("/api/verify", authHTTP.Verify)
	r.POST("/api/resend", authHTTP.Resend)
	r.POST("/api/login", authHTTP.Login)
	r.POST("/api/logout", authHTTP.Logout)
	r.GET("/api/me", authHTTP.Me)

	// Küçük helper: cookie'den JWT okuyup email döndür
	requireAuth := func(c *gin.Context) (string, bool) {
		tok, err := c.Cookie("access_token")
		if err != nil || tok == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return "", false
		}
		email, err := authSvc.ParseJWT(tok)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return "", false
		}
		return email, true
	}

	// --- Products ---
	r.GET("/api/products", func(c *gin.Context) {
		var items []model.Product
		if err := db.WithContext(c.Request.Context()).
			Order("id ASC").
			Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}
		c.JSON(http.StatusOK, items)
	})

	// ----- CART -----

	// GET /api/cart  → kullanıcının sepetini DB'den getirir
	r.GET("/api/cart", func(c *gin.Context) {
		email, ok := requireAuth(c)
		if !ok {
			return
		}

		var items []model.CartItem
		if err := db.WithContext(c.Request.Context()).
			Preload("Product").
			Where("user_email = ?", email).
			Order("id ASC").
			Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}

		c.JSON(http.StatusOK, items)
	})

	// POST /api/cart/add  → mevcut ürüne qty ekle (yoksa oluştur)
	r.POST("/api/cart/add", func(c *gin.Context) {
		email, ok := requireAuth(c)
		if !ok {
			return
		}

		var in struct {
			ProductID uint `json:"product_id"`
			Qty       int  `json:"qty"`
		}

		if err := c.ShouldBindJSON(&in); err != nil || in.ProductID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
			return
		}
		if in.Qty <= 0 {
			in.Qty = 1
		}

		var item model.CartItem
		result := db.WithContext(c.Request.Context()).
			Where("user_email = ? AND product_id = ?", email, in.ProductID).
			First(&item)

		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}

		if result.RowsAffected == 0 {
			item = model.CartItem{
				UserEmail: email,
				ProductID: in.ProductID,
				Qty:       in.Qty,
			}
			if err := db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
				return
			}
		} else {
			item.Qty += in.Qty
			if err := db.WithContext(c.Request.Context()).
				Model(&item).
				Update("qty", item.Qty).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// POST /api/cart/update → qty'yi direkt set et (0 veya altı → sil)
	r.POST("/api/cart/update", func(c *gin.Context) {
		email, ok := requireAuth(c)
		if !ok {
			return
		}

		var in struct {
			ProductID uint `json:"product_id"`
			Qty       int  `json:"qty"`
		}

		if err := c.ShouldBindJSON(&in); err != nil || in.ProductID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
			return
		}

		if in.Qty <= 0 {
			// sil
			if err := db.WithContext(c.Request.Context()).
				Where("user_email = ? AND product_id = ?", email, in.ProductID).
				Delete(&model.CartItem{}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		var item model.CartItem
		result := db.WithContext(c.Request.Context()).
			Where("user_email = ? AND product_id = ?", email, in.ProductID).
			First(&item)

		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}

		if result.RowsAffected == 0 {
			item = model.CartItem{
				UserEmail: email,
				ProductID: in.ProductID,
				Qty:       in.Qty,
			}
			if err := db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
				return
			}
		} else {
			item.Qty = in.Qty
			if err := db.WithContext(c.Request.Context()).
				Model(&item).
				Update("qty", item.Qty).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// ----- CHECKOUT -----
	// POST /api/checkout → toplamı hesapla, mail gönder, sepeti temizle
	r.POST("/api/checkout", func(c *gin.Context) {
		email, ok := requireAuth(c)
		if !ok {
			return
		}

		var items []model.CartItem
		if err := db.WithContext(c.Request.Context()).
			Preload("Product").
			Where("user_email = ?", email).
			Order("id ASC").
			Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}

		if len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cart is empty"})
			return
		}

		var total int64
		for _, it := range items {
			if it.Product.ID == 0 {
				continue
			}
			line := int64(it.Qty) * int64(it.Product.PriceCents)
			total += line
		}

		// Mail body (HTML)
		var sb strings.Builder
		fmt.Fprintf(&sb, "<p>Merhaba,</p>")
		fmt.Fprintf(&sb, "<p>Aşağıdaki ürünleri satın aldınız:</p><ul>")
		for _, it := range items {
			if it.Product.ID == 0 {
				continue
			}
			line := int64(it.Qty) * int64(it.Product.PriceCents)
			fmt.Fprintf(
				&sb,
				"<li>%d x %s — %.2f ₺</li>",
				it.Qty,
				it.Product.Name,
				float64(line)/100.0,
			)
		}
		fmt.Fprintf(&sb, "</ul><p>Toplam: <b>%.2f ₺</b></p>", float64(total)/100.0)
		fmt.Fprintf(&sb, "<p>Cakarokko'yu tercih ettiğiniz için teşekkürler.</p>")

		if emailSender != nil {
			if err := emailSender.Send(email, "Siparişiniz alındı", sb.String()); err != nil {
				log.Printf("checkout email send error: %v", err)
				// mail fail olsa bile kullanıcıya sipariş ok dönebilir
			}
		}

		// sepeti temizle
		if err := db.WithContext(c.Request.Context()).
			Where("user_email = ?", email).
			Delete(&model.CartItem{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":          true,
			"total_cents": total,
		})
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
