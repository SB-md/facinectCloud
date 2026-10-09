package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/facinect/app/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

func strField(body map[string]any, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}

func digitsOnly(s string) string {
	re := regexp.MustCompile(`\D+`)
	return re.ReplaceAllString(s, "")
}

func normalizeWA(raw string) string {
	d := digitsOnly(raw)
	if len(d) == 10 {
		d = "91" + d
	}
	if len(d) == 11 && strings.HasPrefix(d, "0") {
		d = "91" + d[1:]
	}
	return d
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (s *Service) parseIDs(raw any) []int64 {
	out := []int64{}
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			n, _ := strconv.ParseInt(fmt.Sprint(x), 10, 64)
			if n > 0 {
				out = append(out, n)
			}
		}
	case []int64:
		return v
	}
	return out
}

func (s *Service) userRow(ctx context.Context, id int64, wa string) (map[string]any, error) {
	var (
		uid                 int64
		name, whatsapp, ph  string
		email               string
		accessed            []int64
	)
	q := `SELECT id, full_name, whatsapp_no, phone, email, COALESCE(accessed_facility_ids, '{}')::text
FROM app_users WHERE status='active' AND `
	var row *sql.Row
	if id > 0 {
		row = s.DB.QueryRowContext(ctx, q+`id=$1`, id)
	} else {
		row = s.DB.QueryRowContext(ctx, q+`whatsapp_no=$1`, normalizeWA(wa))
	}
	var arrStr string
	err := row.Scan(&uid, &name, &whatsapp, &ph, &email, &arrStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	accessed = parsePgIntArray(arrStr)
	if ph == "<nil>" {
		ph = ""
	}
	if email == "<nil>" {
		email = ""
	}
	return map[string]any{
		"exists":              true,
		"created":             false,
		"userId":              uid,
		"fullName":            name,
		"whatsappNo":          whatsapp,
		"phone":               ph,
		"email":               email,
		"accessedFacilityIds": accessed,
	}, nil
}

func parsePgIntArray(s string) []int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return []int64{}
	}
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if n > 0 {
			out = append(out, n)
		}
	}
	return out
}

func formatPgIntArray(ids []int64) string {
	if len(ids) == 0 {
		return "{}"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func (s *Service) UsersLookup(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	switch action {
	case "lookup":
		wa := normalizeWA(strField(body, "whatsappNo"))
		u, err := s.userRow(ctx, 0, wa)
		if err != nil {
			return false, err.Error(), nil
		}
		if u == nil {
			return true, "Not found", map[string]any{
				"exists": false, "created": false, "userId": 0,
				"fullName": "", "whatsappNo": wa, "phone": "", "email": "",
				"accessedFacilityIds": []int64{},
			}
		}
		return true, "OK", u
	case "register":
		wa := normalizeWA(strField(body, "whatsappNo"))
		name := strField(body, "fullName")
		phone := strField(body, "phone")
		if phone == "" {
			phone = strField(body, "altPhone")
		}
		email := strField(body, "email")
		if wa == "" || name == "" {
			return false, "Name and WhatsApp required", nil
		}
		if existing, _ := s.userRow(ctx, 0, wa); existing != nil {
			existing["created"] = false
			return true, "Already registered", existing
		}
		var id int64
		err := s.DB.QueryRowContext(ctx, `
INSERT INTO app_users (full_name, whatsapp_no, phone, email)
VALUES ($1,$2,$3,$4) RETURNING id`, name, wa, phone, email).Scan(&id)
		if err != nil {
			return false, err.Error(), nil
		}
		u, _ := s.userRow(ctx, id, "")
		if u != nil {
			u["created"] = true
		}
		return true, "Registered", u
	case "add_facility_access":
		uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
		fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
		u, err := s.userRow(ctx, uid, "")
		if err != nil || u == nil {
			return false, "User not found", nil
		}
		ids := s.parseIDs(u["accessedFacilityIds"])
		found := false
		for _, id := range ids {
			if id == fid {
				found = true
				break
			}
		}
		if !found && fid > 0 {
			ids = append(ids, fid)
			_, err = s.DB.ExecContext(ctx, `
UPDATE app_users SET accessed_facility_ids=$1::bigint[], updated_at=NOW() WHERE id=$2`,
				formatPgIntArray(ids), uid)
			if err != nil {
				return false, err.Error(), nil
			}
		}
		return true, "OK", map[string]any{"accessedFacilityIds": ids}
	case "delete_account":
		wa := normalizeWA(fmt.Sprint(body["whatsappNo"]))
		uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
		res, err := s.DB.ExecContext(ctx, `
UPDATE app_users SET status='deleted', updated_at=NOW()
WHERE ($1 > 0 AND id=$1) OR whatsapp_no=$2`, uid, wa)
		if err != nil {
			return false, err.Error(), nil
		}
		n, _ := res.RowsAffected()
		return n > 0, "Deleted", map[string]any{"deleted": n > 0}
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) FacilitiesNearby(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	switch action {
	case "sports":
		rows, err := s.DB.QueryContext(ctx, `SELECT id, name FROM app_sports WHERE status='active' ORDER BY name`)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id int64
			var name string
			_ = rows.Scan(&id, &name)
			list = append(list, map[string]any{"sportId": id, "sportName": name})
		}
		return true, "OK", map[string]any{"sports": list}
	case "cities", "all_cities":
		return s.citiesPayload(ctx)
	case "facility_sports":
		fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
		rows, err := s.DB.QueryContext(ctx, `
SELECT s.id, s.name FROM app_facility_sports fs
JOIN app_sports s ON s.id=fs.sport_id
WHERE fs.facility_id=$1 ORDER BY s.name`, fid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id int64
			var name string
			_ = rows.Scan(&id, &name)
			list = append(list, map[string]any{"sportId": id, "sportName": name})
		}
		return true, "OK", map[string]any{"sports": list}
	case "nearby", "search":
		lat, _ := strconv.ParseFloat(fmt.Sprint(body["lat"]), 64)
		lon, _ := strconv.ParseFloat(fmt.Sprint(body["lon"]), 64)
		radius, _ := strconv.ParseFloat(fmt.Sprint(body["radiusKm"]), 64)
		if radius <= 0 {
			radius = 25
		}
		sportID, _ := strconv.ParseInt(strField(body, "sportId"), 10, 64)
		q := strField(body, "q")
		if q == "" {
			q = strField(body, "city")
		}
		rows, err := s.DB.QueryContext(ctx, `
SELECT f.id, f.name, f.location, f.city_name, f.logo_url, f.latitude, f.longitude,
       f.customer_ok, f.maintenance
FROM app_facilities f WHERE f.status='active'`)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id int64
			var name, loc, city, logo string
			var flat, flon float64
			var custOK, maint bool
			_ = rows.Scan(&id, &name, &loc, &city, &logo, &flat, &flon, &custOK, &maint)
			dist := 0.0
			if lat != 0 || lon != 0 {
				dist = haversineKm(lat, lon, flat, flon)
				if dist > radius {
					continue
				}
			}
			if q != "" {
				ql := strings.ToLower(q)
				if !strings.Contains(strings.ToLower(name), ql) &&
					!strings.Contains(strings.ToLower(city), ql) &&
					!strings.Contains(strings.ToLower(loc), ql) {
					continue
				}
			}
			sports := s.facilitySports(ctx, id)
			if sportID > 0 {
				ok := false
				for _, sp := range sports {
					if sp["sportId"] == sportID {
						ok = true
						break
					}
				}
				if !ok {
					continue
				}
			}
			out = append(out, map[string]any{
				"facilityId":               id,
				"facilityName":             name,
				"location":                 loc,
				"cityName":                 city,
				"facilityLogoUrl":          logo,
				"latitude":                 flat,
				"longitude":                flon,
				"distanceKm":               math.Round(dist*10) / 10,
				"sports":                   sports,
				"customer_services_enabled": custOK,
				"maintenance":              maint,
			})
		}
		return true, "OK", map[string]any{"facilities": out}
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) facilitySports(ctx context.Context, fid int64) []map[string]any {
	rows, err := s.DB.QueryContext(ctx, `
SELECT s.id, s.name FROM app_facility_sports fs
JOIN app_sports s ON s.id=fs.sport_id WHERE fs.facility_id=$1`, fid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		_ = rows.Scan(&id, &name)
		list = append(list, map[string]any{"sportId": id, "sportName": name})
	}
	return list
}

func (s *Service) citiesPayload(ctx context.Context) (bool, string, any) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, state_name, latitude, longitude FROM app_cities ORDER BY name`)
	if err != nil {
		return false, err.Error(), nil
	}
	defer rows.Close()
	cities := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, state string
		var lat, lon float64
		_ = rows.Scan(&id, &name, &state, &lat, &lon)
		arows, _ := s.DB.QueryContext(ctx, `SELECT id, name, latitude, longitude FROM app_areas WHERE city_id=$1`, id)
		areas := []map[string]any{}
		if arows != nil {
			for arows.Next() {
				var aid int64
				var aname string
				var alat, alon float64
				_ = arows.Scan(&aid, &aname, &alat, &alon)
				areas = append(areas, map[string]any{
					"areaId": aid, "areaName": aname, "name": aname,
					"latitude": alat, "longitude": alon, "aliases": []string{},
				})
			}
			arows.Close()
		}
		cities = append(cities, map[string]any{
			"cityId": id, "cityName": name, "stateName": state,
			"latitude": lat, "longitude": lon, "areas": areas,
		})
	}
	return true, "OK", map[string]any{"cities": cities, "geoCitiesReady": true}
}

func (s *Service) GetSlots(ctx context.Context, facilityID, sportID int64, date string) (bool, string, any) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT facility_id, sport_id, court_name, slot_date::text, time_range, status, held_by_user_id
FROM app_slots WHERE facility_id=$1 AND sport_id=$2 AND slot_date=$3::date
ORDER BY court_name, time_range`, facilityID, sportID, date)
	if err != nil {
		return false, err.Error(), nil
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var fid, sid int64
		var court, d, tr, st string
		var held sql.NullInt64
		_ = rows.Scan(&fid, &sid, &court, &d, &tr, &st, &held)
		item := map[string]any{
			"facilityId": fid, "sportId": sid, "courtName": court,
			"date": d, "timeRange": tr, "status": st,
		}
		if held.Valid {
			item["held_by_userId"] = held.Int64
			item["heldByUserId"] = held.Int64
		}
		list = append(list, item)
	}
	return true, "OK", list
}

func (s *Service) MarkBooked(ctx context.Context, body map[string]any) (bool, string, any) {
	fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	notes := strings.TrimSpace(fmt.Sprint(body["notes"]))
	price, _ := strconv.ParseFloat(fmt.Sprint(body["totalPrice"]), 64)
	slotsRaw, _ := body["slotsToBook"].([]any)
	slotsJSON, _ := json.Marshal(slotsRaw)
	var bid int64
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO app_bookings (facility_id, user_id, notes, amount, slots_json, payment_mode, formatted_time, raw_timestamp)
VALUES ($1,$2,$3,$4,$5::jsonb,'spot',$6,NOW()) RETURNING id`,
		fid, uid, notes, price, string(slotsJSON), fmt.Sprint(slotsRaw)).Scan(&bid)
	if err != nil {
		return false, err.Error(), nil
	}
	_, _, _ = s.UsersLookup(ctx, map[string]any{"action": "add_facility_access", "userId": uid, "facilityId": fid})
	return true, "Booked", map[string]any{"bookingId": bid}
}

func (s *Service) BookingCheckout(ctx context.Context, body map[string]any) (bool, string, any, int) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
	sportID, _ := strconv.ParseInt(fmt.Sprint(body["sportId"]), 10, 64)
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)

	var blocked bool
	_ = s.DB.QueryRowContext(ctx, `SELECT booking_blocked FROM app_facilities WHERE id=$1`, fid).Scan(&blocked)
	if blocked && (action == "initiate_payment" || action == "confirm_booking" || action == "payment_options") {
		return false, "Bookings temporarily unavailable", map[string]any{"booking_blocked": true}, 402
	}

	switch action {
	case "payment_options":
		rows, err := s.DB.QueryContext(ctx, `
SELECT court_name, day_type, price, advance_amount FROM app_price_list
WHERE facility_id=$1 AND sport_id=$2`, fid, sportID)
		if err != nil {
			return false, err.Error(), nil, 400
		}
		defer rows.Close()
		plist := []map[string]any{}
		adv := 100.0
		for rows.Next() {
			var court, day string
			var price, advance float64
			_ = rows.Scan(&court, &day, &price, &advance)
			if advance > 0 {
				adv = advance
			}
			plist = append(plist, map[string]any{
				"subFacilityName": court, "dayType": day, "price": price, "advance_amount": advance,
			})
		}
		if len(plist) == 0 {
			plist = append(plist, map[string]any{
				"subFacilityName": "Court 1", "dayType": "weekday", "price": 500, "advance_amount": 100,
			})
		}
		return true, "OK", map[string]any{
			"allow_full_payment": true, "allow_advance_payment": true, "allow_spot_payment": true,
			"advance_amount": adv, "Booking_advance_amount": adv,
			"payment_source": "app", "priceList": plist,
		}, 200
	case "validate_coupon":
		code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(body["couponCode"])))
		var value float64
		var ctype string
		err := s.DB.QueryRowContext(ctx, `
SELECT value, ctype FROM app_coupons WHERE facility_id=$1 AND UPPER(code)=$2 AND status='active'`,
			fid, code).Scan(&value, &ctype)
		if err != nil {
			return false, "Invalid coupon", nil, 400
		}
		return true, "OK", map[string]any{"discount": value, "type": ctype}, 200
	case "get_available_coupons":
		rows, err := s.DB.QueryContext(ctx, `
SELECT code, value, ctype, label, description FROM app_coupons
WHERE facility_id=$1 AND status='active'`, fid)
		if err != nil {
			return false, err.Error(), nil, 400
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var code, ctype, label, desc string
			var value float64
			_ = rows.Scan(&code, &value, &ctype, &label, &desc)
			list = append(list, map[string]any{
				"code": code, "value": value, "type": ctype, "label": label, "description": desc,
			})
		}
		return true, "OK", map[string]any{"coupons": list}, 200
	case "initiate_payment":
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["paymentMode"])))
		amount := 500.0
		if mode == "advance" {
			amount = 100
		} else if mode == "spot" {
			amount = 0
		}
		slots, _ := body["slots"].([]any)
		if n := len(slots); n > 1 && mode != "spot" {
			amount = amount * float64(n)
		}
		orderID := fmt.Sprintf("order_local_%d_%d", uid, time.Now().Unix())
		return true, "OK", map[string]any{
			"orderId": orderID, "amount": amount, "keyId": s.Cfg.RazorpayKeyID,
		}, 200
	case "confirm_booking":
		slots, _ := body["slots"].([]any)
		slotsJSON, _ := json.Marshal(slots)
		mode := fmt.Sprint(body["paymentMode"])
		txn := fmt.Sprint(body["razorpayPaymentId"])
		method := fmt.Sprint(body["paymentMethod"])
		var bid int64
		err := s.DB.QueryRowContext(ctx, `
INSERT INTO app_bookings (facility_id, sport_id, user_id, payment_mode, payment_method, transaction_id, slots_json, formatted_time, raw_timestamp)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,NOW()) RETURNING id`,
			fid, sportID, uid, mode, method, txn, string(slotsJSON), fmt.Sprint(slots)).Scan(&bid)
		if err != nil {
			return false, err.Error(), nil, 400
		}
		// mark slots booked
		for _, sl := range slots {
			m, _ := sl.(map[string]any)
			if m == nil {
				continue
			}
			_, _ = s.DB.ExecContext(ctx, `
UPDATE app_slots SET status='booked' WHERE facility_id=$1 AND sport_id=$2
  AND court_name=$3 AND slot_date=$4::date AND time_range=$5`,
				fid, sportID, fmt.Sprint(m["court"]), fmt.Sprint(m["date"]), fmt.Sprint(m["time"]))
		}
		ok, _, access := s.UsersLookup(ctx, map[string]any{"action": "add_facility_access", "userId": uid, "facilityId": fid})
		accessed := []int64{}
		if ok {
			if m, ok2 := access.(map[string]any); ok2 {
				accessed = s.parseIDs(m["accessedFacilityIds"])
			}
		}
		return true, "Confirmed", map[string]any{
			"bookingId": bid, "transactionId": txn, "facilityId": fid,
			"accessedFacilityIds": accessed,
		}, 200
	default:
		return false, "unknown_action", nil, 400
	}
}

func (s *Service) MyBookings(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	switch action {
	case "facilities":
		u, _ := s.userRow(ctx, uid, fmt.Sprint(body["whatsappNo"]))
		ids := []int64{}
		if u != nil {
			ids = s.parseIDs(u["accessedFacilityIds"])
		}
		facilities := []map[string]any{}
		for _, fid := range ids {
			var name string
			var maint bool
			_ = s.DB.QueryRowContext(ctx, `SELECT name, maintenance FROM app_facilities WHERE id=$1`, fid).Scan(&name, &maint)
			var upc, hist int
			_ = s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status='confirmed' AND raw_timestamp > NOW()),
       COUNT(*) FILTER (WHERE status='cancelled' OR raw_timestamp <= NOW())
FROM app_bookings WHERE user_id=$1 AND facility_id=$2`, uid, fid).Scan(&upc, &hist)
			facilities = append(facilities, map[string]any{
				"facilityId": fid, "facilityName": name, "upcomingCount": upc, "historyCount": hist,
				"group": "accessed", "maintenance": maint,
			})
		}
		return true, "OK", map[string]any{
			"accessedFacilityIds": ids,
			"facilities":          facilities,
			"upcomingFacilities":  facilities,
			"historyFacilities":   facilities,
		}
	case "list":
		fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
		rows, err := s.DB.QueryContext(ctx, `
SELECT b.id, b.facility_id, COALESCE(f.name,''), b.sport_id, b.status,
       b.formatted_time, b.formatted_date, b.raw_timestamp, b.court_names, b.slots_json
FROM app_bookings b
LEFT JOIN app_facilities f ON f.id=b.facility_id
WHERE b.user_id=$1 AND ($2=0 OR b.facility_id=$2)
ORDER BY b.raw_timestamp DESC`, uid, fid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		upcoming, history := []map[string]any{}, []map[string]any{}
		now := time.Now()
		for rows.Next() {
			var id, facilityID, sportID int64
			var fname, status, ftime, fdate, courts string
			var ts time.Time
			var slotsJSON []byte
			_ = rows.Scan(&id, &facilityID, &fname, &sportID, &status, &ftime, &fdate, &ts, &courts, &slotsJSON)
			item := map[string]any{
				"bookingId": id, "facilityId": facilityID, "facilityName": fname,
				"sportId": sportID, "sportName": "", "subFacilityNames": courts,
				"formatted_time": ftime, "formatted_date": fdate,
				"ui_day": fmt.Sprintf("%02d", ts.Day()),
				"ui_month": ts.Format("Jan"), "ui_weekday": ts.Weekday().String()[:3],
				"raw_timestamp": ts.Format(time.RFC3339), "status": status,
			}
			if status == "confirmed" && ts.After(now) {
				upcoming = append(upcoming, item)
			} else {
				history = append(history, item)
			}
		}
		coupons := []map[string]any{}
		crows, _ := s.DB.QueryContext(ctx, `
SELECT coupon_code, amount, expiry_date::text, facility_id FROM app_user_coupons WHERE user_id=$1`, uid)
		if crows != nil {
			defer crows.Close()
			for crows.Next() {
				var code, exp string
				var amt float64
				var cfid int64
				_ = crows.Scan(&code, &amt, &exp, &cfid)
				coupons = append(coupons, map[string]any{
					"couponCode": code, "amount": amt, "expiryDate": exp, "facilityId": cfid,
				})
			}
		}
		return true, "OK", map[string]any{
			"upcoming": upcoming, "history": history, "upcomingGroups": upcoming, "coupons": coupons,
		}
	case "cancel":
		bid, _ := strconv.ParseInt(fmt.Sprint(body["bookingId"]), 10, 64)
		var fname string
		err := s.DB.QueryRowContext(ctx, `
UPDATE app_bookings b SET status='cancelled', cancelled_at=NOW()
FROM app_facilities f WHERE b.id=$1 AND b.user_id=$2 AND f.id=b.facility_id
RETURNING f.name`, bid, uid).Scan(&fname)
		if err != nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE app_bookings SET status='cancelled', cancelled_at=NOW() WHERE id=$1 AND user_id=$2`, bid, uid)
			fname = ""
		}
		return true, "Cancelled", map[string]any{"facilityName": fname}
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) FacilityAccess(ctx context.Context, body map[string]any) (bool, string, any, int) {
	fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
	var allowed, blocked, maint bool
	err := s.DB.QueryRowContext(ctx, `
SELECT customer_ok, booking_blocked, maintenance FROM app_facilities WHERE id=$1`, fid).
		Scan(&allowed, &blocked, &maint)
	if err != nil {
		return true, "OK", map[string]any{"allowed": true, "booking_blocked": false}, 200
	}
	ok := allowed && !blocked && !maint
	code := 200
	if !ok {
		code = 402
	}
	return ok, "OK", map[string]any{"allowed": ok, "booking_blocked": blocked || maint}, code
}

func (s *Service) FacilityFeedback(ctx context.Context, body map[string]any) (bool, string, any) {
	fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	rating, _ := strconv.Atoi(fmt.Sprint(body["rating"]))
	msg := strings.TrimSpace(fmt.Sprint(body["message"]))
	wa := normalizeWA(fmt.Sprint(body["whatsappNo"]))
	if rating < 1 || rating > 5 || fid <= 0 {
		return false, "Invalid rating", nil
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO app_feedback (facility_id, user_id, rating, message, whatsapp_no)
VALUES ($1,$2,$3,$4,$5)`, fid, nullInt(uid), rating, msg, wa)
	if err != nil {
		return false, err.Error(), nil
	}
	return true, "Thanks for your feedback", map[string]any{}
}

func nullInt(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func (s *Service) FCMRegister(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	token := strings.TrimSpace(fmt.Sprint(body["token"]))
	if action == "unregister" {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM app_fcm_devices WHERE token=$1`, token)
		_, _ = s.DB.ExecContext(ctx, `UPDATE device_tokens SET status='inactive', updated_at=NOW() WHERE token=$1`, token)
		return true, "OK", nil
	}
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	platform := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["platform"])))
	if platform == "" {
		platform = "android"
	}
	deviceID := fmt.Sprint(body["deviceId"])
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO app_fcm_devices (user_id, token, platform, device_id, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (token) DO UPDATE SET user_id=EXCLUDED.user_id, platform=EXCLUDED.platform,
  device_id=EXCLUDED.device_id, updated_at=NOW()`, uid, token, platform, deviceID)
	if err != nil {
		return false, err.Error(), nil
	}
	// Mirror into notifications.device_tokens so StrollX user_id push resolves.
	if uid > 0 && token != "" {
		_, _ = s.DB.ExecContext(ctx, `
INSERT INTO device_tokens (user_id, token, platform, status)
VALUES ($1, $2, $3, 'active')
ON CONFLICT (user_id, token) DO UPDATE
SET platform=EXCLUDED.platform, status='active', updated_at=NOW()`,
			uid, token, platform)
	}
	return true, "Registered", map[string]any{"ok": true}
}

func (s *Service) AppNotifications(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	if action == "mark_seen" || action == "seen" {
		_, _ = s.DB.ExecContext(ctx, `UPDATE app_notifications SET seen=TRUE WHERE user_id=$1`, uid)
		return true, "OK", nil
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, user_id, facility_id, booking_id, ntype, title, body, facility_name, facility_logo,
       seen, source, meta_json, created_at, expires_at
FROM app_notifications WHERE user_id=$1
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC LIMIT 100`, uid)
	if err != nil {
		return false, err.Error(), nil
	}
	defer rows.Close()
	items := []map[string]any{}
	unread := 0
	var latest int64
	for rows.Next() {
		var id, userID int64
		var fid sql.NullInt64
		var bookingID sql.NullString
		var ntype, title, body, fname, flogo, source string
		var seen bool
		var meta []byte
		var created time.Time
		var expires sql.NullTime
		_ = rows.Scan(&id, &userID, &fid, &bookingID, &ntype, &title, &body, &fname, &flogo, &seen, &source, &meta, &created, &expires)
		if id > latest {
			latest = id
		}
		if !seen {
			unread++
		}
		item := map[string]any{
			"id": id, "userId": userID, "type": ntype, "title": title, "body": body,
			"facilityName": fname, "facilityLogoUrl": flogo, "seen": seen, "source": source,
			"createdAt": created.Format(time.RFC3339), "meta": json.RawMessage(meta),
		}
		if fid.Valid {
			item["facilityId"] = fid.Int64
		}
		if bookingID.Valid {
			item["bookingId"] = bookingID.String
		}
		items = append(items, item)
	}
	return true, "OK", map[string]any{
		"notifications": items, "unreadCount": unread, "latestEventId": latest, "days": 7,
	}
}

func (s *Service) LiveArena(ctx context.Context, action string, q map[string]string) (bool, string, any) {
	switch action {
	case "ping":
		return true, "OK", map[string]any{"file": "live_arena", "php": "go-app"}
	case "list":
		rows, err := s.DB.QueryContext(ctx, `
SELECT id, tournament_name, tournament_date::text, facility_id, sport_name, categories::text, open_match_count
FROM app_live_tournaments ORDER BY tournament_date DESC`)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id, fid int64
			var name, date, sport string
			var open int
			var catStr string
			_ = rows.Scan(&id, &name, &date, &fid, &sport, &catStr, &open)
			cats := parsePgTextArray(catStr)
			var fname string
			_ = s.DB.QueryRowContext(ctx, `SELECT name FROM app_facilities WHERE id=$1`, fid).Scan(&fname)
			list = append(list, map[string]any{
				"tourId": id, "tournamentName": name, "tournamentDate": date,
				"facilityId": fid, "facilityName": fname, "facilityLogoUrl": "",
				"sportName": sport, "categories": cats, "openMatchCount": open,
			})
		}
		return true, "OK", map[string]any{"tournaments": list}
	case "board":
		tid, _ := strconv.ParseInt(q["tourId"], 10, 64)
		var name, sport string
		var fid int64
		_ = s.DB.QueryRowContext(ctx, `
SELECT tournament_name, facility_id, sport_name FROM app_live_tournaments WHERE id=$1`, tid).
			Scan(&name, &fid, &sport)
		var fname string
		_ = s.DB.QueryRowContext(ctx, `SELECT name FROM app_facilities WHERE id=$1`, fid).Scan(&fname)
		rows, err := s.DB.QueryContext(ctx, `
SELECT id, category, gender, player1, player2, status, winner, court, score1, score2
FROM app_live_matches WHERE tour_id=$1 ORDER BY id`, tid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		matches := []map[string]any{}
		var afterID int64
		for rows.Next() {
			var id int64
			var cat, gender, p1, p2, status, winner, court string
			var s1, s2 int
			_ = rows.Scan(&id, &cat, &gender, &p1, &p2, &status, &winner, &court, &s1, &s2)
			if id > afterID {
				afterID = id
			}
			matches = append(matches, map[string]any{
				"id": id, "matchId": id, "category": cat, "gender": gender,
				"player1": p1, "player2": p2, "status": status, "winner": winner, "court": court,
				"live1": s1, "live2": s2, "score_details": fmt.Sprintf("%d-%d", s1, s2),
			})
		}
		return true, "OK", map[string]any{
			"tourId": tid, "tournamentName": name, "facilityName": fname, "sportName": sport,
			"category": q["category"], "categories": []string{}, "afterId": afterID, "matches": matches,
		}
	default:
		return false, "unknown_action", nil
	}
}

func parsePgTextArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return []string{}
	}
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) ScoreLive(ctx context.Context, action string, q map[string]string) (bool, string, any) {
	switch action {
	case "ping_public", "ping":
		return true, "OK", map[string]any{"ok": true}
	case "since_public", "since":
		after, _ := strconv.ParseInt(q["afterId"], 10, 64)
		mid, _ := strconv.ParseInt(q["matchId"], 10, 64)
		var id int64
		var s1, s2 int
		var ver int64
		err := s.DB.QueryRowContext(ctx, `
SELECT id, score1, score2, version FROM app_live_matches
WHERE ($1=0 OR id=$1) AND id > $2 ORDER BY id DESC LIMIT 1`, mid, after).Scan(&id, &s1, &s2, &ver)
		if err != nil {
			return true, "OK", map[string]any{"changed": false, "version": 0}
		}
		return true, "OK", map[string]any{
			"changed": true, "version": ver, "matchId": id, "score1": s1, "score2": s2,
			"events": []map[string]any{{"matchId": id, "score1": s1, "score2": s2}}, "afterId": id,
		}
	case "wait_public", "wait":
		return s.ScoreLive(ctx, "since_public", q)
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) ScoreFollows(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	switch action {
	case "prefs_get":
		var my, fr bool
		err := s.DB.QueryRowContext(ctx, `SELECT my_matches, friend_matches FROM app_score_prefs WHERE user_id=$1`, uid).Scan(&my, &fr)
		if err != nil {
			my, fr = true, true
		}
		return true, "OK", map[string]any{"myMatches": my, "friendMatches": fr}
	case "prefs_set":
		my := body["myMatches"] == true || fmt.Sprint(body["myMatches"]) == "true"
		fr := body["friendMatches"] == true || fmt.Sprint(body["friendMatches"]) == "true"
		_, _ = s.DB.ExecContext(ctx, `
INSERT INTO app_score_prefs (user_id, my_matches, friend_matches) VALUES ($1,$2,$3)
ON CONFLICT (user_id) DO UPDATE SET my_matches=$2, friend_matches=$3`, uid, my, fr)
		return true, "OK", map[string]any{"myMatches": my, "friendMatches": fr}
	case "follow":
		tid, _ := strconv.ParseInt(fmt.Sprint(body["tourId"]), 10, 64)
		mid, _ := strconv.ParseInt(fmt.Sprint(body["matchId"]), 10, 64)
		tt := fmt.Sprint(body["target_type"])
		tk := fmt.Sprint(body["target_key"])
		_, _ = s.DB.ExecContext(ctx, `
INSERT INTO app_score_follows (user_id, tour_id, match_id, target_type, target_key)
VALUES ($1,$2,$3,$4,$5)`, uid, tid, mid, tt, tk)
		return true, "OK", nil
	case "unfollow":
		mid, _ := strconv.ParseInt(fmt.Sprint(body["matchId"]), 10, 64)
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM app_score_follows WHERE user_id=$1 AND match_id=$2`, uid, mid)
		return true, "OK", nil
	case "status_for_match":
		mid, _ := strconv.ParseInt(fmt.Sprint(body["matchId"]), 10, 64)
		var n int
		_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_score_follows WHERE user_id=$1 AND match_id=$2`, uid, mid).Scan(&n)
		prefs, _, pdata := s.ScoreFollows(ctx, map[string]any{"action": "prefs_get", "userId": uid})
		_ = prefs
		return true, "OK", map[string]any{
			"signedIn": uid > 0, "watchMatch": n > 0,
			"followPlayer1": false, "followPlayer2": false,
			"side1Key": "", "side2Key": "", "prefs": pdata,
		}
	case "list":
		rows, err := s.DB.QueryContext(ctx, `
SELECT target_type, target_key FROM app_score_follows WHERE user_id=$1`, uid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var tt, tk string
			_ = rows.Scan(&tt, &tk)
			list = append(list, map[string]any{"target_type": tt, "target_key": tk})
		}
		return true, "OK", map[string]any{"follows": list}
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) StudentPortal(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	switch action {
	case "my_enrollments":
		rows, err := s.DB.QueryContext(ctx, `
SELECT e.id, e.student_id, e.student_name, e.level, e.status, e.plan_id, e.plan_name,
       e.facility_id, COALESCE(f.name,''), COALESCE(f.logo_url,''), e.sport_id, e.sport_name
FROM app_student_enrollments e
LEFT JOIN app_facilities f ON f.id=e.facility_id
WHERE e.user_id=$1`, uid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		month := time.Now().Format("2006-01")
		for rows.Next() {
			var id, sid, planID, fid, sportID int64
			var sname, level, status, planName, fname, logo, sportName string
			_ = rows.Scan(&id, &sid, &sname, &level, &status, &planID, &planName, &fid, &fname, &logo, &sportID, &sportName)
			var present, absent int
			_ = s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status='present'), COUNT(*) FILTER (WHERE status='absent')
FROM app_student_attendance WHERE enrollment_id=$1 AND to_char(attend_date,'YYYY-MM')=$2`, id, month).
				Scan(&present, &absent)
			list = append(list, map[string]any{
				"enrollmentId": id, "studentId": sid, "studentName": sname, "level": level, "status": status,
				"planId": planID, "planName": planName, "facilityId": fid, "facilityName": fname,
				"facilityLogoUrl": logo, "sportId": sportID, "sportName": sportName,
				"currentMonth": month, "presentCount": present, "absentCount": absent,
			})
		}
		return true, "OK", map[string]any{"enrollments": list}
	case "reports":
		eid, _ := strconv.ParseInt(fmt.Sprint(body["enrollmentId"]), 10, 64)
		return true, "OK", map[string]any{
			"enrollment": map[string]any{"enrollmentId": eid},
			"reports":    []map[string]any{},
		}
	case "month_calendar":
		eid, _ := strconv.ParseInt(fmt.Sprint(body["enrollmentId"]), 10, 64)
		ym := strings.TrimSpace(fmt.Sprint(body["yearMonth"]))
		if ym == "" {
			ym = time.Now().Format("2006-01")
		}
		return true, "OK", map[string]any{
			"enrollment": map[string]any{"enrollmentId": eid},
			"yearMonth": ym, "label": ym, "isCurrentMonth": ym == time.Now().Format("2006-01"),
			"presentCount": 0, "absentCount": 0, "days": []any{},
			"availableMonths": []map[string]any{{"yearMonth": ym, "label": ym, "isCurrent": true}},
		}
	case "leave_request":
		eid, _ := strconv.ParseInt(fmt.Sprint(body["enrollmentId"]), 10, 64)
		date := fmt.Sprint(body["leaveDate"])
		reason := fmt.Sprint(body["reason"])
		_, err := s.DB.ExecContext(ctx, `
INSERT INTO app_student_attendance (enrollment_id, attend_date, status, leave_reason)
VALUES ($1,$2::date,'leave',$3)
ON CONFLICT (enrollment_id, attend_date) DO UPDATE SET status='leave', leave_reason=$3`, eid, date, reason)
		if err != nil {
			return false, err.Error(), nil
		}
		return true, "Leave requested", nil
	case "cancel_leave":
		eid, _ := strconv.ParseInt(fmt.Sprint(body["enrollmentId"]), 10, 64)
		date := fmt.Sprint(body["leaveDate"])
		_, _ = s.DB.ExecContext(ctx, `
DELETE FROM app_student_attendance WHERE enrollment_id=$1 AND attend_date=$2::date AND status='leave'`, eid, date)
		return true, "Cancelled", nil
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) PrimaryMembers(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	switch action {
	case "my_plans":
		rows, err := s.DB.QueryContext(ctx, `
SELECT p.id, p.membership_id, p.member_id, p.member_name, p.plan_id, p.facility_id,
       COALESCE(f.name,''), COALESCE(f.logo_url,''), p.sport_id, p.sport_name, p.plan_name,
       p.sub_facility_name, p.slot_time, p.indiv_price, p.grp_price, p.subscription_type,
       p.max_member_count, p.current_member_count, p.team_name, p.subscription_fee, p.is_primary
FROM app_member_plans p
LEFT JOIN app_facilities f ON f.id=p.facility_id
WHERE p.user_id=$1`, uid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id, mid, memberID, planID, fid, sportID int64
			var mname, fname, logo, sportName, planName, sub, slot, subType, team string
			var indiv, grp, fee float64
			var maxC, curC int
			var primary bool
			_ = rows.Scan(&id, &mid, &memberID, &mname, &planID, &fid, &fname, &logo, &sportID, &sportName, &planName,
				&sub, &slot, &indiv, &grp, &subType, &maxC, &curC, &team, &fee, &primary)
			if mid == 0 {
				mid = id
			}
			list = append(list, map[string]any{
				"membershipId": mid, "memberId": memberID, "memberName": mname, "planId": planID,
				"facilityId": fid, "facilityName": fname, "facilityLogoUrl": logo,
				"sportId": sportID, "sportName": sportName, "planName": planName,
				"subFacilityName": sub, "slotTime": slot, "indiv_price": indiv, "grp_price": grp,
				"subscription_type": subType, "max_member_count": maxC, "current_member_count": curC,
				"teamName": team, "subscriptionFee": fee, "isPrimary": primary, "primaryMember": primary,
			})
		}
		return true, "OK", map[string]any{"plans": list}
	case "plan_members":
		mid, _ := strconv.ParseInt(fmt.Sprint(body["membershipId"]), 10, 64)
		rows, err := s.DB.QueryContext(ctx, `
SELECT membership_id, member_id, member_name, whatsapp_number, is_primary, user_id= $2 AS is_mine,
       status_label, coming_today FROM app_member_plans
WHERE membership_id=$1 OR id=$1 OR plan_id=(SELECT plan_id FROM app_member_plans WHERE id=$1 OR membership_id=$1 LIMIT 1)`, mid, uid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		members := []map[string]any{}
		myIDs := []int64{}
		coming := 0
		viewerPrimary := false
		for rows.Next() {
			var membershipID, memberID int64
			var mname, wa, status string
			var primary, isMine, comingToday bool
			_ = rows.Scan(&membershipID, &memberID, &mname, &wa, &primary, &isMine, &status, &comingToday)
			if comingToday {
				coming++
			}
			if isMine {
				myIDs = append(myIDs, membershipID)
				if primary {
					viewerPrimary = true
				}
			}
			members = append(members, map[string]any{
				"membershipId": membershipID, "memberId": memberID, "memberName": mname,
				"whatsappNumber": wa, "isPrimary": primary, "isMine": isMine,
				"statusLabel": status, "canSelect": true, "request_status": "",
				"comingToday": comingToday, "canToggleComing": isMine || viewerPrimary,
			})
		}
		return true, "OK", map[string]any{
			"plan": map[string]any{"membershipId": mid}, "members": members,
			"myMembershipIds": myIDs, "isViewerPrimary": viewerPrimary,
			"playDate": time.Now().Format("2006-01-02"), "comingTodayCount": coming,
		}
	case "toggle_coming_today":
		mid, _ := strconv.ParseInt(fmt.Sprint(body["membershipId"]), 10, 64)
		_, _ = s.DB.ExecContext(ctx, `
UPDATE app_member_plans SET coming_today = NOT coming_today WHERE membership_id=$1 OR id=$1`, mid)
		return true, "OK", nil
	case "add_member":
		name := strings.TrimSpace(fmt.Sprint(body["memberName"]))
		wa := normalizeWA(fmt.Sprint(body["whatsappNumber"]))
		fid, _ := strconv.ParseInt(fmt.Sprint(body["facilityId"]), 10, 64)
		_, err := s.DB.ExecContext(ctx, `
INSERT INTO app_member_plans (user_id, membership_id, member_id, member_name, facility_id, whatsapp_number, is_primary)
VALUES ($1, nextval('app_member_plans_id_seq'), nextval('app_member_plans_id_seq'), $2, $3, $4, FALSE)`,
			uid, name, fid, wa)
		if err != nil {
			return false, err.Error(), nil
		}
		return true, "Added", nil
	case "submit_request":
		return true, "Submitted", nil
	default:
		return false, "unknown_action", nil
	}
}

func (s *Service) BillingPortal(ctx context.Context, body map[string]any) (bool, string, any) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	uid, _ := strconv.ParseInt(fmt.Sprint(body["userId"]), 10, 64)
	switch action {
	case "my_bills":
		rows, err := s.DB.QueryContext(ctx, `
SELECT b.id, b.enrollment_id, b.bill_type, b.bill_amount, b.paid_amount, b.billing_month,
       b.due_date::text, b.payment_status, b.plan_name, b.display_name, b.facility_id,
       COALESCE(f.name,''), COALESCE(f.logo_url,''), b.transaction_id
FROM app_fee_bills b
LEFT JOIN app_facilities f ON f.id=b.facility_id
WHERE b.user_id=$1 ORDER BY b.created_at DESC`, uid)
		if err != nil {
			return false, err.Error(), nil
		}
		defer rows.Close()
		bills := []map[string]any{}
		pendingCount, paidCount := 0, 0
		pendingTotal := 0.0
		facMap := map[int64]map[string]any{}
		for rows.Next() {
			var id, eid, fid int64
			var btype, month, due, status, plan, display, fname, logo, txn string
			var amount, paid float64
			_ = rows.Scan(&id, &eid, &btype, &amount, &paid, &month, &due, &status, &plan, &display, &fid, &fname, &logo, &txn)
			bal := amount - paid
			isPaid := status == "paid" || bal <= 0
			if isPaid {
				paidCount++
			} else {
				pendingCount++
				pendingTotal += bal
			}
			bills = append(bills, map[string]any{
				"billId": id, "enrollmentId": eid, "type": btype, "audience": btype,
				"billAmount": amount, "paidAmount": paid, "balanceDue": bal,
				"billingMonth": month, "billingMonthLabel": month, "dueDate": due,
				"paymentStatus": status, "isPaid": isPaid, "canPayOnline": !isPaid,
				"paymentMethod": "", "transactionId": txn, "facilityId": fid,
				"facilityName": fname, "facilityLogoUrl": logo, "planName": plan, "displayName": display,
			})
			if _, ok := facMap[fid]; !ok {
				facMap[fid] = map[string]any{
					"facilityId": fid, "facilityName": fname, "facilityLogoUrl": logo,
					"pendingCount": 0, "pendingTotal": 0.0, "payUrl": "",
				}
			}
			if !isPaid {
				facMap[fid]["pendingCount"] = facMap[fid]["pendingCount"].(int) + 1
				facMap[fid]["pendingTotal"] = facMap[fid]["pendingTotal"].(float64) + bal
			}
		}
		facilities := []map[string]any{}
		for _, v := range facMap {
			facilities = append(facilities, v)
		}
		return true, "OK", map[string]any{
			"bills": bills, "facilities": facilities,
			"pendingCount": pendingCount, "paidCount": paidCount, "pendingTotal": pendingTotal,
		}
	case "pay_url":
		return true, "OK", map[string]any{"payUrl": ""}
	case "initiate_payment":
		orderID := fmt.Sprintf("fee_order_%d_%d", uid, time.Now().Unix())
		amount, _ := strconv.ParseFloat(fmt.Sprint(body["amount"]), 64)
		return true, "OK", map[string]any{
			"orderId": orderID, "keyId": s.Cfg.RazorpayKeyID, "amount": amount,
			"currency": "INR", "category": "FEES", "billIds": body["billIds"],
		}
	case "confirm_payment":
		return true, "Paid", map[string]any{"ok": true}
	default:
		return false, "unknown_action", nil
	}
}
