package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	RedisHost          string
	RedisPort          string
	JWTPrivateKeyPath  string
	JWTPublicKeyPath   string
	JWTIssuer          string
	JWTAudience        string
	AccessTTLSeconds   int
	RefreshTTLSeconds  int
	BootstrapEmail     string
	BootstrapPassword  string
	BootstrapName      string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	CORSOrigins        string
	NotificationsURL   string
	NotificationsKey   string
	OTPTemplate        string
	OTPDevInline       bool
}

func FromEnv() Config {
	devInline := strings.EqualFold(os.Getenv("OTP_DEV_INLINE"), "true") ||
		strings.EqualFold(os.Getenv("OTP_DEV_INLINE"), "1")
	return Config{
		AppEnv:             getenv("APP_ENV", "local"),
		HTTPAddr:           getenv("HTTP_ADDR", ":8080"),
		DBHost:             getenv("DB_HOST", "postgres"),
		DBPort:             getenv("DB_PORT", "5432"),
		DBName:             getenv("DB_NAME", "fac_identity"),
		DBUser:             getenv("DB_USER", "identity"),
		DBPassword:         getenv("DB_PASSWORD", ""),
		RedisHost:          getenv("REDIS_HOST", "redis"),
		RedisPort:          getenv("REDIS_PORT", "6379"),
		JWTPrivateKeyPath:  getenv("JWT_PRIVATE_KEY_PATH", "/keys/jwt_private.pem"),
		JWTPublicKeyPath:   getenv("JWT_PUBLIC_KEY_PATH", "/keys/jwt_public.pem"),
		JWTIssuer:          getenv("JWT_ISSUER", "https://api.facinect.local"),
		JWTAudience:        getenv("JWT_AUDIENCE", "facinect-apps"),
		AccessTTLSeconds:   getenvInt("JWT_ACCESS_TTL_SECONDS", 900),
		RefreshTTLSeconds:  getenvInt("JWT_REFRESH_TTL_SECONDS", 2592000),
		BootstrapEmail:     strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")),
		BootstrapPassword:  os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapName:      getenv("BOOTSTRAP_ADMIN_NAME", "Facinect Admin"),
		GoogleClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURI:  getenv("GOOGLE_OAUTH_REDIRECT_URI", "http://localhost:8080/v1/auth/google/callback"),
		CORSOrigins:        getenv("CORS_ORIGINS", "*"),
		NotificationsURL:   strings.TrimRight(getenv("NOTIFICATIONS_BASE_URL", "http://notifications:8080"), "/"),
		NotificationsKey:   strings.TrimSpace(os.Getenv("NOTIFICATIONS_SERVICE_KEY")),
		OTPTemplate:        getenv("OTP_WHATSAPP_TEMPLATE", "facinect_otp"),
		OTPDevInline:       devInline,
	}
}

func (c Config) GoogleConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
