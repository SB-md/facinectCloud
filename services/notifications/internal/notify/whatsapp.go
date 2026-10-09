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
			Timeout: 20 * time.Second,
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
	lang := "en"
	if req.Data != nil {
		if v, ok := req.Data["language"].(string); ok && strings.TrimSpace(v) != "" {
			lang = strings.TrimSpace(v)
		}
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

	template := map[string]interface{}{
		"name": tmpl,
		"language": map[string]string{
			"code": lang,
		},
	}
	if comps := buildTemplateComponents(req); len(comps) > 0 {
		template["components"] = comps
	}

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template":          template,
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

func dataString(data map[string]interface{}, keys ...string) string {
	if data == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := data[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

// buildTemplateComponents mirrors Facinect PHP OTP + generic body-param templates.
func buildTemplateComponents(req SendRequest) []interface{} {
	otp := dataString(req.Data, "otp", "code", "otp_code")
	if otp != "" {
		return []interface{}{
			map[string]interface{}{
				"type": "body",
				"parameters": []map[string]string{
					{"type": "text", "text": otp},
				},
			},
			map[string]interface{}{
				"type":     "button",
				"sub_type": "url",
				"index":    "0",
				"parameters": []map[string]string{
					{"type": "text", "text": otp},
				},
			},
		}
	}

	// Explicit body_params: ["a","b"]
	if raw, ok := req.Data["body_params"]; ok {
		params := []map[string]string{}
		switch list := raw.(type) {
		case []interface{}:
			for _, item := range list {
				s := strings.TrimSpace(fmt.Sprint(item))
				if s != "" && s != "<nil>" {
					params = append(params, map[string]string{"type": "text", "text": s})
				}
			}
		case []string:
			for _, s := range list {
				s = strings.TrimSpace(s)
				if s != "" {
					params = append(params, map[string]string{"type": "text", "text": s})
				}
			}
		}
		if len(params) > 0 {
			return []interface{}{
				map[string]interface{}{"type": "body", "parameters": params},
			}
		}
	}

	body := dataString(req.Data, "body", "text", "message")
	if body != "" {
		return []interface{}{
			map[string]interface{}{
				"type": "body",
				"parameters": []map[string]string{
					{"type": "text", "text": body},
				},
			},
		}
	}
	return nil
}
