package students

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/facinect/students/internal/config"
)

type Service struct {
	DB  *sql.DB
	Cfg config.Config
}

type StudentRow struct {
	EnrollmentID int64    `json:"enrollment_id"`
	StudentID    int64    `json:"student_id"`
	FacilityID   int64    `json:"facility_id"`
	FirstName    string   `json:"first_name"`
	LastName     string   `json:"last_name"`
	Email        string   `json:"contact_email,omitempty"`
	Phone        string   `json:"contact_phone,omitempty"`
	WhatsApp     string   `json:"whatsapp,omitempty"`
	SportID      *int64   `json:"sport_id,omitempty"`
	PlanName     string   `json:"plan_name,omitempty"`
	StartDate    string   `json:"start_date,omitempty"`
	EndDate      string   `json:"end_date,omitempty"`
	ActualFee    *float64 `json:"actual_fee,omitempty"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at,omitempty"`
}

type AttendanceRow struct {
	EnrollmentID   int64  `json:"enrollment_id"`
	StudentID      int64  `json:"student_id"`
	FacilityID     int64  `json:"facility_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Phone          string `json:"contact_phone,omitempty"`
	PlanName       string `json:"plan_name,omitempty"`
	AttendanceDate string `json:"attendance_date"`
	Status         string `json:"status,omitempty"`
	AttendanceID   *int64 `json:"attendance_id,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type EnrollInput struct {
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     string   `json:"contact_email"`
	Phone     string   `json:"contact_phone"`
	WhatsApp  string   `json:"whatsapp"`
	SportID   *int64   `json:"sport_id"`
	PlanName  string   `json:"plan_name"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	ActualFee *float64 `json:"actual_fee"`
}

type MarkAttendanceInput struct {
	EnrollmentID int64  `json:"enrollment_id"`
	Date         string `json:"date"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
	MarkedBy     *int64 `json:"marked_by"`
}

type UpdateStatusInput struct {
	Status string `json:"status"`
}

type UpdateStudentInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"contact_email"`
	Phone     string `json:"contact_phone"`
	WhatsApp  string `json:"whatsapp"`
}

type UpdatePlanInput struct {
	PlanName  string   `json:"plan_name"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	ActualFee *float64 `json:"actual_fee"`
}

type PlanRow struct {
	PlanID   int64  `json:"plan_id"`
	PlanName string `json:"plan_name"`
	Count    int    `json:"student_count"`
}

func (s *Service) ListStudents(ctx context.Context, facilityID int64, sportID *int64, status string) ([]StudentRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	q := `
SELECT e.id, s.id, e.facility_id, s.first_name, s.last_name,
       COALESCE(s.contact_email,''), COALESCE(s.contact_phone,''), COALESCE(s.whatsapp,''),
       e.sport_id, COALESCE(e.plan_name,''),
       COALESCE(e.start_date::text,''), COALESCE(e.end_date::text,''), e.actual_fee,
       e.status, e.created_at::text
FROM student_enrollments e
JOIN students s ON s.id = e.student_id
WHERE e.facility_id=$1`
	args := []interface{}{facilityID}
	argN := 2
	if sportID != nil && *sportID > 0 {
		q += fmt.Sprintf(` AND e.sport_id=$%d`, argN)
		args = append(args, *sportID)
		argN++
	}
	st := strings.TrimSpace(strings.ToLower(status))
	if st != "" && st != "all" {
		q += fmt.Sprintf(` AND e.status=$%d`, argN)
		args = append(args, st)
	}
	q += ` ORDER BY s.first_name, s.last_name, e.id DESC`

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StudentRow{}
	for rows.Next() {
		var r StudentRow
		var sport sql.NullInt64
		var fee sql.NullFloat64
		if err := rows.Scan(
			&r.EnrollmentID, &r.StudentID, &r.FacilityID, &r.FirstName, &r.LastName,
			&r.Email, &r.Phone, &r.WhatsApp, &sport, &r.PlanName,
			&r.StartDate, &r.EndDate, &fee, &r.Status, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		if sport.Valid {
			v := sport.Int64
			r.SportID = &v
		}
		if fee.Valid {
			v := fee.Float64
			r.ActualFee = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) CountActive(ctx context.Context, facilityID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM student_enrollments
WHERE facility_id=$1 AND status='active'`, facilityID).Scan(&n)
	return n, err
}

func (s *Service) Enroll(ctx context.Context, facilityID int64, in EnrollInput) (*StudentRow, error) {
	first := strings.TrimSpace(in.FirstName)
	last := strings.TrimSpace(in.LastName)
	if facilityID <= 0 || first == "" {
		return nil, fmt.Errorf("invalid_student")
	}
	if in.StartDate != "" {
		if err := validateDate(in.StartDate); err != nil {
			return nil, err
		}
	}
	if in.EndDate != "" {
		if err := validateDate(in.EndDate); err != nil {
			return nil, err
		}
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var studentID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO students (first_name, last_name, contact_email, contact_phone, whatsapp, status)
VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),'active')
RETURNING id`,
		first, last,
		strings.TrimSpace(in.Email),
		strings.TrimSpace(in.Phone),
		strings.TrimSpace(in.WhatsApp),
	).Scan(&studentID)
	if err != nil {
		return nil, err
	}

	var sport interface{}
	if in.SportID != nil {
		sport = *in.SportID
	}
	var start, end, fee interface{}
	if strings.TrimSpace(in.StartDate) != "" {
		start = in.StartDate
	}
	if strings.TrimSpace(in.EndDate) != "" {
		end = in.EndDate
	}
	if in.ActualFee != nil {
		fee = *in.ActualFee
	}

	var enrollmentID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO student_enrollments
  (facility_id, student_id, sport_id, plan_name, start_date, end_date, actual_fee, status)
VALUES ($1,$2,$3,NULLIF($4,''),$5::date,$6::date,$7,'active')
RETURNING id`,
		facilityID, studentID, sport, strings.TrimSpace(in.PlanName), start, end, fee,
	).Scan(&enrollmentID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetEnrollment(ctx, enrollmentID)
}

func (s *Service) GetEnrollment(ctx context.Context, enrollmentID int64) (*StudentRow, error) {
	var r StudentRow
	var sport sql.NullInt64
	var fee sql.NullFloat64
	err := s.DB.QueryRowContext(ctx, `
SELECT e.id, s.id, e.facility_id, s.first_name, s.last_name,
       COALESCE(s.contact_email,''), COALESCE(s.contact_phone,''), COALESCE(s.whatsapp,''),
       e.sport_id, COALESCE(e.plan_name,''),
       COALESCE(e.start_date::text,''), COALESCE(e.end_date::text,''), e.actual_fee,
       e.status, e.created_at::text
FROM student_enrollments e
JOIN students s ON s.id = e.student_id
WHERE e.id=$1`, enrollmentID).Scan(
		&r.EnrollmentID, &r.StudentID, &r.FacilityID, &r.FirstName, &r.LastName,
		&r.Email, &r.Phone, &r.WhatsApp, &sport, &r.PlanName,
		&r.StartDate, &r.EndDate, &fee, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sport.Valid {
		v := sport.Int64
		r.SportID = &v
	}
	if fee.Valid {
		v := fee.Float64
		r.ActualFee = &v
	}
	return &r, nil
}

func (s *Service) ListAttendance(ctx context.Context, facilityID int64, date string) ([]AttendanceRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT e.id, s.id, e.facility_id, s.first_name, s.last_name,
       COALESCE(s.contact_phone,''), COALESCE(e.plan_name,''),
       $2::text,
       COALESCE(a.status,''), a.id, COALESCE(a.notes,'')
FROM student_enrollments e
JOIN students s ON s.id = e.student_id
LEFT JOIN student_attendance a
  ON a.enrollment_id = e.id AND a.attendance_date = $2::date
WHERE e.facility_id=$1 AND e.status='active'
ORDER BY s.first_name, s.last_name, e.id`, facilityID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttendanceRow{}
	for rows.Next() {
		var r AttendanceRow
		var attID sql.NullInt64
		if err := rows.Scan(
			&r.EnrollmentID, &r.StudentID, &r.FacilityID, &r.FirstName, &r.LastName,
			&r.Phone, &r.PlanName, &r.AttendanceDate, &r.Status, &attID, &r.Notes,
		); err != nil {
			return nil, err
		}
		if attID.Valid {
			v := attID.Int64
			r.AttendanceID = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) MarkAttendance(ctx context.Context, facilityID int64, in MarkAttendanceInput) (*AttendanceRow, error) {
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if facilityID <= 0 || in.EnrollmentID <= 0 {
		return nil, fmt.Errorf("invalid_attendance")
	}
	if status != "present" && status != "absent" && status != "leave" {
		return nil, fmt.Errorf("invalid_status")
	}
	if err := validateDate(in.Date); err != nil {
		return nil, err
	}

	var enrFacility int64
	var enrStatus string
	err := s.DB.QueryRowContext(ctx, `
SELECT facility_id, status FROM student_enrollments WHERE id=$1`, in.EnrollmentID).
		Scan(&enrFacility, &enrStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("enrollment_not_found")
	}
	if err != nil {
		return nil, err
	}
	if enrFacility != facilityID {
		return nil, fmt.Errorf("facility_mismatch")
	}

	var markedBy interface{}
	if in.MarkedBy != nil {
		markedBy = *in.MarkedBy
	}

	_, err = s.DB.ExecContext(ctx, `
INSERT INTO student_attendance (facility_id, enrollment_id, attendance_date, status, marked_by, notes)
VALUES ($1,$2,$3::date,$4,$5,NULLIF($6,''))
ON CONFLICT (enrollment_id, attendance_date) DO UPDATE
SET status=EXCLUDED.status,
    marked_by=EXCLUDED.marked_by,
    notes=EXCLUDED.notes,
    updated_at=NOW()`,
		facilityID, in.EnrollmentID, in.Date, status, markedBy, strings.TrimSpace(in.Notes),
	)
	if err != nil {
		return nil, err
	}

	// Present / late-as-leave on inactive → reactivate (legacy mark_attendance_status1).
	if status == "present" && enrStatus != "active" {
		_, _ = s.DB.ExecContext(ctx, `
UPDATE student_enrollments SET status='active', updated_at=NOW() WHERE id=$1`, in.EnrollmentID)
	}

	list, err := s.ListAttendance(ctx, facilityID, in.Date)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].EnrollmentID == in.EnrollmentID {
			return &list[i], nil
		}
	}
	// Inactive enrollments are omitted from ListAttendance — synthesize row.
	row, gerr := s.GetEnrollment(ctx, in.EnrollmentID)
	if gerr != nil {
		return nil, fmt.Errorf("attendance_not_found")
	}
	return &AttendanceRow{
		EnrollmentID:   row.EnrollmentID,
		StudentID:      row.StudentID,
		FacilityID:     row.FacilityID,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Phone:          row.Phone,
		PlanName:       row.PlanName,
		AttendanceDate: in.Date,
		Status:         status,
		Notes:          strings.TrimSpace(in.Notes),
	}, nil
}

func (s *Service) UpdateEnrollmentStatus(ctx context.Context, facilityID, enrollmentID int64, status string) (*StudentRow, error) {
	st := strings.ToLower(strings.TrimSpace(status))
	if facilityID <= 0 || enrollmentID <= 0 {
		return nil, fmt.Errorf("invalid_enrollment")
	}
	if st != "active" && st != "inactive" && st != "completed" {
		return nil, fmt.Errorf("invalid_status")
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE student_enrollments SET status=$1, updated_at=NOW()
WHERE id=$2 AND facility_id=$3`, st, enrollmentID, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("enrollment_not_found")
	}
	return s.GetEnrollment(ctx, enrollmentID)
}

func (s *Service) UpdateStudent(ctx context.Context, facilityID, studentID int64, in UpdateStudentInput) (*StudentRow, error) {
	if facilityID <= 0 || studentID <= 0 {
		return nil, fmt.Errorf("invalid_student")
	}
	first := strings.TrimSpace(in.FirstName)
	last := strings.TrimSpace(in.LastName)
	if first == "" {
		return nil, fmt.Errorf("invalid_student")
	}
	// Ensure student belongs to facility via an enrollment.
	var enrollmentID int64
	err := s.DB.QueryRowContext(ctx, `
SELECT id FROM student_enrollments WHERE facility_id=$1 AND student_id=$2
ORDER BY id DESC LIMIT 1`, facilityID, studentID).Scan(&enrollmentID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("student_not_found")
	}
	if err != nil {
		return nil, err
	}
	_, err = s.DB.ExecContext(ctx, `
UPDATE students SET
  first_name=$1, last_name=$2,
  contact_email=NULLIF($3,''), contact_phone=NULLIF($4,''), whatsapp=NULLIF($5,''),
  updated_at=NOW()
WHERE id=$6`,
		first, last,
		strings.TrimSpace(in.Email),
		strings.TrimSpace(in.Phone),
		strings.TrimSpace(in.WhatsApp),
		studentID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetEnrollment(ctx, enrollmentID)
}

func (s *Service) UpdatePlan(ctx context.Context, facilityID, enrollmentID int64, in UpdatePlanInput) (*StudentRow, error) {
	if facilityID <= 0 || enrollmentID <= 0 {
		return nil, fmt.Errorf("invalid_enrollment")
	}
	plan := strings.TrimSpace(in.PlanName)
	if in.StartDate != "" {
		if err := validateDate(in.StartDate); err != nil {
			return nil, err
		}
	}
	if in.EndDate != "" {
		if err := validateDate(in.EndDate); err != nil {
			return nil, err
		}
	}
	var start, end, fee interface{}
	if strings.TrimSpace(in.StartDate) != "" {
		start = in.StartDate
	}
	if strings.TrimSpace(in.EndDate) != "" {
		end = in.EndDate
	}
	if in.ActualFee != nil {
		fee = *in.ActualFee
	}
	res, err := s.DB.ExecContext(ctx, `
UPDATE student_enrollments SET
  plan_name=NULLIF($1,''),
  start_date=COALESCE($2::date, start_date),
  end_date=COALESCE($3::date, end_date),
  actual_fee=COALESCE($4, actual_fee),
  updated_at=NOW()
WHERE id=$5 AND facility_id=$6`,
		plan, start, end, fee, enrollmentID, facilityID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("enrollment_not_found")
	}
	return s.GetEnrollment(ctx, enrollmentID)
}

func (s *Service) ListPlans(ctx context.Context, facilityID int64) ([]PlanRow, error) {
	if facilityID <= 0 {
		return nil, fmt.Errorf("invalid_facility")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT COALESCE(NULLIF(TRIM(plan_name),''),'(No batch)') AS plan_name, COUNT(*)::int
FROM student_enrollments
WHERE facility_id=$1
GROUP BY 1
ORDER BY 1`, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlanRow{}
	var id int64 = 1
	for rows.Next() {
		var p PlanRow
		if err := rows.Scan(&p.PlanName, &p.Count); err != nil {
			return nil, err
		}
		p.PlanID = id
		id++
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) CreatePlan(ctx context.Context, facilityID int64, planName string) (*PlanRow, error) {
	name := strings.TrimSpace(planName)
	if facilityID <= 0 || name == "" {
		return nil, fmt.Errorf("invalid_plan")
	}
	// Plans are synthetic (plan_name on enrollments). Creating a plan is a no-op
	// registry row — return the name so UI can assign it on enroll.
	return &PlanRow{PlanID: time.Now().Unix()%100000 + 1, PlanName: name, Count: 0}, nil
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
