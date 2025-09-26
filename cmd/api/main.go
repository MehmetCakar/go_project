package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/ecom-go/internal/app"
	"example.com/ecom-go/internal/handlers"
	"example.com/ecom-go/internal/service"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	// .env bir kez
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Uygulama config (Port vb.)
	cfg := app.LoadConfig() // cfg.Port kullanıyoruz

	// --- DB bağlan (ENV'den) ---
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getenv("PGHOST", "127.0.0.1")
		port := getenv("PGPORT", "5433")          // sen 5433 kullanıyordun
		user := getenv("PGUSER", "postgres")
		pass := getenv("PGPASSWORD", "postgres")
		name := getenv("PGDATABASE", "ecom")
		ssl  := getenv("PGSSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, pass, name, ssl)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	// --- Server ---
	srv, cleanup, err := app.NewServer(cfg) // *gin.Engine bekleniyor
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer cleanup()

	// --- Auth servis + HTTP handler ---
	authSvc := service.NewAuthService(db)
	authHTTP := handlers.NewAuthHTTP(authSvc)

	// --- ROUTES (gin) ---
	// Not: authHTTP.Register/… http.HandlerFunc imzasında; gin.WrapF ile kullanıyoruz.
	srv.POST("/api/register", gin.WrapF(authHTTP.Register))
	srv.POST("/api/verify",   gin.WrapF(authHTTP.Verify))
	srv.POST("/api/login",    gin.WrapF(authHTTP.Login))   // Login içinde HttpOnly cookie set edilir
	srv.POST("/api/logout",   gin.WrapF(authHTTP.Logout))
	srv.GET ("/api/me",       gin.WrapF(authHTTP.Me))
	srv.POST("/api/resend", gin.WrapF(authHTTP.Resend))
	// (opsiyonel) yeniden kod gönder
	// srv.POST("/api/resend",   gin.WrapF(authHTTP.Resend))

	// Sağlık kontrolü
	srv.GET("/api/ping", func(c *gin.Context) { c.String(200, "ok") })

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
