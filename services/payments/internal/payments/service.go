package payments

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/facinect/payments/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

type LedgerRow struct {
	ID             int64   `json:"id"`
	FacilityID     int64   `json:"facility_id"`
	Category       string  `json:"category"`
	Title          string  `json:"title"`
	CustomerName   string  `json:"customer_name,omitempty"`
	CustomerPhone  string  `json:"customer_phone,omitempty"`
	Amount         float64 `json:"amount"`
	PaidAmount     float64 `json:"paid_amount"`
	Status         string  `json:"status"`
	PaymentMethod  string  `json:"payment_method,omitempty"`
	ReferenceType  string  `json:"reference_type,omitempty"`
	ReferenceID    string  `json:"reference_id,omitempty"`
	Notes          string  `json:"notes,omitempty"`
	PaidAt         string  `json:"paid_at,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
}

type CreateInput struct {
	Category      string  `json:"category"`
	Title         string  `json:"title"`
	CustomerName  string  `json:"customer_name"`
	CustomerPhone string  `json:"customer_phone"`
	Amount        float64 `json:"amount"`
	PaidAmount    float64 `json:"paid_amount"`
	Status        string  `json:"status"`
	PaymentMethod string  `json:"payment_method"`
	ReferenceType string  `json:"reference_type"`
	ReferenceID   string  `json:"reference_id"`
	Notes         string  `json:"notes"`
}

type MarkPaidInput struct {
	PaidAmount    *float64 `json:"paid_amount"`
	PaymentMethod string   `json:"payment_method"`
	Notes         string   `json:"notes"`
}

type ListFilter struct {
	Category string
	Status   string
	FromDate string
	ToDate   string
}

type Summary struct {
	TotalAmount   float64 `json:"total_amount"`
	TotalPaid     float64 `json:"total_paid"`
	PendingCount  int     `json:"pending_count"`
	PaidCount     int     `json:"paid_count"`
	LedgerCount   int     `json:"ledger_count"`
	PendingAmount float64 `json:"pending_amount"`
}

var allowedCategories = map[string]bool{
	"BOOKING": true, "MEMBERSHIP": true, "COACHING": true, "TOURNAMENT": true, "OTHER": true,
}

var allowedStatus = map[string]bool{
	"pending": true, "paid": true, "partial": true, "failed": true, "refunded": true,
}

func (s *Service) List(ctx context.Context, facilityID int64, f ListFilter) ([]LedgerRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	q := `
SELECT id, facility_id, category, title, COALESCE(customer_name,''), COALESCE(customer_phone,''),
       amount, paid_amount, status, COALESCE(payment_method,''), COALESCE(reference_type,''),
       COALESCE(reference_id,''), COALESCE(notes,''), COALESCE(paid_at::text,''), created_at::text
FROM payment_ledger WHERE facility_id=$1`
	args := []interface{}{facilityID}
	argN := 2

	cat := strings.ToUpper(strings.TrimSpace(f.Category))
	if cat != "" && cat != "ALL" {
		if !allowedCategories[cat] {
			return nil, fmt.Errorf("invalid_category")
		}
		q += fmt.Sprintf(` AND category=$%d`, argN)
		args = append(args, cat)
		argN++
	}
	st := strings.ToLower(strings.TrimSpace(f.Status))
	if st != "" && st != "all" {
		if !allowedStatus[st] {
			return nil, fmt.Errorf("invalid_status")
		}
		q += fmt.Sprintf(` AND status=$%d`, argN)
		args = append(args, st)
		argN++
	}
	if from := strings.TrimSpace(f.FromDate); from != "" {
		if err := validateDate(from); err != nil {
			return nil, err
		}
		q += fmt.Sprintf(` AND created_at::date >= $%d::date`, argN)
		args = append(args, from)
		argN++
	}
	if to := strings.TrimSpace(f.ToDate); to != "" {
		if err := validateDate(to); err != nil {
			return nil, err
		}
		q += fmt.Sprintf(` AND created_at::date <= $%d::date`, argN)
		args = append(args, to)
	}
	q += ` ORDER BY created_at DESC, id DESC`

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LedgerRow{}
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) Summary(ctx context.Context, facilityID int64, f ListFilter) (*Summary, error) {
	list, err := s.List(ctx, facilityID, f)
	if err != nil {
		return nil, err
	}
	sum := &Summary{LedgerCount: len(list)}
	for _, r := range list {
		sum.TotalAmount += r.Amount
		sum.TotalPaid += r.PaidAmount
		switch r.Status {
		case "pending", "partial":
			sum.PendingCount++
			due := r.Amount - r.PaidAmount
			if due > 0 {
				sum.PendingAmount += due
			}
		case "paid":
			sum.PaidCount++
		}
	}
	return sum, nil
}

func (s *Service) Create(ctx context.Context, facilityID int64, in CreateInput) (*LedgerRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	cat := strings.ToUpper(strings.TrimSpace(in.Category))
	if cat == "" {
		cat = "BOOKING"
	}
	if !allowedCategories[cat] {
		return nil, fmt.Errorf("invalid_category")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = cat + " payment"
	}
	if in.Amount < 0 || in.PaidAmount < 0 {
		return nil, fmt.Errorf("invalid_amount")
	}
	st := strings.ToLower(strings.TrimSpace(in.Status))
	if st == "" {
		if in.PaidAmount <= 0 {
			st = "pending"
		} else if in.PaidAmount >= in.Amount {
			st = "paid"
		} else {
			st = "partial"
		}
	}
	if !allowedStatus[st] {
		return nil, fmt.Errorf("invalid_status")
	}
	var paidAt interface{}
	if st == "paid" || st == "partial" {
		paidAt = time.Now().UTC()
	}

	var id int64
	err := s.DB.QueryRowContext(ctx, `
INSERT INTO payment_ledger
  (facility_id, category, title, customer_name, customer_phone, amount, paid_amount,
   status, payment_method, reference_type, reference_id, notes, paid_at)
VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),NULLIF($12,''),$13)
RETURNING id`,
		facilityID, cat, title,
		strings.TrimSpace(in.CustomerName), strings.TrimSpace(in.CustomerPhone),
		in.Amount, in.PaidAmount, st,
		strings.TrimSpace(in.PaymentMethod), strings.TrimSpace(in.ReferenceType),
		strings.TrimSpace(in.ReferenceID), strings.TrimSpace(in.Notes), paidAt,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*LedgerRow, error) {
	row := s.DB.QueryRowContext(ctx, `
SELECT id, facility_id, category, title, COALESCE(customer_name,''), COALESCE(customer_phone,''),
       amount, paid_amount, status, COALESCE(payment_method,''), COALESCE(reference_type,''),
       COALESCE(reference_id,''), COALESCE(notes,''), COALESCE(paid_at::text,''), created_at::text
FROM payment_ledger WHERE id=$1`, id)
	r, err := scanRow(row)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) MarkPaid(ctx context.Context, facilityID, paymentID int64, in MarkPaidInput) (*LedgerRow, error) {
	cur, err := s.Get(ctx, paymentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not_found")
		}
		return nil, err
	}
	if cur.FacilityID != facilityID {
		return nil, fmt.Errorf("not_found")
	}
	paid := cur.Amount
	if in.PaidAmount != nil {
		if *in.PaidAmount < 0 {
			return nil, fmt.Errorf("invalid_amount")
		}
		paid = *in.PaidAmount
	}
	st := "paid"
	if paid < cur.Amount {
		st = "partial"
	}
	method := strings.TrimSpace(in.PaymentMethod)
	if method == "" {
		method = "cash"
	}
	notes := strings.TrimSpace(in.Notes)
	if notes == "" {
		notes = cur.Notes
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE payment_ledger
SET paid_amount=$1, status=$2, payment_method=$3, notes=NULLIF($4,''),
    paid_at=NOW(), updated_at=NOW()
WHERE id=$5 AND facility_id=$6`, paid, st, method, notes, paymentID, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("not_found")
	}
	return s.Get(ctx, paymentID)
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRow(row scannable) (LedgerRow, error) {
	var r LedgerRow
	err := row.Scan(
		&r.ID, &r.FacilityID, &r.Category, &r.Title, &r.CustomerName, &r.CustomerPhone,
		&r.Amount, &r.PaidAmount, &r.Status, &r.PaymentMethod, &r.ReferenceType,
		&r.ReferenceID, &r.Notes, &r.PaidAt, &r.CreatedAt,
	)
	return r, err
}

func validateDate(date string) error {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(date)); err != nil {
		return fmt.Errorf("invalid_date")
	}
	return nil
}
