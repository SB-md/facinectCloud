package booking

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

type CollectPaymentInput struct {
	PayMode string   `json:"pay_mode"`
	Amount  *float64 `json:"amount"`
}

type RecurrenceBlockInput struct {
	CourtID *int64   `json:"court_id"`
	Days    []string `json:"days"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
	Reason  string   `json:"reason"`
}

type CreateCouponsInput struct {
	Codes    []string `json:"codes"`
	Count    int      `json:"count"`
	Discount float64  `json:"discount"`
}

type Coupon struct {
	ID         int64   `json:"id"`
	FacilityID int64   `json:"facility_id"`
	Code       string  `json:"code"`
	Discount   float64 `json:"discount"`
	Status     string  `json:"status"`
}

type UserStatusInput struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (s *Service) CleanupSlots(ctx context.Context, facilityID int64) (int, error) {
	if facilityID <= 0 {
		return 0, fmt.Errorf("invalid_facility")
	}
	res, err := s.DB.ExecContext(ctx, `
DELETE FROM booking_slots
WHERE facility_id=$1 AND slot_date < CURRENT_DATE AND status='available'`, facilityID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *Service) CreateRecurrenceBlocks(ctx context.Context, facilityID int64, in RecurrenceBlockInput) (int, error) {
	if facilityID <= 0 {
		return 0, fmt.Errorf("invalid_facility")
	}
	start := normalizeTime(in.Start)
	end := normalizeTime(in.End)
	if start == "" || end == "" {
		return 0, fmt.Errorf("start_end_required")
	}
	if len(in.Days) == 0 {
		return 0, fmt.Errorf("days_required")
	}
	reason := strings.TrimSpace(in.Reason)
	blocked := 0
	for _, day := range in.Days {
		day = strings.TrimSpace(day)
		if err := validateDate(day); err != nil {
			return blocked, err
		}
		var court interface{}
		if in.CourtID != nil && *in.CourtID > 0 {
			court = *in.CourtID
		}
		_, err := s.DB.ExecContext(ctx, `
INSERT INTO booking_blocks (facility_id, court_id, block_date, start_time, end_time, reason)
VALUES ($1,$2,$3::date,$4::time,$5::time,NULLIF($6,''))`,
			facilityID, court, day, start, end, reason)
		if err != nil {
			return blocked, err
		}

		q := `
UPDATE booking_slots SET status='blocked', notes=COALESCE(NULLIF($1,''), notes), updated_at=NOW()
WHERE facility_id=$2 AND slot_date=$3::date AND status='available'
  AND start_time >= $4::time AND end_time <= $5::time`
		args := []interface{}{reason, facilityID, day, start, end}
		if in.CourtID != nil && *in.CourtID > 0 {
			q += ` AND court_id=$6`
			args = append(args, *in.CourtID)
		}
		res, err := s.DB.ExecContext(ctx, q, args...)
		if err != nil {
			return blocked, err
		}
		n, _ := res.RowsAffected()
		blocked += int(n)
	}
	return blocked, nil
}

func (s *Service) CollectPayment(ctx context.Context, bookingID int64, in CollectPaymentInput) (*Booking, error) {
	mode := strings.TrimSpace(in.PayMode)
	if mode == "" {
		return nil, fmt.Errorf("pay_mode_required")
	}
	var amount interface{}
	if in.Amount != nil {
		amount = *in.Amount
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE booking_bookings
SET payment_status='paid', pay_mode=$2, amount_paid=$3, updated_at=NOW()
WHERE id=$1 AND status='confirmed'`, bookingID, mode, amount)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found")
	}
	return s.GetBooking(ctx, bookingID)
}

func (s *Service) CreateCoupons(ctx context.Context, facilityID int64, in CreateCouponsInput) ([]Coupon, error) {
	codes := append([]string{}, in.Codes...)
	if len(codes) == 0 {
		count := in.Count
		if count <= 0 {
			count = 1
		}
		if count > 100 {
			count = 100
		}
		for i := 0; i < count; i++ {
			c, err := stubBookingCoupon()
			if err != nil {
				return nil, err
			}
			codes = append(codes, c)
		}
	}
	out := make([]Coupon, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(strings.ToUpper(code))
		if code == "" {
			continue
		}
		var id int64
		err := s.DB.QueryRowContext(ctx, `
INSERT INTO booking_coupons (facility_id, code, discount, status)
VALUES ($1,$2,$3,'active')
ON CONFLICT (facility_id, code) DO UPDATE SET discount=EXCLUDED.discount, status='active'
RETURNING id`, facilityID, code, in.Discount).Scan(&id)
		if err != nil {
			return out, err
		}
		out = append(out, Coupon{
			ID: id, FacilityID: facilityID, Code: code, Discount: in.Discount, Status: "active",
		})
	}
	return out, nil
}

func stubBookingCoupon() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return "SAVE" + string(b), nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, facilityID, userID int64, in UserStatusInput) error {
	st := strings.ToLower(strings.TrimSpace(in.Status))
	if st != "active" && st != "blocked" {
		return fmt.Errorf("invalid_status")
	}
	if facilityID <= 0 || userID <= 0 {
		return fmt.Errorf("invalid_user")
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO booking_user_status (facility_id, user_id, status, reason, updated_at)
VALUES ($1,$2,$3,NULLIF($4,''),NOW())
ON CONFLICT (facility_id, user_id) DO UPDATE
SET status=EXCLUDED.status, reason=EXCLUDED.reason, updated_at=NOW()`,
		facilityID, userID, st, strings.TrimSpace(in.Reason))
	return err
}

func normalizeTime(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	if len(t) == 5 {
		return t + ":00"
	}
	if len(t) >= 8 {
		return t[:8]
	}
	return t
}
