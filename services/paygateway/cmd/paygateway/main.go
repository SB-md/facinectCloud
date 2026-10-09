package main

import (
	"log"
	"net/http"

	"github.com/facinect/paygateway/internal/authn"
	"github.com/facinect/paygateway/internal/config"
	"github.com/facinect/paygateway/internal/db"
	"github.com/facinect/paygateway/internal/gateway"
	"github.com/facinect/paygateway/internal/gateway/providers/razorpay"
	"github.com/facinect/paygateway/internal/gateway/providers/stub"
	"github.com/facinect/paygateway/internal/httpapi"
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

	reg := gateway.NewRegistry(cfg.DefaultProvider)
	reg.Register(stub.New())
	reg.Register(razorpay.New(cfg.RazorpayKeyID, cfg.RazorpayKeySecret, cfg.RazorpayWebhookSecret))

	svc := &gateway.Service{DB: sqlDB, Cfg: cfg, Registry: reg}
	srv := &httpapi.Server{Cfg: cfg, Gateway: svc, JWT: jwtVerifier}
	log.Printf("paygateway (go) listening on %s default=%s razorpay=%v stub=true",
		cfg.HTTPAddr, cfg.DefaultProvider, cfg.RazorpayKeyID != "" && cfg.RazorpayKeySecret != "")
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
