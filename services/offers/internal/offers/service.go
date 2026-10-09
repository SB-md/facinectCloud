package offers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/facinect/offers/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

type OfferRow struct {
	ID            int64    `json:"id"`
	FacilityID    int64    `json:"facility_id"`
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	OfferKind     string   `json:"offer_kind"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue float64  `json:"discount_value"`
	ValidFrom     string   `json:"valid_from,omitempty"`
	ValidTo       string   `json:"valid_to,omitempty"`
	UsageLimit    *int     `json:"usage_limit,omitempty"`
	UsedCount     int      `json:"used_count"`
	SportID       *int64   `json:"sport_id,omitempty"`
	Status        string   `json:"status"`
	IsExpired     bool     `json:"is_expired"`
	CreatedAt     string   `json:"created_at,omitempty"`
}

type CreateInput struct {
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	OfferKind     string   `json:"offer_kind"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue float64  `json:"discount_value"`
	ValidFrom     string   `json:"valid_from"`
	ValidTo       string   `json:"valid_to"`
	UsageLimit    *int     `json:"usage_limit"`
	SportID       *int64   `json:"sport_id"`
	DurationDays  *int     `json:"duration_days"`
}

type UpdateStatusInput struct {
	Status string `json:"status"`
}

type ValidateResult struct {
	Valid         bool     `json:"valid"`
	Reason        string   `json:"reason,omitempty"`
	Offer         *OfferRow `json:"offer,omitempty"`
	DiscountType  string   `json:"discount_type,omitempty"`
	DiscountValue float64  `json:"discount_value,omitempty"`
}

func (s *Service) List(ctx context.Context, facilityID int64, kind, status string) ([]OfferRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	q := `
SELECT id, facility_id, code, title, COALESCE(description,''),
       offer_kind, discount_type, discount_value,
       COALESCE(valid_from::text,''), COALESCE(valid_to::text,''),
       usage_limit, used_count, sport_id, status, created_at::text
FROM offers
WHERE facility_id=$1`
	args := []interface{}{facilityID}
	argN := 2

	k := strings.ToLower(strings.TrimSpace(kind))
	if k != "" && k != "all" {
		q += fmt.Sprintf(` AND offer_kind=$%d`, argN)
		args = append(args, k)
		argN++
	}
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "" && st != "all" {
		q += fmt.Sprintf(` AND status=$%d`, argN)
		args = append(args, st)
	}
	q += ` ORDER BY created_at DESC, id DESC`

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OfferRow{}
	today := time.Now().Format("2006-01-02")
	for rows.Next() {
		r, err := scanOffer(rows)
		if err != nil {
			return nil, err
		}
		r.IsExpired = r.ValidTo != "" && r.ValidTo < today
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) CountActive(ctx context.Context, facilityID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM offers
WHERE facility_id=$1 AND status='active'
  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)`, facilityID).Scan(&n)
	return n, err
}

func (s *Service) Create(ctx context.Context, facilityID int64, in CreateInput) (*OfferRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	code := normalizeCode(in.Code)
	title := strings.TrimSpace(in.Title)
	if code == "" || title == "" {
		return nil, fmt.Errorf("invalid_offer")
	}
	kind := strings.ToLower(strings.TrimSpace(in.OfferKind))
	if kind == "" {
		kind = "discount"
	}
	if kind != "promotion" && kind != "discount" {
		return nil, fmt.Errorf("invalid_offer_kind")
	}
	dtype := strings.ToLower(strings.TrimSpace(in.DiscountType))
	if dtype == "" {
		dtype = "flat"
	}
	if dtype != "flat" && dtype != "percent" {
		return nil, fmt.Errorf("invalid_discount_type")
	}
	if in.DiscountValue < 0 {
		return nil, fmt.Errorf("invalid_discount_value")
	}
	if dtype == "percent" && in.DiscountValue > 100 {
		return nil, fmt.Errorf("invalid_discount_value")
	}

	validFrom := strings.TrimSpace(in.ValidFrom)
	validTo := strings.TrimSpace(in.ValidTo)
	if validFrom != "" {
		if err := validateDate(validFrom); err != nil {
			return nil, err
		}
	}
	if validTo != "" {
		if err := validateDate(validTo); err != nil {
			return nil, err
		}
	}
	if validTo == "" && in.DurationDays != nil && *in.DurationDays > 0 {
		base := time.Now()
		if validFrom != "" {
			if t, err := time.Parse("2006-01-02", validFrom); err == nil {
				base = t
			}
		} else {
			validFrom = base.Format("2006-01-02")
		}
		validTo = base.AddDate(0, 0, *in.DurationDays).Format("2006-01-02")
	}
	if validFrom != "" && validTo != "" && validTo < validFrom {
		return nil, fmt.Errorf("invalid_date_range")
	}

	var fromArg, toArg, sportArg, limitArg interface{}
	if validFrom != "" {
		fromArg = validFrom
	}
	if validTo != "" {
		toArg = validTo
	}
	if in.SportID != nil && *in.SportID > 0 {
		sportArg = *in.SportID
	}
	if in.UsageLimit != nil {
		if *in.UsageLimit < 0 {
			return nil, fmt.Errorf("invalid_usage_limit")
		}
		limitArg = *in.UsageLimit
	}

	var id int64
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO offers
  (facility_id, code, title, description, offer_kind, discount_type, discount_value,
   valid_from, valid_to, usage_limit, sport_id, status)
VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8::date,$9::date,$10,$11,'active')
RETURNING id`,
		facilityID, code, title, strings.TrimSpace(in.Description),
		kind, dtype, in.DiscountValue, fromArg, toArg, limitArg, sportArg,
	).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("code_exists")
		}
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*OfferRow, error) {
	row := s.DB.QueryRowContext(ctx, `
SELECT id, facility_id, code, title, COALESCE(description,''),
       offer_kind, discount_type, discount_value,
       COALESCE(valid_from::text,''), COALESCE(valid_to::text,''),
       usage_limit, used_count, sport_id, status, created_at::text
FROM offers WHERE id=$1`, id)
	r, err := scanOffer(row)
	if err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	r.IsExpired = r.ValidTo != "" && r.ValidTo < today
	return &r, nil
}

func (s *Service) UpdateStatus(ctx context.Context, facilityID, offerID int64, status string) (*OfferRow, error) {
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "active" && st != "inactive" {
		return nil, fmt.Errorf("invalid_status")
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE offers SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3`, st, offerID, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found")
	}
	return s.Get(ctx, offerID)
}

func (s *Service) Validate(ctx context.Context, facilityID int64, code string, sportID *int64) (*ValidateResult, error) {
	code = normalizeCode(code)
	if facilityID <= 0 || code == "" {
		return &ValidateResult{Valid: false, Reason: "invalid_request"}, nil
	}
	row := s.DB.QueryRowContext(ctx, `
SELECT id, facility_id, code, title, COALESCE(description,''),
       offer_kind, discount_type, discount_value,
       COALESCE(valid_from::text,''), COALESCE(valid_to::text,''),
       usage_limit, used_count, sport_id, status, created_at::text
FROM offers WHERE facility_id=$1 AND code=$2`, facilityID, code)
	r, err := scanOffer(row)
	if err == sql.ErrNoRows {
		return &ValidateResult{Valid: false, Reason: "not_found"}, nil
	}
	if err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	r.IsExpired = r.ValidTo != "" && r.ValidTo < today

	if r.Status != "active" {
		return &ValidateResult{Valid: false, Reason: "inactive", Offer: &r}, nil
	}
	if r.ValidFrom != "" && today < r.ValidFrom {
		return &ValidateResult{Valid: false, Reason: "not_started", Offer: &r}, nil
	}
	if r.IsExpired {
		return &ValidateResult{Valid: false, Reason: "expired", Offer: &r}, nil
	}
	if r.UsageLimit != nil && r.UsedCount >= *r.UsageLimit {
		return &ValidateResult{Valid: false, Reason: "usage_exhausted", Offer: &r}, nil
	}
	if sportID != nil && *sportID > 0 && r.SportID != nil && *r.SportID != *sportID {
		return &ValidateResult{Valid: false, Reason: "sport_mismatch", Offer: &r}, nil
	}
	return &ValidateResult{
		Valid:         true,
		Offer:         &r,
		DiscountType:  r.DiscountType,
		DiscountValue: r.DiscountValue,
	}, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanOffer(row scannable) (OfferRow, error) {
	var r OfferRow
	var limit sql.NullInt64
	var sport sql.NullInt64
	err := row.Scan(
		&r.ID, &r.FacilityID, &r.Code, &r.Title, &r.Description,
		&r.OfferKind, &r.DiscountType, &r.DiscountValue,
		&r.ValidFrom, &r.ValidTo, &limit, &r.UsedCount, &sport, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return r, err
	}
	if limit.Valid {
		v := int(limit.Int64)
		r.UsageLimit = &v
	}
	if sport.Valid {
		v := sport.Int64
		r.SportID = &v
	}
	return r, nil
}

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	var b strings.Builder
	for _, r := range code {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

func validateDate(date string) error {
	date = strings.TrimSpace(date)
	if date == "" {
		return fmt.Errorf("date_required")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid_date")
	}
	return nil
}
