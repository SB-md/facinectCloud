package administration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/facinect/administration/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

// --- Catalog ---

func PagesCatalog() []map[string]string {
	return []map[string]string{
		{"key": "dashboard", "label": "Dashboard"},
		{"key": "bookings", "label": "Slots Setup"},
		{"key": "view_bookings", "label": "View Bookings"},
		{"key": "students", "label": "Students"},
		{"key": "members", "label": "Members"},
		{"key": "payments", "label": "Payments"},
		{"key": "tournaments", "label": "Tournaments"},
		{"key": "enquiry", "label": "Enquiry"},
		{"key": "offers", "label": "Offers"},
		{"key": "administration", "label": "Administration"},
	}
}

func RoleDefaults(role string) []string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return []string{
			"dashboard", "bookings", "view_bookings", "students", "members",
			"payments", "tournaments", "enquiry", "offers", "administration",
		}
	case "sub_admin":
		return []string{
			"dashboard", "bookings", "view_bookings", "students", "members",
			"tournaments", "enquiry",
		}
	case "headcoach":
		return []string{"students", "tournaments"}
	case "coach":
		return []string{"students"}
	case "tournament_admin":
		return []string{"tournaments"}
	default:
		return []string{"dashboard"}
	}
}

func resolvePageAccess(role string, raw sql.NullString) []string {
	if raw.Valid && strings.TrimSpace(raw.String) != "" && raw.String != "null" {
		var keys []string
		if err := json.Unmarshal([]byte(raw.String), &keys); err == nil && len(keys) > 0 {
			return keys
		}
	}
	return RoleDefaults(role)
}

// --- Profile ---

type Profile struct {
	FacilityID  int64  `json:"facility_id"`
	DisplayName string `json:"display_name"`
	Location    string `json:"location,omitempty"`
	City        string `json:"city,omitempty"`
	Address     string `json:"address,omitempty"`
	Phone       string `json:"phone,omitempty"`
	WhatsApp    string `json:"whatsapp,omitempty"`
	Email       string `json:"email,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
	MapURL      string `json:"map_url,omitempty"`
	Notes       string `json:"notes,omitempty"`
	OpenTime    string `json:"open_time,omitempty"`
	CloseTime   string `json:"close_time,omitempty"`
	Slug        string `json:"slug,omitempty"`
	Status      string `json:"status,omitempty"`
}

func (s *Service) GetProfile(ctx context.Context, facilityID int64) (*Profile, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	var p Profile
	var loc, city, addr, phone, wa, email, logo, mapURL, notes, openT, closeT sql.NullString
	err := s.DB.QueryRowContext(ctx, `
SELECT f.id, COALESCE(NULLIF(p.display_name,''), f.name), COALESCE(p.location,''), COALESCE(p.city,''),
       COALESCE(p.address,''), COALESCE(p.phone,''), COALESCE(p.whatsapp,''), COALESCE(p.email,''),
       COALESCE(p.logo_url,''), COALESCE(p.map_url,''), COALESCE(p.notes,''),
       COALESCE(p.open_time,''), COALESCE(p.close_time,''), f.slug, f.status
FROM facilities f
LEFT JOIN facility_profiles p ON p.facility_id = f.id
WHERE f.id=$1`, facilityID).Scan(
		&p.FacilityID, &p.DisplayName, &loc, &city, &addr, &phone, &wa, &email,
		&logo, &mapURL, &notes, &openT, &closeT, &p.Slug, &p.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("facility_not_found")
	}
	if err != nil {
		return nil, err
	}
	p.Location, p.City, p.Address = loc.String, city.String, addr.String
	p.Phone, p.WhatsApp, p.Email = phone.String, wa.String, email.String
	p.LogoURL, p.MapURL, p.Notes = logo.String, mapURL.String, notes.String
	p.OpenTime, p.CloseTime = openT.String, closeT.String
	return &p, nil
}

func (s *Service) UpsertProfile(ctx context.Context, facilityID int64, in Profile) (*Profile, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	name := strings.TrimSpace(in.DisplayName)
	if name == "" {
		return nil, fmt.Errorf("display_name_required")
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO facility_profiles
  (facility_id, display_name, location, city, address, phone, whatsapp, email,
   logo_url, map_url, notes, open_time, close_time, updated_at)
VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),
        NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),NOW())
ON CONFLICT (facility_id) DO UPDATE SET
  display_name=EXCLUDED.display_name,
  location=EXCLUDED.location, city=EXCLUDED.city, address=EXCLUDED.address,
  phone=EXCLUDED.phone, whatsapp=EXCLUDED.whatsapp, email=EXCLUDED.email,
  logo_url=EXCLUDED.logo_url, map_url=EXCLUDED.map_url, notes=EXCLUDED.notes,
  open_time=EXCLUDED.open_time, close_time=EXCLUDED.close_time, updated_at=NOW()`,
		facilityID, name,
		strings.TrimSpace(in.Location), strings.TrimSpace(in.City), strings.TrimSpace(in.Address),
		strings.TrimSpace(in.Phone), strings.TrimSpace(in.WhatsApp), strings.TrimSpace(in.Email),
		strings.TrimSpace(in.LogoURL), strings.TrimSpace(in.MapURL), strings.TrimSpace(in.Notes),
		strings.TrimSpace(in.OpenTime), strings.TrimSpace(in.CloseTime),
	)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.ExecContext(ctx, `UPDATE facilities SET name=$1, updated_at=NOW() WHERE id=$2`, name, facilityID)
	return s.GetProfile(ctx, facilityID)
}

// --- Sports ---

type Sport struct {
	ID         int64  `json:"id"`
	FacilityID int64  `json:"facility_id"`
	Name       string `json:"name"`
	SortOrder  int    `json:"sort_order"`
	Status     string `json:"status"`
}

type SportInput struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

func (s *Service) ListSports(ctx context.Context, facilityID int64, status string) ([]Sport, error) {
	q := `SELECT id, facility_id, name, sort_order, status FROM facility_sports WHERE facility_id=$1`
	args := []interface{}{facilityID}
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "" && st != "all" {
		q += ` AND status=$2`
		args = append(args, st)
	}
	q += ` ORDER BY sort_order, name`
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Sport{}
	for rows.Next() {
		var r Sport
		if err := rows.Scan(&r.ID, &r.FacilityID, &r.Name, &r.SortOrder, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) CreateSport(ctx context.Context, facilityID int64, in SportInput) (*Sport, error) {
	name := strings.TrimSpace(in.Name)
	if facilityID <= 0 || name == "" {
		return nil, fmt.Errorf("invalid_sport")
	}
	var r Sport
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO facility_sports (facility_id, name, sort_order, status)
VALUES ($1,$2,$3,'active') RETURNING id, facility_id, name, sort_order, status`,
		facilityID, name, in.SortOrder,
	).Scan(&r.ID, &r.FacilityID, &r.Name, &r.SortOrder, &r.Status)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("sport_exists")
		}
		return nil, err
	}
	return &r, nil
}

func (s *Service) UpdateSportStatus(ctx context.Context, facilityID, sportID int64, status string) (*Sport, error) {
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "active" && st != "inactive" {
		return nil, fmt.Errorf("invalid_status")
	}
	var r Sport
	err := s.DB.QueryRowContext(ctx, `
UPDATE facility_sports SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3
RETURNING id, facility_id, name, sort_order, status`, st, sportID, facilityID).Scan(
		&r.ID, &r.FacilityID, &r.Name, &r.SortOrder, &r.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not_found")
	}
	return &r, err
}

// --- Courts ---

type Court struct {
	ID           int64    `json:"id"`
	FacilityID   int64    `json:"facility_id"`
	SportID      *int64   `json:"sport_id,omitempty"`
	Name         string   `json:"name"`
	PricePerHour *float64 `json:"price_per_hour,omitempty"`
	OpenTime     string   `json:"open_time,omitempty"`
	CloseTime    string   `json:"close_time,omitempty"`
	Status       string   `json:"status"`
}

type CourtInput struct {
	SportID      *int64   `json:"sport_id"`
	Name         string   `json:"name"`
	PricePerHour *float64 `json:"price_per_hour"`
	OpenTime     string   `json:"open_time"`
	CloseTime    string   `json:"close_time"`
}

func (s *Service) ListCourts(ctx context.Context, facilityID int64, status string) ([]Court, error) {
	q := `
SELECT id, facility_id, sport_id, name, price_per_hour,
       COALESCE(open_time,''), COALESCE(close_time,''), status
FROM facility_courts WHERE facility_id=$1`
	args := []interface{}{facilityID}
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "" && st != "all" {
		q += ` AND status=$2`
		args = append(args, st)
	}
	q += ` ORDER BY name`
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Court{}
	for rows.Next() {
		c, err := scanCourt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) CreateCourt(ctx context.Context, facilityID int64, in CourtInput) (*Court, error) {
	name := strings.TrimSpace(in.Name)
	if facilityID <= 0 || name == "" {
		return nil, fmt.Errorf("invalid_court")
	}
	var sport, price interface{}
	if in.SportID != nil && *in.SportID > 0 {
		sport = *in.SportID
	}
	if in.PricePerHour != nil {
		price = *in.PricePerHour
	}
	row := s.DB.QueryRowContext(ctx, `
INSERT INTO facility_courts
  (facility_id, sport_id, name, price_per_hour, open_time, close_time, status)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),'active')
RETURNING id, facility_id, sport_id, name, price_per_hour,
          COALESCE(open_time,''), COALESCE(close_time,''), status`,
		facilityID, sport, name, price,
		strings.TrimSpace(in.OpenTime), strings.TrimSpace(in.CloseTime),
	)
	c, err := scanCourt(row)
	return &c, err
}

func (s *Service) UpdateCourtStatus(ctx context.Context, facilityID, courtID int64, status string) (*Court, error) {
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "active" && st != "inactive" {
		return nil, fmt.Errorf("invalid_status")
	}
	row := s.DB.QueryRowContext(ctx, `
UPDATE facility_courts SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3
RETURNING id, facility_id, sport_id, name, price_per_hour,
          COALESCE(open_time,''), COALESCE(close_time,''), status`, st, courtID, facilityID)
	c, err := scanCourt(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not_found")
	}
	return &c, err
}

type courtScanner interface {
	Scan(dest ...interface{}) error
}

func scanCourt(row courtScanner) (Court, error) {
	var c Court
	var sport sql.NullInt64
	var price sql.NullFloat64
	err := row.Scan(&c.ID, &c.FacilityID, &sport, &c.Name, &price, &c.OpenTime, &c.CloseTime, &c.Status)
	if err != nil {
		return c, err
	}
	if sport.Valid {
		v := sport.Int64
		c.SportID = &v
	}
	if price.Valid {
		v := price.Float64
		c.PricePerHour = &v
	}
	return c, nil
}

// --- Payments ---

type PaymentSetting struct {
	ID                  int64   `json:"id,omitempty"`
	FacilityID          int64   `json:"facility_id"`
	SportID             int64   `json:"sport_id"` // 0 = facility-wide default
	AllowFullPayment    bool    `json:"allow_full_payment"`
	AllowAdvancePayment bool    `json:"allow_advance_payment"`
	AllowSpotPayment    bool    `json:"allow_spot_payment"`
	AdvanceAmount       float64 `json:"advance_amount"`
}

func (s *Service) ListPayments(ctx context.Context, facilityID int64) ([]PaymentSetting, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, facility_id, sport_id, allow_full_payment, allow_advance_payment,
       allow_spot_payment, advance_amount
FROM facility_sport_payments WHERE facility_id=$1 ORDER BY sport_id`, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PaymentSetting{}
	for rows.Next() {
		var p PaymentSetting
		if err := rows.Scan(&p.ID, &p.FacilityID, &p.SportID, &p.AllowFullPayment, &p.AllowAdvancePayment,
			&p.AllowSpotPayment, &p.AdvanceAmount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		out = append(out, PaymentSetting{
			FacilityID:          facilityID,
			SportID:             0,
			AllowFullPayment:    true,
			AllowAdvancePayment: true,
			AllowSpotPayment:    false,
		})
	}
	return out, rows.Err()
}

func (s *Service) UpsertPayment(ctx context.Context, facilityID int64, in PaymentSetting) (*PaymentSetting, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	sportID := in.SportID
	if sportID < 0 {
		sportID = 0
	}
	var p PaymentSetting
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO facility_sport_payments
  (facility_id, sport_id, allow_full_payment, allow_advance_payment, allow_spot_payment, advance_amount)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (facility_id, sport_id) DO UPDATE SET
  allow_full_payment=EXCLUDED.allow_full_payment,
  allow_advance_payment=EXCLUDED.allow_advance_payment,
  allow_spot_payment=EXCLUDED.allow_spot_payment,
  advance_amount=EXCLUDED.advance_amount,
  updated_at=NOW()
RETURNING id, facility_id, sport_id, allow_full_payment, allow_advance_payment,
          allow_spot_payment, advance_amount`,
		facilityID, sportID, in.AllowFullPayment, in.AllowAdvancePayment, in.AllowSpotPayment, in.AdvanceAmount,
	).Scan(&p.ID, &p.FacilityID, &p.SportID, &p.AllowFullPayment, &p.AllowAdvancePayment, &p.AllowSpotPayment, &p.AdvanceAmount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- Services / maintenance ---

type ServiceFlags struct {
	FacilityID       int64 `json:"facility_id"`
	StaffEnabled     bool  `json:"staff_enabled"`
	CustomerEnabled  bool  `json:"customer_enabled"`
}

func (s *Service) GetServices(ctx context.Context, facilityID int64) (*ServiceFlags, error) {
	var f ServiceFlags
	err := s.DB.QueryRowContext(ctx, `
SELECT facility_id, staff_enabled, customer_enabled FROM facility_service_flags WHERE facility_id=$1`,
		facilityID).Scan(&f.FacilityID, &f.StaffEnabled, &f.CustomerEnabled)
	if err == sql.ErrNoRows {
		return &ServiceFlags{FacilityID: facilityID, StaffEnabled: true, CustomerEnabled: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Service) UpsertServices(ctx context.Context, facilityID int64, in ServiceFlags) (*ServiceFlags, error) {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO facility_service_flags (facility_id, staff_enabled, customer_enabled, updated_at)
VALUES ($1,$2,$3,NOW())
ON CONFLICT (facility_id) DO UPDATE SET
  staff_enabled=EXCLUDED.staff_enabled,
  customer_enabled=EXCLUDED.customer_enabled,
  updated_at=NOW()`, facilityID, in.StaffEnabled, in.CustomerEnabled)
	if err != nil {
		return nil, err
	}
	return s.GetServices(ctx, facilityID)
}

// --- Staff (identity tables) ---

type StaffRow struct {
	MembershipID int64    `json:"membership_id"`
	UserID       int64    `json:"user_id"`
	FacilityID   int64    `json:"facility_id"`
	Email        string   `json:"email"`
	Name         string   `json:"name,omitempty"`
	Role         string   `json:"role"`
	PageAccess   []string `json:"page_access"`
	Status       string   `json:"status"`
}

type StaffInput struct {
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	PageAccess []string `json:"page_access"`
}

func (s *Service) ListStaff(ctx context.Context, facilityID int64) ([]StaffRow, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT m.id, u.id, m.facility_id, u.email, COALESCE(u.full_name,''), m.role, m.page_access::text, m.status
FROM user_facility_memberships m
JOIN users u ON u.id = m.user_id
WHERE m.facility_id=$1
ORDER BY u.email`, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StaffRow{}
	for rows.Next() {
		var r StaffRow
		var pageRaw sql.NullString
		if err := rows.Scan(&r.MembershipID, &r.UserID, &r.FacilityID, &r.Email, &r.Name, &r.Role, &pageRaw, &r.Status); err != nil {
			return nil, err
		}
		r.PageAccess = resolvePageAccess(r.Role, pageRaw)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) UpsertStaff(ctx context.Context, facilityID int64, in StaffInput) (*StaffRow, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if facilityID <= 0 || email == "" {
		return nil, fmt.Errorf("invalid_staff")
	}
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if role == "" {
		role = "coach"
	}
	allowed := map[string]bool{
		"admin": true, "sub_admin": true, "headcoach": true, "coach": true, "tournament_admin": true,
	}
	if !allowed[role] {
		return nil, fmt.Errorf("invalid_role")
	}

	var userID int64
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email)=$1`, email).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user_not_found")
	}
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		_, _ = s.DB.ExecContext(ctx, `UPDATE users SET full_name=$1, updated_at=NOW() WHERE id=$2`, name, userID)
	}

	var pageJSON interface{}
	if len(in.PageAccess) > 0 {
		b, err := json.Marshal(in.PageAccess)
		if err != nil {
			return nil, err
		}
		pageJSON = string(b)
	}

	var membershipID int64
	err = s.DB.QueryRowContext(ctx, `
INSERT INTO user_facility_memberships (user_id, facility_id, role, page_access, status)
VALUES ($1,$2,$3,$4::jsonb,'active')
ON CONFLICT (user_id, facility_id) DO UPDATE SET
  role=EXCLUDED.role,
  page_access=EXCLUDED.page_access,
  status='active',
  updated_at=NOW()
RETURNING id`, userID, facilityID, role, pageJSON).Scan(&membershipID)
	if err != nil {
		return nil, err
	}
	return s.getStaff(ctx, membershipID)
}

func (s *Service) UpdateStaffStatus(ctx context.Context, facilityID, membershipID int64, status string) (*StaffRow, error) {
	st := strings.ToLower(strings.TrimSpace(status))
	if st != "active" && st != "disabled" {
		return nil, fmt.Errorf("invalid_status")
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE user_facility_memberships SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3`, st, membershipID, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found")
	}
	return s.getStaff(ctx, membershipID)
}

func (s *Service) getStaff(ctx context.Context, membershipID int64) (*StaffRow, error) {
	var r StaffRow
	var pageRaw sql.NullString
	err := s.DB.QueryRowContext(ctx, `
SELECT m.id, u.id, m.facility_id, u.email, COALESCE(u.full_name,''), m.role, m.page_access::text, m.status
FROM user_facility_memberships m
JOIN users u ON u.id = m.user_id
WHERE m.id=$1`, membershipID).Scan(
		&r.MembershipID, &r.UserID, &r.FacilityID, &r.Email, &r.Name, &r.Role, &pageRaw, &r.Status,
	)
	if err != nil {
		return nil, err
	}
	r.PageAccess = resolvePageAccess(r.Role, pageRaw)
	return &r, nil
}

// --- Summary ---

type Summary struct {
	SportsActive   int  `json:"sports_active"`
	CourtsActive   int  `json:"courts_active"`
	StaffActive    int  `json:"staff_active"`
	StaffEnabled   bool `json:"staff_enabled"`
	CustomerEnabled bool `json:"customer_enabled"`
}

func (s *Service) Summary(ctx context.Context, facilityID int64) (*Summary, error) {
	var out Summary
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM facility_sports WHERE facility_id=$1 AND status='active'`, facilityID).Scan(&out.SportsActive)
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM facility_courts WHERE facility_id=$1 AND status='active'`, facilityID).Scan(&out.CourtsActive)
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_facility_memberships WHERE facility_id=$1 AND status='active'`, facilityID).Scan(&out.StaffActive)
	flags, err := s.GetServices(ctx, facilityID)
	if err != nil {
		return nil, err
	}
	out.StaffEnabled = flags.StaffEnabled
	out.CustomerEnabled = flags.CustomerEnabled
	return &out, nil
}
