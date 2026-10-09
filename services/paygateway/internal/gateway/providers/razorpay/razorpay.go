package razorpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/facinect/paygateway/internal/gateway"
)

type Provider struct {
	KeyID         string
	KeySecret     string
	WebhookSecret string
	HTTP          *http.Client
}

func New(keyID, keySecret, webhookSecret string) *Provider {
	return &Provider{
		KeyID:         strings.TrimSpace(keyID),
		KeySecret:     strings.TrimSpace(keySecret),
		WebhookSecret: strings.TrimSpace(webhookSecret),
		HTTP:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *Provider) Name() string { return "razorpay" }

func (p *Provider) Configured() bool {
	return p.KeyID != "" && p.KeySecret != ""
}

func (p *Provider) CreateOrder(ctx context.Context, req gateway.CreateRequest) (*gateway.CreateResult, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("razorpay_not_configured")
	}
	paise := req.AmountPaise
	if paise <= 0 && req.Amount > 0 {
		paise = int64(req.Amount*100 + 0.5)
	}
	if paise <= 0 {
		return nil, fmt.Errorf("invalid_amount")
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "INR"
	}
	body := map[string]interface{}{
		"amount":   paise,
		"currency": cur,
	}
	if req.Receipt != "" {
		body["receipt"] = req.Receipt
	}
	notes := map[string]interface{}{}
	for k, v := range req.Metadata {
		notes[k] = v
	}
	if req.Purpose != "" {
		notes["purpose"] = req.Purpose
	}
	if req.FacilityID != nil {
		notes["facility_id"] = *req.FacilityID
	}
	if len(notes) > 0 {
		body["notes"] = notes
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.SetBasicAuth(p.KeyID, p.KeySecret)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("razorpay_http_%d: %s", res.StatusCode, truncate(string(respBody), 200))
	}
	var out struct {
		ID       string `json:"id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
		Receipt  string `json:"receipt"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &gateway.CreateResult{
		Provider:        "razorpay",
		ExternalOrderID: out.ID,
		Status:          "created",
		AmountPaise:     out.Amount,
		Currency:        out.Currency,
		Receipt:         out.Receipt,
		Checkout: map[string]interface{}{
			"key":      p.KeyID,
			"order_id": out.ID,
			"amount":   out.Amount,
			"currency": out.Currency,
			"name":     "Facinect",
			"prefill": map[string]string{
				"name":    req.Customer.Name,
				"email":   req.Customer.Email,
				"contact": req.Customer.Phone,
			},
		},
	}, nil
}

func (p *Provider) Verify(ctx context.Context, req gateway.VerifyRequest) (*gateway.VerifyResult, error) {
	_ = ctx
	if !p.Configured() {
		return nil, fmt.Errorf("razorpay_not_configured")
	}
	if req.ExternalOrderID == "" || req.PaymentID == "" || req.Signature == "" {
		return &gateway.VerifyResult{Provider: "razorpay", Valid: false, Reason: "missing_fields"}, nil
	}
	mac := hmac.New(sha256.New, []byte(p.KeySecret))
	mac.Write([]byte(req.ExternalOrderID + "|" + req.PaymentID))
	expected := hex.EncodeToString(mac.Sum(nil))
	ok := hmac.Equal([]byte(expected), []byte(req.Signature))
	st := "failed"
	if ok {
		st = "paid"
	}
	return &gateway.VerifyResult{
		Provider:        "razorpay",
		Valid:           ok,
		Status:          st,
		ExternalOrderID: req.ExternalOrderID,
		PaymentID:       req.PaymentID,
		Reason:          map[bool]string{true: "", false: "invalid_signature"}[ok],
	}, nil
}

func (p *Provider) ParseWebhook(headers http.Header, body []byte) (*gateway.NormalizedEvent, error) {
	if p.WebhookSecret != "" {
		sig := headers.Get("X-Razorpay-Signature")
		mac := hmac.New(sha256.New, []byte(p.WebhookSecret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(sig)) {
			return nil, fmt.Errorf("invalid_webhook_signature")
		}
	}
	var payload struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity map[string]interface{} `json:"entity"`
			} `json:"payment"`
			Order struct {
				Entity map[string]interface{} `json:"entity"`
			} `json:"order"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	ev := &gateway.NormalizedEvent{
		Provider: "razorpay",
		Type:     normalizeRazorpayEvent(payload.Event),
		Raw:      map[string]interface{}{"event": payload.Event},
	}
	if id, ok := payload.Payload.Payment.Entity["id"].(string); ok {
		ev.PaymentID = id
	}
	if oid, ok := payload.Payload.Payment.Entity["order_id"].(string); ok {
		ev.ExternalOrderID = oid
	}
	if oid, ok := payload.Payload.Order.Entity["id"].(string); ok && ev.ExternalOrderID == "" {
		ev.ExternalOrderID = oid
	}
	if amt, ok := asInt64(payload.Payload.Payment.Entity["amount"]); ok {
		ev.AmountPaise = amt
	}
	if cur, ok := payload.Payload.Payment.Entity["currency"].(string); ok {
		ev.Currency = cur
	}
	return ev, nil
}

func normalizeRazorpayEvent(e string) string {
	switch e {
	case "payment.captured", "order.paid":
		return "payment.captured"
	case "payment.failed":
		return "payment.failed"
	case "refund.created", "refund.processed":
		return "refunded"
	default:
		if e == "" {
			return "unknown"
		}
		return e
	}
}

func asInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
