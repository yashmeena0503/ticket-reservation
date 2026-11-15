package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	DatabaseUrl string
	JWTSecret   string
}

func Load() (Config, error) {
	database_url := os.Getenv("DATABASE_URL")
	if database_url == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return Config{}, errors.New("JWT_SECRET is required (minimum 32 characters)")
	}
	return Config{
		Port:        getEnv("PORT", "3000"),
		DatabaseUrl: database_url,
		JWTSecret:   secret,
	}, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return fallback
}
