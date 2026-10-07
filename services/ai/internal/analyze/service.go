package analyze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/facinect/ai/internal/config"
)

type Service struct {
	Cfg    config.Config
	Client *http.Client
}

func New(cfg config.Config) *Service {
	return &Service{
		Cfg: cfg,
		Client: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

type AnalyzeResult map[string]interface{}

const systemPrompt = `You are an AI Sports Enquiry Analyzer.

Analyze CUSTOMER–COACH conversations carefully.

GOALS:
- Understand customer requirement
- Identify booking/business opportunity
- Generate structured CRM-ready enquiry output
- Return ONLY valid JSON

TASKS:
1. Detect all sports mentioned
2. Detect primary enquiry category: Booking, Membership, Coaching, Tournament
3. Extract customer requirements clearly
4. Generate: enquiry_type, customer_intent, intent_summary, customer_requirements, operational_requirements, business_opportunity, short_enquiry_tag

RULES:
- Pricing/plans/packages → Membership
- Court/slot/timing availability → Booking
- Training/coaching/classes → Coaching
- Events/matches → Tournament
- Choose ONLY one dominant category
- Do NOT guess missing information
- If sport unclear use: "Unknown - Court Sport"

OUTPUT FORMAT:
{
  "enquiry_type": "",
  "sports": [],
  "category": "",
  "customer_intent": "",
  "intent_summary": "",
  "customer_requirements": {},
  "operational_requirements": {},
  "business_opportunity": {},
  "customer_profile": {},
  "short_enquiry_tag": ""
}`

func (s *Service) AnalyzeEnquiry(ctx context.Context, transcript string) (AnalyzeResult, error) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return nil, fmt.Errorf("transcript_empty")
	}
	if len(transcript) > 15000 {
		transcript = transcript[:15000]
	}

	if s.Cfg.GeminiAPIKey == "" {
		return stubAnalyze(transcript), nil
	}
	return s.geminiAnalyze(ctx, transcript)
}

func (s *Service) SuggestReply(ctx context.Context, facilityName, customerName string, ai AnalyzeResult) (string, error) {
	intent, _ := ai["intent_summary"].(string)
	category, _ := ai["category"].(string)
	if intent == "" {
		intent = "your enquiry"
	}
	if category == "" {
		category = "services"
	}
	if customerName == "" {
		customerName = "Customer"
	}
	if facilityName == "" {
		facilityName = "our facility"
	}
	// Deterministic template (Facinect parity); optional LLM polish later.
	_ = ctx
	return fmt.Sprintf(
		"Thank you for your enquiry with %s.\nYou asked for %s, %s.\nPlease let us know if this works for you.",
		facilityName, intent, category,
	), nil
}

func stubAnalyze(transcript string) AnalyzeResult {
	lower := strings.ToLower(transcript)
	category := "Booking"
	switch {
	case strings.Contains(lower, "tournament") || strings.Contains(lower, "match event"):
		category = "Tournament"
	case strings.Contains(lower, "member") || strings.Contains(lower, "package") || strings.Contains(lower, "subscription"):
		category = "Membership"
	case strings.Contains(lower, "coach") || strings.Contains(lower, "training") || strings.Contains(lower, "class"):
		category = "Coaching"
	case strings.Contains(lower, "book") || strings.Contains(lower, "court") || strings.Contains(lower, "slot"):
		category = "Booking"
	}
	summary := "Customer enquiry received"
	if len(transcript) > 120 {
		summary = strings.TrimSpace(transcript[:120]) + "…"
	} else if transcript != "" {
		summary = transcript
	}
	return AnalyzeResult{
		"enquiry_type":             category,
		"sports":                   []string{"Unknown - Court Sport"},
		"category":                 category,
		"customer_intent":          category,
		"intent_summary":           summary,
		"customer_requirements":    map[string]interface{}{},
		"operational_requirements": map[string]interface{}{},
		"business_opportunity":     map[string]interface{}{"source": "stub"},
		"customer_profile":         map[string]interface{}{},
		"short_enquiry_tag":        category,
		"provider":                 "stub",
	}
}

type geminiReq struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig map[string]interface{} `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Service) geminiAnalyze(ctx context.Context, transcript string) (AnalyzeResult, error) {
	model := s.Cfg.GeminiModel
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, s.Cfg.GeminiAPIKey,
	)
	body := geminiReq{
		SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: systemPrompt}}},
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: "Analyze this conversation:\n\n" + transcript}},
		}},
		GenerationConfig: map[string]interface{}{
			"temperature":     0.2,
			"responseMimeType": "application/json",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini_http_%d", res.StatusCode)
	}
	var gr geminiResp
	if err := json.Unmarshal(respBody, &gr); err != nil {
		return nil, err
	}
	if gr.Error != nil && gr.Error.Message != "" {
		return nil, fmt.Errorf("gemini: %s", gr.Error.Message)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini_empty")
	}
	text := strings.TrimSpace(gr.Candidates[0].Content.Parts[0].Text)
	text = stripCodeFence(text)
	var out AnalyzeResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini_json: %w", err)
	}
	out["provider"] = "gemini"
	return out, nil
}

var fenceRe = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

func stripCodeFence(s string) string {
	m := fenceRe.FindStringSubmatch(strings.TrimSpace(s))
	if len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return s
}
