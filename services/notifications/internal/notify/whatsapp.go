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

type WhatsAppClient struct {
	Cfg    config.Config
	Client *http.Client
}

type WhatsAppSendCreds struct {
	AccessToken   string
	PhoneNumberID string
}

func NewWhatsApp(cfg config.Config) *WhatsAppClient {
	return &WhatsAppClient{
		Cfg: cfg,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (w *WhatsAppClient) Send(ctx context.Context, req SendRequest, creds WhatsAppSendCreds) (string, error) {
	to := normalizePhone(req.To.WhatsApp)
	if to == "" {
		return "", fmt.Errorf("whatsapp_required")
	}
	tmpl := strings.TrimSpace(req.TemplateKey)
	if tmpl == "" {
		tmpl = "hello_world"
	}

	token := strings.TrimSpace(creds.AccessToken)
	phoneID := strings.TrimSpace(creds.PhoneNumberID)
	if token == "" {
		token = w.Cfg.MetaWhatsAppToken
	}
	if phoneID == "" {
		phoneID = w.Cfg.MetaPhoneNumberID
	}

	if w.Cfg.DryRun || token == "" || phoneID == "" {
		return fmt.Sprintf("dry_run:whatsapp:%s:%s:phone=%s", to, tmpl, phoneID), nil
	}

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name": tmpl,
			"language": map[string]string{
				"code": "en",
			},
		},
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf(
		"https://graph.facebook.com/%s/%s/messages",
		w.Cfg.MetaAPIVersion,
		phoneID,
	)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := w.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("whatsapp_api_%d: %s", res.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed map[string]interface{}
	_ = json.Unmarshal(respBody, &parsed)
	if msgs, ok := parsed["messages"].([]interface{}); ok && len(msgs) > 0 {
		if m, ok := msgs[0].(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				return id, nil
			}
		}
	}
	return "whatsapp:ok", nil
}
