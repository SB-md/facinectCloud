package app

import (
	"database/sql"
	"time"
)

func (s *Service) Seed(db *sql.DB) error {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM app_facilities`).Scan(&n)
	if n > 0 {
		return nil
	}

	_, err := db.Exec(`
INSERT INTO app_sports (id, name) OVERRIDING SYSTEM VALUE VALUES
  (1, 'Badminton'), (2, 'Tennis'), (3, 'Pickleball');

INSERT INTO app_cities (id, name, state_name, latitude, longitude) OVERRIDING SYSTEM VALUE VALUES
  (1, 'Chennai', 'Tamil Nadu', 13.0827, 80.2707);

INSERT INTO app_areas (city_id, name, latitude, longitude, aliases) VALUES
  (1, 'Anna Nagar', 13.0850, 80.2100, '{}'),
  (1, 'Velachery', 12.9750, 80.2210, '{}');

INSERT INTO app_facilities (id, name, location, city_name, logo_url, latitude, longitude, customer_ok)
OVERRIDING SYSTEM VALUE VALUES
  (1, 'Demo Arena', 'Anna Nagar, Chennai', 'Chennai', '', 13.0850, 80.2101, TRUE),
  (29, 'North Court Hub', 'Velachery, Chennai', 'Chennai', '', 12.9755, 80.2215, TRUE);

INSERT INTO app_facility_sports (facility_id, sport_id) VALUES
  (1,1),(1,2),(29,1),(29,3);

INSERT INTO app_price_list (facility_id, sport_id, court_name, day_type, price, advance_amount) VALUES
  (1, 1, 'Court 1', 'weekday', 500, 100),
  (1, 1, 'Court 1', 'weekend', 700, 150),
  (1, 1, 'Court 2', 'weekday', 500, 100),
  (29, 1, 'Court A', 'weekday', 400, 80);

INSERT INTO app_coupons (facility_id, code, value, ctype, label, description) VALUES
  (1, 'WELCOME50', 50, 'flat', 'Welcome ₹50', 'Flat discount'),
  (1, 'SAVE10', 10, 'percent', '10% off', 'Percent discount');
`)
	if err != nil {
		return err
	}

	today := time.Now()
	for d := 0; d < 7; d++ {
		day := today.AddDate(0, 0, d).Format("2006-01-02")
		for _, tr := range []string{"06:00-07:00", "07:00-08:00", "18:00-19:00", "19:00-20:00", "20:00-21:00"} {
			_, _ = db.Exec(`
INSERT INTO app_slots (facility_id, sport_id, court_name, slot_date, time_range, status)
VALUES (1, 1, 'Court 1', $1::date, $2, 'available'),
       (1, 1, 'Court 2', $1::date, $2, 'available'),
       (29, 1, 'Court A', $1::date, $2, 'available')
ON CONFLICT DO NOTHING`, day, tr)
		}
	}

	_, err = db.Exec(`
INSERT INTO app_live_tournaments (id, tournament_name, tournament_date, facility_id, sport_name, categories, open_match_count)
OVERRIDING SYSTEM VALUE VALUES
  (1, 'Demo Open 2026', CURRENT_DATE, 1, 'Badminton', '{Open,Men Singles}', 2);

INSERT INTO app_live_matches (tour_id, category, player1, player2, status, court, score1, score2, version)
VALUES
  (1, 'Open', 'Player A', 'Player B', 'live', 'Court 1', 11, 9, 1),
  (1, 'Men Singles', 'Player C', 'Player D', 'upcoming', 'Court 2', 0, 0, 1);
`)
	return err
}
