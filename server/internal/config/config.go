package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	DataDir       string
	AllowedOrigin string
	AccessToken   string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() *Config {
	return &Config{
		Port:          env("PORT", "8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://bailian:bailian123@localhost:15432/bailian_studio?sslmode=disable"),
		DataDir:       env("DATA_DIR", "../data"),
		AllowedOrigin: env("ALLOWED_ORIGIN", "http://localhost:5173"),
		AccessToken:   env("ACCESS_TOKEN", ""),
	}
}
