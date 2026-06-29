package app

import "time"

type Subscription struct {
	ID          string    `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      string    `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type subscriptionRow struct {
	ID          string     `db:"id"`
	ServiceName string     `db:"service_name"`
	Price       int        `db:"price"`
	UserID      string     `db:"user_id"`
	StartDate   time.Time  `db:"start_date"`
	EndDate     *time.Time `db:"end_date"`
	CreatedAt   time.Time  `db:"created_at"`
}

func (r subscriptionRow) toSubscription() Subscription {
	s := Subscription{
		ID:          r.ID,
		ServiceName: r.ServiceName,
		Price:       r.Price,
		UserID:      r.UserID,
		StartDate:   r.StartDate.Format("01-2006"),
		CreatedAt:   r.CreatedAt,
	}
	if r.EndDate != nil {
		ed := r.EndDate.Format("01-2006")
		s.EndDate = &ed
	}
	return s
}

func parseMonthYear(s string) (time.Time, error) {
	return time.ParseInLocation("01-2006", s, time.UTC)
}

type CreateSubscriptionInput struct {
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
}

type UpdateSubscriptionInput struct {
	ServiceName *string `json:"service_name"`
	Price       *int    `json:"price"`
	EndDate     *string `json:"end_date"`
}

type TotalCostFilter struct {
	UserID      string
	ServiceName string
	From        string
	To          string
}

type TotalCostResponse struct {
	Total int `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
