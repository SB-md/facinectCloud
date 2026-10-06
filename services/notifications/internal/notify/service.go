package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/facinect/notifications/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
	WA  *WhatsAppClient
	Push *PushClient
}

type SendRequest struct {
	Channel      string                 `json:"channel"`
	FacilityID   *int64                 `json:"facility_id"`
	TemplateKey  string                 `json:"template"`
	To           SendTo                 `json:"to"`
	Data         map[string]interface{} `json:"data"`
	Priority     string                 `json:"priority"`
}

type SendTo struct {
	WhatsApp     string   `json:"whatsapp"`
	UserID       *int64   `json:"user_id"`
	DeviceTokens []string `json:"device_tokens"`
}

type Job struct {
	ID          int64           `json:"id"`
	FacilityID  *int64          `json:"facility_id,omitempty"`
	Channel     string          `json:"channel"`
	TemplateKey string          `json:"template_key,omitempty"`
	ToWhatsApp  string          `json:"to_whatsapp,omitempty"`
	ToUserID    *int64          `json:"to_user_id,omitempty"`
	Status      string          `json:"status"`
	ProviderRef string          `json:"provider_ref,omitempty"`
	ErrorText   string          `json:"error_text,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (s *Service) Send(ctx context.Context, req SendRequest) (*Job, error) {
	ch := strings.ToLower(strings.TrimSpace(req.Channel))
	if ch == "" {
		ch = "whatsapp"
	}
	if ch != "whatsapp" && ch != "push" && ch != "both" {
		return nil, fmt.Errorf("invalid_channel")
	}
	wa := normalizePhone(req.To.WhatsApp)
	if (ch == "whatsapp" || ch == "both") && wa == "" && req.To.UserID == nil {
		return nil, fmt.Errorf("whatsapp_or_user_required")
	}
	if (ch == "push" || ch == "both") && len(req.To.DeviceTokens) == 0 && req.To.UserID == nil {
		return nil, fmt.Errorf("device_token_or_user_required")
	}

	payload := map[string]interface{}{
		"template": req.TemplateKey,
		"data":     req.Data,
		"priority": req.Priority,
		"to":       req.To,
	}
	raw, _ := json.Marshal(payload)

	var facility interface{}
	if req.FacilityID != nil {
		facility = *req.FacilityID
	}
	var userID interface{}
	if req.To.UserID != nil {
		userID = *req.To.UserID
	}

	var id int64
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO notification_jobs (facility_id, channel, template_key, to_whatsapp, to_user_id, payload_json, status)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, 'pending')
RETURNING id`,
		facility, ch, nullStr(req.TemplateKey), nullStr(wa), userID, string(raw),
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.process(ctx, job, req)
}

func (s *Service) process(ctx context.Context, job *Job, req SendRequest) (*Job, error) {
	_, _ = s.DB.ExecContext(ctx, `UPDATE notification_jobs SET status='processing', updated_at=NOW() WHERE id=$1`, job.ID)

	var lastErr error
	providerRef := ""
	status := "sent"

	if job.Channel == "whatsapp" || job.Channel == "both" {
		facilityID := int64(0)
		if req.FacilityID != nil {
			facilityID = *req.FacilityID
		}
		credsRow, err := s.ResolveWhatsAppCreds(ctx, facilityID)
		if err != nil {
			lastErr = err
			status = "failed"
		} else if facilityID > 0 && !credsRow.WhatsAppEnabled {
			lastErr = fmt.Errorf("whatsapp_disabled_for_facility")
			status = "failed"
		} else {
			ref, err := s.WA.Send(ctx, req, WhatsAppSendCreds{
				AccessToken:   credsRow.ResolvedToken,
				PhoneNumberID: credsRow.ResolvedPhoneID,
			})
			if err != nil {
				lastErr = err
				status = "failed"
			} else {
				providerRef = ref
				if strings.HasPrefix(ref, "dry_run:") {
					status = "dry_run"
				}
			}
		}
	}

	if lastErr == nil && (job.Channel == "push" || job.Channel == "both") {
		ref, err := s.Push.Send(ctx, req)
		if err != nil {
			lastErr = err
			status = "failed"
		} else {
			if providerRef != "" {
				providerRef = providerRef + "," + ref
			} else {
				providerRef = ref
			}
			if strings.HasPrefix(ref, "dry_run:") && status == "sent" {
				status = "dry_run"
			}
		}
	}

	errText := ""
	if lastErr != nil {
		errText = lastErr.Error()
	}
	_, _ = s.DB.ExecContext(ctx, `
UPDATE notification_jobs
SET status=$2, provider_ref=$3, error_text=$4, updated_at=NOW()
WHERE id=$1`, job.ID, status, nullStr(providerRef), nullStr(errText))

	return s.GetJob(ctx, job.ID)
}

func (s *Service) GetJob(ctx context.Context, id int64) (*Job, error) {
	row := s.DB.QueryRowContext(ctx, `
SELECT id, facility_id, channel, COALESCE(template_key,''), COALESCE(to_whatsapp,''), to_user_id,
       payload_json, status, COALESCE(provider_ref,''), COALESCE(error_text,''), created_at, updated_at
FROM notification_jobs WHERE id=$1`, id)
	var j Job
	var facility, userID sql.NullInt64
	var payload []byte
	if err := row.Scan(
		&j.ID, &facility, &j.Channel, &j.TemplateKey, &j.ToWhatsApp, &userID,
		&payload, &j.Status, &j.ProviderRef, &j.ErrorText, &j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if facility.Valid {
		v := facility.Int64
		j.FacilityID = &v
	}
	if userID.Valid {
		v := userID.Int64
		j.ToUserID = &v
	}
	j.Payload = payload
	return &j, nil
}

func (s *Service) RegisterDevice(ctx context.Context, userID int64, token, platform string) error {
	token = strings.TrimSpace(token)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if userID <= 0 || token == "" {
		return fmt.Errorf("invalid_device")
	}
	if platform == "" {
		platform = "android"
	}
	if platform != "android" && platform != "ios" && platform != "web" {
		return fmt.Errorf("invalid_platform")
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO device_tokens (user_id, token, platform, status)
VALUES ($1, $2, $3, 'active')
ON CONFLICT (user_id, token) DO UPDATE
SET platform=EXCLUDED.platform, status='active', updated_at=NOW()`,
		userID, token, platform)
	return err
}

func normalizePhone(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, " ", "")
	p = strings.ReplaceAll(p, "+", "")
	p = strings.ReplaceAll(p, "-", "")
	return p
}

func nullStr(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
