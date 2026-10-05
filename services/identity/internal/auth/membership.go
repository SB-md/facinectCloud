package auth

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// RoleDefaults mirrors Facinect PageAccess::roleDefaults.
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
	case "tour_admin":
		return []string{"tournaments"}
	default:
		return []string{"dashboard"}
	}
}

func ResolvePageAccess(role string, raw sql.NullString) []string {
	if raw.Valid && strings.TrimSpace(raw.String) != "" && raw.String != "null" {
		var keys []string
		if err := json.Unmarshal([]byte(raw.String), &keys); err == nil && len(keys) > 0 {
			return keys
		}
	}
	return RoleDefaults(role)
}

type FacilityMembership struct {
	FacilityID   int64
	FacilityName string
	Slug         string
	Role         string
	PageAccess   []string
}

func (m FacilityMembership) Public() map[string]interface{} {
	return map[string]interface{}{
		"facilityId":   m.FacilityID,
		"facilityName": m.FacilityName,
		"slug":         m.Slug,
		"role":         m.Role,
		"page_access":  m.PageAccess,
	}
}

// RedirectPath mirrors Facinect WebAuthLogin::completeLogin role paths.
func RedirectPath(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "sub_admin":
		return "/view-bookings"
	case "headcoach", "coach":
		return "/students"
	case "tour_admin":
		return "/tournaments"
	default:
		return ""
	}
}

func FacilityHomePath(m FacilityMembership) string {
	base := "/facility/" + m.Slug
	return base + RedirectPath(m.Role)
}

func (s *Service) LoadMemberships(userID int64) ([]FacilityMembership, error) {
	rows, err := s.DB.Query(
		`SELECT f.id, f.name, f.slug, m.role, m.page_access::text
		 FROM user_facility_memberships m
		 INNER JOIN facilities f ON f.id = m.facility_id
		 WHERE m.user_id = $1 AND m.status = 'active' AND f.status = 'active'
		 ORDER BY m.id ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]FacilityMembership, 0)
	for rows.Next() {
		var m FacilityMembership
		var pageAccess sql.NullString
		if err := rows.Scan(&m.FacilityID, &m.FacilityName, &m.Slug, &m.Role, &pageAccess); err != nil {
			return nil, err
		}
		m.PageAccess = ResolvePageAccess(m.Role, pageAccess)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) BootstrapDemoFacility(userID int64) error {
	var fid int64
	err := s.DB.QueryRow(`SELECT id FROM facilities WHERE slug = 'demo-arena' LIMIT 1`).Scan(&fid)
	if err == sql.ErrNoRows {
		err = s.DB.QueryRow(
			`INSERT INTO facilities (name, slug, status) VALUES ($1, $2, 'active') RETURNING id`,
			"Demo Arena", "demo-arena",
		).Scan(&fid)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	_, err = s.DB.Exec(
		`INSERT INTO user_facility_memberships (user_id, facility_id, role, page_access, status)
		 VALUES ($1, $2, 'admin', NULL, 'active')
		 ON CONFLICT (user_id, facility_id) DO NOTHING`,
		userID, fid,
	)
	return err
}
