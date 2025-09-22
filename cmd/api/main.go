package main

import (
	"log"

	"github.com/joho/godotenv"
	"example.com/ecom-go/internal/app"
        
)

func main() {
	err := godotenv.Load()
	if err != nil {
	    log.Println("No .env file found, using system environment variables")
	}

	_ = godotenv.Load() // .env varsa oku (yoksa sorun değil)
	cfg := app.LoadConfig()

	srv, cleanup, err := app.NewServer(cfg)
	if err != nil { log.Fatalf("init: %v", err) }
	defer cleanup()

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.Run(":" + cfg.Port); err != nil { log.Fatal(err) }
}
