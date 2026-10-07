package addfacility

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/facinect/addfacility/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

type SportSpec struct {
	Name   string      `json:"name"`
	Courts []CourtSpec `json:"courts"`
}

type CourtSpec struct {
	Name         string   `json:"name"`
	PricePerHour *float64 `json:"price_per_hour,omitempty"`
	OpenTime     string   `json:"open_time,omitempty"`
	CloseTime    string   `json:"close_time,omitempty"`
}

type SubmitInput struct {
	FacilityName    string      `json:"facility_name"`
	Location        string      `json:"location"`
	ContactName     string      `json:"contact_name"`
	WhatsApp        string      `json:"whatsapp"`
	AlternateNumber string      `json:"alternate_number"`
	GoogleMapURL    string      `json:"google_map_url"`
	Notes           string      `json:"notes"`
	Sports          []SportSpec `json:"sports"`
	// Local/service-key testing only
	RequesterUserID int64  `json:"requester_user_id,omitempty"`
	RequesterEmail  string `json:"requester_email,omitempty"`
}

type ReviewInput struct {
	Note string `json:"note"`
}

type RequestRow struct {
	ID              int64       `json:"id"`
	RequesterUserID int64       `json:"requester_user_id"`
	RequesterEmail  string      `json:"requester_email"`
	FacilityName    string      `json:"facility_name"`
	Location        string      `json:"location"`
	ContactName     string      `json:"contact_name,omitempty"`
	WhatsApp        string      `json:"whatsapp,omitempty"`
	AlternateNumber string      `json:"alternate_number,omitempty"`
	GoogleMapURL    string      `json:"google_map_url,omitempty"`
	Notes           string      `json:"notes,omitempty"`
	Sports          []SportSpec `json:"sports,omitempty"`
	Status          string      `json:"status"`
	FacilityID      *int64      `json:"facility_id,omitempty"`
	FacilitySlug    string      `json:"facility_slug,omitempty"`
	ReviewedBy      *int64      `json:"reviewed_by,omitempty"`
	ReviewedAt      string      `json:"reviewed_at,omitempty"`
	ReviewNote      string      `json:"review_note,omitempty"`
	CreatedAt       string      `json:"created_at,omitempty"`
}

func (s *Service) Submit(ctx context.Context, userID int64, email string, in SubmitInput) (*RequestRow, error) {
	if userID <= 0 {
		userID = in.RequesterUserID
	}
	if email == "" {
		email = strings.ToLower(strings.TrimSpace(in.RequesterEmail))
	}
	if userID <= 0 {
		return nil, fmt.Errorf("requester_required")
	}
	if email == "" {
		_ = s.DB.QueryRowContext(ctx, `SELECT lower(email) FROM users WHERE id=$1`, userID).Scan(&email)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	name := strings.TrimSpace(in.FacilityName)
	loc := strings.TrimSpace(in.Location)
	if name == "" || loc == "" {
		return nil, fmt.Errorf("facility_name_and_location_required")
	}
	mapURL := strings.TrimSpace(in.GoogleMapURL)
	if mapURL != "" && !isValidMapsURL(mapURL) {
		return nil, fmt.Errorf("invalid_google_map_url")
	}
	wa := digitsOnly(in.WhatsApp)
	if wa != "" && len(wa) < 8 {
		return nil, fmt.Errorf("invalid_whatsapp")
	}
	sports := in.Sports
	if len(sports) == 0 {
		sports = []SportSpec{{Name: "General", Courts: []CourtSpec{{Name: "Court 1", OpenTime: "06:00", CloseTime: "22:00"}}}}
	}
	for i := range sports {
		sports[i].Name = strings.TrimSpace(sports[i].Name)
		if sports[i].Name == "" {
			return nil, fmt.Errorf("invalid_sport")
		}
		if len(sports[i].Courts) == 0 {
			sports[i].Courts = []CourtSpec{{Name: "Court 1", OpenTime: "06:00", CloseTime: "22:00"}}
		}
	}
	sportsJSON, err := json.Marshal(sports)
	if err != nil {
		return nil, err
	}
	contact := strings.TrimSpace(in.ContactName)
	if contact == "" {
		contact = name
	}

	var id int64
	err = s.DB.QueryRowContext(ctx, `
INSERT INTO facility_requests
  (requester_user_id, requester_email, facility_name, location, contact_name, whatsapp,
   alternate_number, google_map_url, notes, sports_json, status)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),$10::jsonb,'pending')
RETURNING id`,
		userID, email, name, loc, contact, strings.TrimSpace(in.WhatsApp),
		strings.TrimSpace(in.AlternateNumber), mapURL, strings.TrimSpace(in.Notes), string(sportsJSON),
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) ListMine(ctx context.Context, userID int64, status string) ([]RequestRow, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("requester_required")
	}
	q := `
SELECT id, requester_user_id, requester_email, facility_name, location,
       COALESCE(contact_name,''), COALESCE(whatsapp,''), COALESCE(alternate_number,''),
       COALESCE(google_map_url,''), COALESCE(notes,''), COALESCE(sports_json::text,'[]'),
       status, facility_id, reviewed_by, COALESCE(reviewed_at::text,''), COALESCE(review_note,''),
       created_at::text
FROM facility_requests WHERE requester_user_id=$1`
	args := []interface{}{userID}
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "" && st != "all" {
		q += ` AND status=$2`
		args = append(args, st)
	}
	q += ` ORDER BY created_at DESC`
	return s.queryList(ctx, q, args...)
}

func (s *Service) ListPending(ctx context.Context) ([]RequestRow, error) {
	return s.queryList(ctx, `
SELECT id, requester_user_id, requester_email, facility_name, location,
       COALESCE(contact_name,''), COALESCE(whatsapp,''), COALESCE(alternate_number,''),
       COALESCE(google_map_url,''), COALESCE(notes,''), COALESCE(sports_json::text,'[]'),
       status, facility_id, reviewed_by, COALESCE(reviewed_at::text,''), COALESCE(review_note,''),
       created_at::text
FROM facility_requests WHERE status='pending'
ORDER BY created_at ASC`)
}

func (s *Service) Get(ctx context.Context, id int64) (*RequestRow, error) {
	list, err := s.queryList(ctx, `
SELECT id, requester_user_id, requester_email, facility_name, location,
       COALESCE(contact_name,''), COALESCE(whatsapp,''), COALESCE(alternate_number,''),
       COALESCE(google_map_url,''), COALESCE(notes,''), COALESCE(sports_json::text,'[]'),
       status, facility_id, reviewed_by, COALESCE(reviewed_at::text,''), COALESCE(review_note,''),
       created_at::text
FROM facility_requests WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("not_found")
	}
	row := list[0]
	if row.FacilityID != nil {
		var slug string
		_ = s.DB.QueryRowContext(ctx, `SELECT slug FROM facilities WHERE id=$1`, *row.FacilityID).Scan(&slug)
		row.FacilitySlug = slug
	}
	return &row, nil
}

func (s *Service) Approve(ctx context.Context, requestID, reviewerID int64, note string) (*RequestRow, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var row RequestRow
	var sportsRaw string
	var facilityID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT id, requester_user_id, requester_email, facility_name, location,
       COALESCE(contact_name,''), COALESCE(whatsapp,''), COALESCE(alternate_number,''),
       COALESCE(google_map_url,''), COALESCE(notes,''), COALESCE(sports_json::text,'[]'), status, facility_id
FROM facility_requests WHERE id=$1 FOR UPDATE`, requestID).Scan(
		&row.ID, &row.RequesterUserID, &row.RequesterEmail, &row.FacilityName, &row.Location,
		&row.ContactName, &row.WhatsApp, &row.AlternateNumber, &row.GoogleMapURL, &row.Notes,
		&sportsRaw, &row.Status, &facilityID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not_found")
	}
	if err != nil {
		return nil, err
	}
	if row.Status != "pending" {
		return nil, fmt.Errorf("not_pending")
	}

	_ = json.Unmarshal([]byte(sportsRaw), &row.Sports)
	slug, err := uniqueSlug(ctx, tx, row.FacilityName)
	if err != nil {
		return nil, err
	}

	var newFacilityID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO facilities (name, slug, status) VALUES ($1,$2,'active') RETURNING id`,
		row.FacilityName, slug,
	).Scan(&newFacilityID)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO user_facility_memberships (user_id, facility_id, role, page_access, status)
VALUES ($1,$2,'admin',NULL,'active')
ON CONFLICT (user_id, facility_id) DO UPDATE SET role='admin', status='active', updated_at=NOW()`,
		row.RequesterUserID, newFacilityID,
	)
	if err != nil {
		return nil, err
	}

	// Best-effort seed administration tables if present (same DB).
	_, _ = tx.ExecContext(ctx, `
INSERT INTO facility_profiles (facility_id, display_name, location, phone, whatsapp, map_url, notes, updated_at)
VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NOW())
ON CONFLICT (facility_id) DO NOTHING`,
		newFacilityID, row.FacilityName, row.Location, row.AlternateNumber, row.WhatsApp, row.GoogleMapURL, row.Notes,
	)
	for _, sp := range row.Sports {
		var sportID int64
		err = tx.QueryRowContext(ctx, `
INSERT INTO facility_sports (facility_id, name, status)
VALUES ($1,$2,'active')
ON CONFLICT (facility_id, name) DO UPDATE SET status='active', updated_at=NOW()
RETURNING id`, newFacilityID, sp.Name).Scan(&sportID)
		if err != nil {
			continue
		}
		for _, c := range sp.Courts {
			cname := strings.TrimSpace(c.Name)
			if cname == "" {
				cname = "Court 1"
			}
			var price interface{}
			if c.PricePerHour != nil {
				price = *c.PricePerHour
			}
			_, _ = tx.ExecContext(ctx, `
INSERT INTO facility_courts (facility_id, sport_id, name, price_per_hour, open_time, close_time, status)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),'active')`,
				newFacilityID, sportID, cname, price,
				strings.TrimSpace(c.OpenTime), strings.TrimSpace(c.CloseTime),
			)
		}
	}
	_, _ = tx.ExecContext(ctx, `
INSERT INTO facility_service_flags (facility_id, staff_enabled, customer_enabled, updated_at)
VALUES ($1,TRUE,TRUE,NOW()) ON CONFLICT (facility_id) DO NOTHING`, newFacilityID)

	var reviewer interface{}
	if reviewerID > 0 {
		reviewer = reviewerID
	}
	_, err = tx.ExecContext(ctx, `
UPDATE facility_requests
SET status='approved', facility_id=$1, reviewed_by=$2, reviewed_at=NOW(),
    review_note=NULLIF($3,''), updated_at=NOW()
WHERE id=$4`, newFacilityID, reviewer, strings.TrimSpace(note), requestID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, requestID)
}

func (s *Service) Reject(ctx context.Context, requestID, reviewerID int64, note string) (*RequestRow, error) {
	var reviewer interface{}
	if reviewerID > 0 {
		reviewer = reviewerID
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE facility_requests
SET status='rejected', reviewed_by=$1, reviewed_at=NOW(),
    review_note=NULLIF($2,''), updated_at=NOW()
WHERE id=$3 AND status='pending'`, reviewer, strings.TrimSpace(note), requestID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found_or_not_pending")
	}
	return s.Get(ctx, requestID)
}

func (s *Service) Summary(ctx context.Context, userID int64) (map[string]int, error) {
	out := map[string]int{"pending": 0, "approved": 0, "rejected": 0}
	rows, err := s.DB.QueryContext(ctx, `
SELECT status, COUNT(*) FROM facility_requests
WHERE requester_user_id=$1 GROUP BY status`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

func (s *Service) queryList(ctx context.Context, q string, args ...interface{}) ([]RequestRow, error) {
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RequestRow{}
	for rows.Next() {
		var r RequestRow
		var sportsRaw string
		var facilityID, reviewedBy sql.NullInt64
		if err := rows.Scan(
			&r.ID, &r.RequesterUserID, &r.RequesterEmail, &r.FacilityName, &r.Location,
			&r.ContactName, &r.WhatsApp, &r.AlternateNumber, &r.GoogleMapURL, &r.Notes,
			&sportsRaw, &r.Status, &facilityID, &reviewedBy, &r.ReviewedAt, &r.ReviewNote, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(sportsRaw), &r.Sports)
		if facilityID.Valid {
			v := facilityID.Int64
			r.FacilityID = &v
		}
		if reviewedBy.Valid {
			v := reviewedBy.Int64
			r.ReviewedBy = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func uniqueSlug(ctx context.Context, tx *sql.Tx, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "facility"
	}
	slug := base
	for i := 0; i < 50; i++ {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM facilities WHERE slug=$1 LIMIT 1`, slug).Scan(&exists)
		if err == sql.ErrNoRows {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		slug = fmt.Sprintf("%s-%d", base, i+2)
	}
	return "", fmt.Errorf("slug_exhausted")
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = out[:80]
		out = strings.Trim(out, "-")
	}
	return out
}

var mapsURLRe = regexp.MustCompile(`(?i)(maps\.app\.goo\.gl|google\.[^/]+/maps|goo\.gl/maps|maps\.google\.)`)

func isValidMapsURL(u string) bool {
	u = strings.TrimSpace(u)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return false
	}
	return mapsURLRe.MatchString(u)
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
