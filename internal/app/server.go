package app

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func NewServer(cfg Config) (*gin.Engine, func(), error) {
	// release ise Gin'in default writer’ını kıs
	if gin.Mode() == gin.ReleaseMode {
		gin.DisableConsoleColor()
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Basit access log
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	})

	// Güvenlik header'ları (CSP’yi çok katı yapma; inline’a ihtiyaç oldu)
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer-when-downgrade")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';")
	})

	// Statik dosyalar (varsa)
	if _, err := os.Stat("./web/assets"); err == nil {
		r.Static("/assets", "./web/assets")
	}
	if _, err := os.Stat("./web"); err == nil {
		r.StaticFile("/", "./web/index.html")
	}

	// SPA fallback: /api ve /assets dışını index.html’e düşür
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/assets/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if _, err := os.Stat("./web/index.html"); err == nil {
			c.File("./web/index.html")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	cleanup := func() {}

	return r, cleanup, nil
}
