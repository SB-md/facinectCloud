package tournaments

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/facinect/tournaments/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

type Tournament struct {
	ID             int64  `json:"id"`
	FacilityID     int64  `json:"facility_id"`
	SportID        *int64 `json:"sport_id,omitempty"`
	Name           string `json:"name"`
	StartDate      string `json:"start_date,omitempty"`
	EndDate        string `json:"end_date,omitempty"`
	VenueAddress   string `json:"venue_address,omitempty"`
	LocationURL    string `json:"location_url,omitempty"`
	ContactNumbers string `json:"contact_numbers,omitempty"`
	Rules          string `json:"rules,omitempty"`
	Slug           string `json:"slug,omitempty"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type UpsertInput struct {
	Name           string `json:"name"`
	SportID        *int64 `json:"sport_id"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	VenueAddress   string `json:"venue_address"`
	LocationURL    string `json:"location_url"`
	ContactNumbers string `json:"contact_numbers"`
	Rules          string `json:"rules"`
	Slug           string `json:"slug"`
	Status         string `json:"status"`
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) List(ctx context.Context, facilityID int64, status string) ([]Tournament, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	q := `
SELECT id, facility_id, sport_id, name,
       COALESCE(start_date::text,''), COALESCE(end_date::text,''),
       COALESCE(venue_address,''), COALESCE(location_url,''),
       COALESCE(contact_numbers,''), COALESCE(rules,''), COALESCE(slug,''),
       status, created_at::text, updated_at::text
FROM tournaments WHERE facility_id=$1`
	args := []interface{}{facilityID}
	st := strings.TrimSpace(strings.ToLower(status))
	if st != "" && st != "all" {
		q += ` AND status=$2`
		args = append(args, st)
	}
	q += ` ORDER BY COALESCE(start_date, DATE '1970-01-01') DESC, id DESC`

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tournament{}
	for rows.Next() {
		t, err := scanTournament(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Service) CountActive(ctx context.Context, facilityID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM tournaments
WHERE facility_id=$1 AND status IN ('draft','upcoming','live')`, facilityID).Scan(&n)
	return n, err
}

func (s *Service) Get(ctx context.Context, id int64) (*Tournament, error) {
	row := s.DB.QueryRowContext(ctx, `
SELECT id, facility_id, sport_id, name,
       COALESCE(start_date::text,''), COALESCE(end_date::text,''),
       COALESCE(venue_address,''), COALESCE(location_url,''),
       COALESCE(contact_numbers,''), COALESCE(rules,''), COALESCE(slug,''),
       status, created_at::text, updated_at::text
FROM tournaments WHERE id=$1`, id)
	return scanTournament(row)
}

func (s *Service) Create(ctx context.Context, facilityID int64, in UpsertInput) (*Tournament, error) {
	name := strings.TrimSpace(in.Name)
	if facilityID <= 0 || name == "" {
		return nil, fmt.Errorf("invalid_tournament")
	}
	if err := validateOptionalDates(in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	status := normalizeStatus(in.Status, "draft")
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(name)
	} else {
		slug = slugify(slug)
	}

	var sport, start, end interface{}
	if in.SportID != nil {
		sport = *in.SportID
	}
	if strings.TrimSpace(in.StartDate) != "" {
		start = in.StartDate
	}
	if strings.TrimSpace(in.EndDate) != "" {
		end = in.EndDate
	}

	var id int64
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO tournaments
  (facility_id, sport_id, name, start_date, end_date, venue_address, location_url,
   contact_numbers, rules, slug, status)
VALUES ($1,$2,$3,$4::date,$5::date,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),$11)
RETURNING id`,
		facilityID, sport, name, start, end,
		strings.TrimSpace(in.VenueAddress),
		strings.TrimSpace(in.LocationURL),
		strings.TrimSpace(in.ContactNumbers),
		strings.TrimSpace(in.Rules),
		slug, status,
	).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "idx_tournaments_facility_slug") {
			return nil, fmt.Errorf("slug_taken")
		}
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, facilityID, id int64, in UpsertInput) (*Tournament, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.FacilityID != facilityID {
		return nil, fmt.Errorf("facility_mismatch")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = cur.Name
	}
	if err := validateOptionalDates(in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	status := normalizeStatus(in.Status, cur.Status)
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = cur.Slug
	} else {
		slug = slugify(slug)
	}

	sport := interface{}(nil)
	if in.SportID != nil {
		sport = *in.SportID
	} else if cur.SportID != nil {
		sport = *cur.SportID
	}
	start := nullableDate(in.StartDate, cur.StartDate)
	end := nullableDate(in.EndDate, cur.EndDate)

	venue := pickStr(in.VenueAddress, cur.VenueAddress)
	loc := pickStr(in.LocationURL, cur.LocationURL)
	contacts := pickStr(in.ContactNumbers, cur.ContactNumbers)
	rules := pickStr(in.Rules, cur.Rules)

	_, err = s.DB.ExecContext(ctx, `
UPDATE tournaments SET
  sport_id=$1, name=$2, start_date=$3::date, end_date=$4::date,
  venue_address=NULLIF($5,''), location_url=NULLIF($6,''),
  contact_numbers=NULLIF($7,''), rules=NULLIF($8,''), slug=NULLIF($9,''),
  status=$10, updated_at=NOW()
WHERE id=$11 AND facility_id=$12`,
		sport, name, start, end, venue, loc, contacts, rules, slug, status, id, facilityID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "idx_tournaments_facility_slug") {
			return nil, fmt.Errorf("slug_taken")
		}
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, facilityID, id int64, status string) (*Tournament, error) {
	st := normalizeStatus(status, "")
	if st == "" {
		return nil, fmt.Errorf("invalid_status")
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE tournaments SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3`, st, id, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found")
	}
	return s.Get(ctx, id)
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanTournament(row scannable) (*Tournament, error) {
	var t Tournament
	var sport sql.NullInt64
	err := row.Scan(
		&t.ID, &t.FacilityID, &sport, &t.Name,
		&t.StartDate, &t.EndDate, &t.VenueAddress, &t.LocationURL,
		&t.ContactNumbers, &t.Rules, &t.Slug, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sport.Valid {
		v := sport.Int64
		t.SportID = &v
	}
	return &t, nil
}

func normalizeStatus(status, fallback string) string {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "draft", "upcoming", "live", "completed", "cancelled":
		return st
	case "":
		return strings.ToLower(strings.TrimSpace(fallback))
	default:
		return ""
	}
}

func validateOptionalDates(start, end string) error {
	if strings.TrimSpace(start) != "" {
		if err := validateDate(start); err != nil {
			return err
		}
	}
	if strings.TrimSpace(end) != "" {
		if err := validateDate(end); err != nil {
			return err
		}
	}
	return nil
}

func validateDate(date string) error {
	date = strings.TrimSpace(date)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid_date")
	}
	return nil
}

func nullableDate(in, fallback string) interface{} {
	in = strings.TrimSpace(in)
	if in != "" {
		return in
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return nil
}

func pickStr(in, fallback string) string {
	if strings.TrimSpace(in) != "" {
		return strings.TrimSpace(in)
	}
	return fallback
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	out := nonSlug.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		return "tournament"
	}
	if len(out) > 80 {
		out = out[:80]
		out = strings.Trim(out, "-")
	}
	return out
}
