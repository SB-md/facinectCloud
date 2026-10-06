package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/facinect/notifications/internal/config"
)

type PushClient struct {
	Cfg    config.Config
	Client *http.Client
}

func NewPush(cfg config.Config) *PushClient {
	return &PushClient{
		Cfg: cfg,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *PushClient) Send(ctx context.Context, req SendRequest) (string, error) {
	tokens := req.To.DeviceTokens
	if len(tokens) == 0 {
		return "", fmt.Errorf("device_token_required")
	}
	title := "Facinect"
	body := strings.TrimSpace(req.TemplateKey)
	if body == "" {
		body = "You have a new notification"
	}
	if req.Data != nil {
		if t, ok := req.Data["title"].(string); ok && t != "" {
			title = t
		}
		if b, ok := req.Data["body"].(string); ok && b != "" {
			body = b
		}
	}

	if p.Cfg.DryRun || !p.Cfg.PushConfigured() {
		return fmt.Sprintf("dry_run:push:%d", len(tokens)), nil
	}

	// Legacy FCM HTTP API (simple N1). Migrate to HTTP v1 later.
	payload := map[string]interface{}{
		"registration_ids": tokens,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
		"data": req.Data,
	}
	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fcm.googleapis.com/fcm/send", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "key="+p.Cfg.FCMServerKey)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := p.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("fcm_api_%d: %s", res.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return "fcm:ok", nil
}
