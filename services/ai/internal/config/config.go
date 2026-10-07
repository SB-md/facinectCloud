package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv      string
	HTTPAddr    string
	ServiceKey  string
	CORSOrigins string
	GeminiAPIKey string
	GeminiModel  string
}

func FromEnv() Config {
	return Config{
		AppEnv:       getenv("APP_ENV", "local"),
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		ServiceKey:   strings.TrimSpace(os.Getenv("AI_SERVICE_KEY")),
		CORSOrigins:  getenv("CORS_ORIGINS", "*"),
		GeminiAPIKey: strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:  getenv("GEMINI_MODEL", "gemini-2.0-flash"),
	}
}

func (c Config) IsLocal() bool {
	return c.AppEnv == "local" || c.AppEnv == "development"
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
