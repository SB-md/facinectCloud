package stub

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/facinect/paygateway/internal/gateway"
)

// Provider is a local dry-run gateway (no external calls).
type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string       { return "stub" }
func (p *Provider) Configured() bool   { return true }

func (p *Provider) CreateOrder(ctx context.Context, req gateway.CreateRequest) (*gateway.CreateResult, error) {
	_ = ctx
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
	ext := fmt.Sprintf("stub_order_%d", time.Now().UnixNano())
	return &gateway.CreateResult{
		Provider:        "stub",
		ExternalOrderID: ext,
		Status:          "created",
		AmountPaise:     paise,
		Currency:        cur,
		Receipt:         req.Receipt,
		Checkout: map[string]interface{}{
			"mode":     "stub",
			"order_id": ext,
			"key":      "stub_key",
			"amount":   paise,
			"currency": cur,
			"name":     "Facinect Stub",
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
	if req.ExternalOrderID == "" || req.PaymentID == "" {
		return &gateway.VerifyResult{Provider: "stub", Valid: false, Reason: "missing_fields"}, nil
	}
	// Accept signature "stub" or HMAC of order|payment with secret "stub"
	ok := req.Signature == "stub" || req.Signature == stubSig(req.ExternalOrderID, req.PaymentID)
	if !ok && req.Signature == "" {
		ok = true // local convenience
	}
	st := "failed"
	if ok {
		st = "paid"
	}
	return &gateway.VerifyResult{
		Provider:        "stub",
		Valid:           ok,
		Status:          st,
		ExternalOrderID: req.ExternalOrderID,
		PaymentID:       req.PaymentID,
	}, nil
}

func (p *Provider) ParseWebhook(headers http.Header, body []byte) (*gateway.NormalizedEvent, error) {
	_ = headers
	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)
	evType := "payment.captured"
	if t, ok := raw["event"].(string); ok && t != "" {
		evType = t
	}
	orderID, _ := raw["order_id"].(string)
	payID, _ := raw["payment_id"].(string)
	return &gateway.NormalizedEvent{
		Provider:        "stub",
		Type:            evType,
		ExternalOrderID: orderID,
		PaymentID:       payID,
		Raw:             raw,
	}, nil
}

func stubSig(orderID, paymentID string) string {
	mac := hmac.New(sha256.New, []byte("stub"))
	mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}
