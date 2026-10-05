package main

import (
	"log"
	"net/http"
	"time"

	"github.com/facinect/identity/internal/auth"
	"github.com/facinect/identity/internal/config"
	"github.com/facinect/identity/internal/db"
	"github.com/facinect/identity/internal/httpapi"
)

func main() {
	cfg := config.FromEnv()
	sqlDB, err := db.Open(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer sqlDB.Close()
	if err := db.EnsureSchema(sqlDB); err != nil {
		log.Fatalf("schema: %v", err)
	}

	jwtSvc, err := auth.NewJWTService(
		cfg.JWTPrivateKeyPath,
		cfg.JWTPublicKeyPath,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		time.Duration(cfg.AccessTTLSeconds)*time.Second,
		time.Duration(cfg.RefreshTTLSeconds)*time.Second,
	)
	if err != nil {
		log.Fatalf("jwt: %v", err)
	}

	svc := &auth.Service{
		DB:     sqlDB,
		JWT:    jwtSvc,
		Google: auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI),
		AppEnv: cfg.AppEnv,
	}
	if err := svc.BootstrapAdmin(cfg.BootstrapEmail, cfg.BootstrapPassword, cfg.BootstrapName); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	srv := &httpapi.Server{Cfg: cfg, Auth: svc}
	log.Printf("identity (go) listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
