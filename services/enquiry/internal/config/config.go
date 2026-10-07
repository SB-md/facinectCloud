package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv               string
	HTTPAddr             string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	ServiceKey           string
	CORSOrigins          string
	JWTPublicKeyPath     string
	JWTIssuer            string
	JWTAudience          string
	AIBaseURL            string
	AIServiceKey         string
	NotificationsBaseURL string
	NotificationsKey     string
}

func FromEnv() Config {
	return Config{
		AppEnv:               getenv("APP_ENV", "local"),
		HTTPAddr:             getenv("HTTP_ADDR", ":8080"),
		DBHost:               getenv("DB_HOST", "postgres"),
		DBPort:               getenv("DB_PORT", "5432"),
		DBUser:               getenv("DB_USER", "identity"),
		DBPassword:           os.Getenv("DB_PASSWORD"),
		DBName:               getenv("DB_NAME", "fac_identity"),
		ServiceKey:           strings.TrimSpace(os.Getenv("ENQUIRY_SERVICE_KEY")),
		CORSOrigins:          getenv("CORS_ORIGINS", "*"),
		JWTPublicKeyPath:     getenv("JWT_PUBLIC_KEY_PATH", "/keys/jwt_public.pem"),
		JWTIssuer:            getenv("JWT_ISSUER", "https://api.facinect.local"),
		JWTAudience:          getenv("JWT_AUDIENCE", "facinect-apps"),
		AIBaseURL:            strings.TrimRight(getenv("AI_BASE_URL", "http://ai:8080"), "/"),
		AIServiceKey:         strings.TrimSpace(os.Getenv("AI_SERVICE_KEY")),
		NotificationsBaseURL: strings.TrimRight(getenv("NOTIFICATIONS_BASE_URL", "http://notifications:8080"), "/"),
		NotificationsKey:     strings.TrimSpace(firstNonEmpty(os.Getenv("NOTIFICATIONS_SERVICE_KEY"), os.Getenv("ENQUIRY_SERVICE_KEY"))),
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
