package app

import "os"

type Config struct {
	Env, Port string
}	


func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

func LoadConfig() Config {
	return Config{
		Env:  getEnv("APP_ENV", "dev"),
		Port: getEnv("APP_PORT", "8080"),
	}
}
