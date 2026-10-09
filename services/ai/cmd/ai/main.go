package main

import (
	"log"
	"net/http"

	"github.com/facinect/ai/internal/analyze"
	"github.com/facinect/ai/internal/config"
	"github.com/facinect/ai/internal/httpapi"
)

func main() {
	cfg := config.FromEnv()
	svc := analyze.New(cfg)
	srv := &httpapi.Server{Cfg: cfg, AI: svc}
	log.Printf("ai (go) listening on %s gemini=%v service_key=%v",
		cfg.HTTPAddr, cfg.GeminiAPIKey != "", cfg.ServiceKey != "")
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
