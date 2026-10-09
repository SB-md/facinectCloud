package main

import (
	"log"
	"net/http"

	"github.com/facinect/booking/internal/authn"
	"github.com/facinect/booking/internal/booking"
	"github.com/facinect/booking/internal/config"
	"github.com/facinect/booking/internal/db"
	"github.com/facinect/booking/internal/httpapi"
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

	svc := &booking.Service{DB: sqlDB, Cfg: cfg}
	srv := &httpapi.Server{Cfg: cfg, Book: svc, JWT: jwtVerifier}
	log.Printf("booking (go) listening on %s service_key=%v jwt=%v",
		cfg.HTTPAddr, cfg.ServiceKey != "", jwtVerifier != nil)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
