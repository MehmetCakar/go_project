package app

import (
	"os"
)

type Config struct {
	Port         string
	DSN          string
	JWTSecret    string
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	CookieDomain string
	CookieSecure bool
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func LoadFromEnv() Config {
	cfg := Config{}

	cfg.Port = firstNonEmpty(os.Getenv("PORT"), "8080")

	cfg.DSN = firstNonEmpty(os.Getenv("DB_DSN"), os.Getenv("DATABASE_URL"))
	if cfg.DSN == "" {
		// Legacy PG* fallbacks
		host := firstNonEmpty(os.Getenv("PGHOST"), "127.0.0.1")
		port := firstNonEmpty(os.Getenv("PGPORT"), "5432")
		user := firstNonEmpty(os.Getenv("PGUSER"), "postgres")
		pass := firstNonEmpty(os.Getenv("PGPASSWORD"), "postgres")
		name := firstNonEmpty(os.Getenv("PGDATABASE"), "postgres")
		ssl := firstNonEmpty(os.Getenv("PGSSLMODE"), "disable")
		cfg.DSN = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=" + ssl
	}

	cfg.JWTSecret = firstNonEmpty(os.Getenv("JWT_SECRET"), "CHANGE_ME_LONG_RANDOM")

	cfg.SMTPHost = os.Getenv("SMTP_HOST")
	cfg.SMTPPort = firstNonEmpty(os.Getenv("SMTP_PORT"), "587")
	cfg.SMTPUsername = firstNonEmpty(os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_USER"))
	cfg.SMTPPassword = firstNonEmpty(os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_PASS"))
	cfg.SMTPFrom = firstNonEmpty(os.Getenv("SMTP_FROM"), "noreply@localhost")
	cfg.SMTPFromName = firstNonEmpty(os.Getenv("SMTP_FROM_NAME"), "App")

	cfg.CookieDomain = os.Getenv("APP_COOKIE_DOMAIN")
	cfg.CookieSecure = os.Getenv("APP_COOKIE_SECURE") == "1"

	return cfg
}
