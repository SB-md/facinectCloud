package notify

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// FacilityWhatsApp mirrors PHP FacilityWhatsApp + facinect_facility_whatsapp_sender_id.
type FacilityWhatsApp struct {
	FacilityID         int64  `json:"facility_id"`
	MetaPhoneNumberID  string `json:"meta_phone_number_id,omitempty"`
	WhatsAppAPIToken   string `json:"whatsapp_api_token,omitempty"` // write-only on GET (masked)
	HasTokenOverride   bool   `json:"has_token_override"`
	WABAID             string `json:"waba_id,omitempty"`
	WhatsAppEnabled    bool   `json:"whatsapp_enabled"`
	// Resolved values used for send (not always persisted)
	ResolvedToken   string `json:"-"`
	ResolvedPhoneID string `json:"-"`
	TokenFrom       string `json:"token_from,omitempty"` // "facility" | "env" | ""
	PhoneFrom       string `json:"phone_from,omitempty"` // "facility" | "env" | ""
}

type FacilityWhatsAppUpsert struct {
	MetaPhoneNumberID *string `json:"meta_phone_number_id"`
	WhatsAppAPIToken  *string `json:"whatsapp_api_token"` // null = leave; "" = clear to env
	WABAID            *string `json:"waba_id"`
	WhatsAppEnabled   *bool   `json:"whatsapp_enabled"`
}

// ResolveWhatsAppCreds: facility DB override → env fallback (PHP logic).
func (s *Service) ResolveWhatsAppCreds(ctx context.Context, facilityID int64) (*FacilityWhatsApp, error) {
	out := &FacilityWhatsApp{
		FacilityID:      facilityID,
		WhatsAppEnabled: true,
	}

	if facilityID > 0 {
		row := s.DB.QueryRowContext(ctx, `
SELECT COALESCE(meta_phone_number_id,''), COALESCE(whatsapp_api_token,''),
       COALESCE(waba_id,''), whatsapp_enabled
FROM facility_whatsapp_config WHERE facility_id=$1`, facilityID)
		var phone, token, waba string
		var enabled bool
		err := row.Scan(&phone, &token, &waba, &enabled)
		if err == nil {
			out.MetaPhoneNumberID = phone
			out.WABAID = waba
			out.WhatsAppEnabled = enabled
			out.HasTokenOverride = strings.TrimSpace(token) != ""
			if out.HasTokenOverride {
				out.ResolvedToken = token
				out.TokenFrom = "facility"
			}
			if strings.TrimSpace(phone) != "" {
				out.ResolvedPhoneID = phone
				out.PhoneFrom = "facility"
			}
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	if out.ResolvedToken == "" {
		out.ResolvedToken = s.Cfg.MetaWhatsAppToken
		if out.ResolvedToken != "" {
			out.TokenFrom = "env"
		}
	}
	if out.ResolvedPhoneID == "" {
		out.ResolvedPhoneID = s.Cfg.MetaPhoneNumberID
		if out.ResolvedPhoneID != "" {
			out.PhoneFrom = "env"
		}
	}
	return out, nil
}

func (s *Service) GetFacilityWhatsApp(ctx context.Context, facilityID int64) (*FacilityWhatsApp, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility_id")
	}
	cfg, err := s.ResolveWhatsAppCreds(ctx, facilityID)
	if err != nil {
		return nil, err
	}
	// Never return raw token on GET
	cfg.WhatsAppAPIToken = ""
	return cfg, nil
}

func (s *Service) UpsertFacilityWhatsApp(ctx context.Context, facilityID int64, in FacilityWhatsAppUpsert) (*FacilityWhatsApp, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility_id")
	}

	cur, err := s.ResolveWhatsAppCreds(ctx, facilityID)
	if err != nil {
		return nil, err
	}

	phone := cur.MetaPhoneNumberID
	token := ""
	if cur.HasTokenOverride {
		// keep existing token unless explicitly updated
		row := s.DB.QueryRowContext(ctx, `SELECT COALESCE(whatsapp_api_token,'') FROM facility_whatsapp_config WHERE facility_id=$1`, facilityID)
		_ = row.Scan(&token)
	}
	waba := cur.WABAID
	enabled := cur.WhatsAppEnabled

	if in.MetaPhoneNumberID != nil {
		phone = strings.TrimSpace(*in.MetaPhoneNumberID)
	}
	if in.WhatsAppAPIToken != nil {
		token = strings.TrimSpace(*in.WhatsAppAPIToken)
	}
	if in.WABAID != nil {
		waba = strings.TrimSpace(*in.WABAID)
	}
	if in.WhatsAppEnabled != nil {
		enabled = *in.WhatsAppEnabled
	}

	_, err = s.DB.ExecContext(ctx, `
INSERT INTO facility_whatsapp_config (
  facility_id, meta_phone_number_id, whatsapp_api_token, waba_id, whatsapp_enabled, updated_at
) VALUES ($1, NULLIF($2,''), NULLIF($3,''), NULLIF($4,''), $5, NOW())
ON CONFLICT (facility_id) DO UPDATE SET
  meta_phone_number_id = EXCLUDED.meta_phone_number_id,
  whatsapp_api_token = CASE
    WHEN $6 THEN EXCLUDED.whatsapp_api_token
    ELSE facility_whatsapp_config.whatsapp_api_token
  END,
  waba_id = EXCLUDED.waba_id,
  whatsapp_enabled = EXCLUDED.whatsapp_enabled,
  updated_at = NOW()`,
		facilityID, phone, token, waba, enabled, in.WhatsAppAPIToken != nil,
	)
	if err != nil {
		return nil, err
	}
	return s.GetFacilityWhatsApp(ctx, facilityID)
}

func (s *Service) FindFacilityByPhoneNumberID(ctx context.Context, phoneNumberID string) (int64, error) {
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	if phoneNumberID == "" {
		return 0, fmt.Errorf("phone_number_id_required")
	}
	var id int64
	err := s.DB.QueryRowContext(ctx, `
SELECT facility_id FROM facility_whatsapp_config
WHERE meta_phone_number_id = $1 LIMIT 1`, phoneNumberID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("not_found")
	}
	return id, err
}
