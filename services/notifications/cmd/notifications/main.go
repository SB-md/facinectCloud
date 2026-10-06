package main

import (
	"log"
	"net/http"

	"github.com/facinect/notifications/internal/config"
	"github.com/facinect/notifications/internal/db"
	"github.com/facinect/notifications/internal/httpapi"
	"github.com/facinect/notifications/internal/notify"
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

	svc := &notify.Service{
		DB:   sqlDB,
		Cfg:  cfg,
		WA:   notify.NewWhatsApp(cfg),
		Push: notify.NewPush(cfg),
	}
	srv := &httpapi.Server{Cfg: cfg, Notify: svc}
	log.Printf("notifications (go) listening on %s dry_run=%v whatsapp=%v push=%v",
		cfg.HTTPAddr, cfg.DryRun, cfg.WhatsAppConfigured(), cfg.PushConfigured())
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
