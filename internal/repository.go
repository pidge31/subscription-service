package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, in CreateSubscriptionInput) (Subscription, error) {
	startDate, err := parseMonthYear(in.StartDate)
	if err != nil {
		return Subscription{}, fmt.Errorf("invalid start_date: %w", err)
	}

	var endDateStr *string
	if in.EndDate != nil {
		ed, err := parseMonthYear(*in.EndDate)
		if err != nil {
			return Subscription{}, fmt.Errorf("invalid end_date: %w", err)
		}
		s := ed.Format("2006-01-02")
		endDateStr = &s
	}

	row := subscriptionRow{}
	err = r.db.QueryRowxContext(ctx, `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at
	`, in.ServiceName, in.Price, in.UserID, startDate.Format("2006-01-02"), endDateStr).
		StructScan(&row)
	if err != nil {
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}
	return row.toSubscription(), nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (Subscription, error) {
	row := subscriptionRow{}
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at
		FROM subscriptions WHERE id = $1
	`, id).StructScan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("get subscription: %w", err)
	}
	return row.toSubscription(), nil
}

func (r *Repository) List(ctx context.Context, userID, serviceName string) ([]Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", idx)
		args = append(args, userID)
		idx++
	}
	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name = $%d", idx)
		args = append(args, serviceName)
		idx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		row := subscriptionRow{}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, row.toSubscription())
	}
	return subs, nil
}

func (r *Repository) Update(ctx context.Context, id string, in UpdateSubscriptionInput) (Subscription, error) {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if in.ServiceName != nil {
		sets = append(sets, fmt.Sprintf("service_name = $%d", idx))
		args = append(args, *in.ServiceName)
		idx++
	}
	if in.Price != nil {
		sets = append(sets, fmt.Sprintf("price = $%d", idx))
		args = append(args, *in.Price)
		idx++
	}
	if in.EndDate != nil {
		ed, err := parseMonthYear(*in.EndDate)
		if err != nil {
			return Subscription{}, fmt.Errorf("invalid end_date: %w", err)
		}
		sets = append(sets, fmt.Sprintf("end_date = $%d", idx))
		args = append(args, ed.Format("2006-01-02"))
		idx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE subscriptions SET %s WHERE id = $%d
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at
	`, strings.Join(sets, ", "), idx)

	row := subscriptionRow{}
	err := r.db.QueryRowxContext(ctx, query, args...).StructScan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("update subscription: %w", err)
	}
	return row.toSubscription(), nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) TotalCost(ctx context.Context, f TotalCostFilter) (int, error) {
	fromDate, err := parseMonthYear(f.From)
	if err != nil {
		return 0, fmt.Errorf("invalid from: %w", err)
	}
	toDate, err := parseMonthYear(f.To)
	if err != nil {
		return 0, fmt.Errorf("invalid to: %w", err)
	}

	lastDay := time.Date(toDate.Year(), toDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	conds := []string{
		"s.start_date <= $2",
		"(s.end_date IS NULL OR s.end_date >= $1)",
	}
	args := []interface{}{fromDate.Format("2006-01-02"), lastDay.Format("2006-01-02")}
	idx := 3

	if f.UserID != "" {
		conds = append(conds, fmt.Sprintf("s.user_id = $%d", idx))
		args = append(args, f.UserID)
		idx++
	}
	if f.ServiceName != "" {
		conds = append(conds, fmt.Sprintf("s.service_name = $%d", idx))
		args = append(args, f.ServiceName)
		idx++
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(
			s.price * (
				(EXTRACT(YEAR  FROM LEAST(COALESCE(s.end_date, $2::date), $2::date))::int
				- EXTRACT(YEAR  FROM GREATEST(s.start_date, $1::date))::int) * 12
				+ (EXTRACT(MONTH FROM LEAST(COALESCE(s.end_date, $2::date), $2::date))::int
				- EXTRACT(MONTH FROM GREATEST(s.start_date, $1::date))::int) + 1
			)
		)::int, 0) AS total
		FROM subscriptions s
		WHERE %s
	`, strings.Join(conds, " AND "))

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("total cost query: %w", err)
	}
	return total, nil
}
