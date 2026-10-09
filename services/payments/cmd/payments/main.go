package main

import (
	"log"
	"net/http"

	"github.com/facinect/payments/internal/authn"
	"github.com/facinect/payments/internal/config"
	"github.com/facinect/payments/internal/db"
	"github.com/facinect/payments/internal/httpapi"
	"github.com/facinect/payments/internal/payments"
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

	var jwtVerifier *authn.Verifier
	jv, err := authn.NewVerifier(cfg.JWTPublicKeyPath, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		if cfg.IsLocal() {
			log.Printf("jwt verifier unavailable (local): %v", err)
		} else {
			log.Fatalf("jwt: %v", err)
		}
	} else {
		jwtVerifier = jv
	}

	svc := &payments.Service{DB: sqlDB, Cfg: cfg}
	srv := &httpapi.Server{Cfg: cfg, Payments: svc, JWT: jwtVerifier}
	log.Printf("payments (go) listening on %s service_key=%v jwt=%v",
		cfg.HTTPAddr, cfg.ServiceKey != "", jwtVerifier != nil)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
