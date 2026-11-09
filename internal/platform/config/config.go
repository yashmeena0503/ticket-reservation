package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	DatabaseUrl string
}

func Load() (Config, error) {
	database_url := os.Getenv("DATABASE_URL")
	if database_url == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return Config{
		Port:        getEnv("PORT", "3000"),
		DatabaseUrl: database_url,
	}, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return fallback
}
