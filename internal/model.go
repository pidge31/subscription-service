package main

import "time"

type Subscription struct {
	ID          string    `db:"id" json:"id"`
	ServiceName string    `db:"service_name" json:"service_name"`
	Price       int       `db:"price" json:"price"`
	UserID      string    `db:"user_id" json:"user_id"`
	StartDate   string    `db:"start_date" json:"start_date"`
	EndDate     *string   `db:"end_date" json:"end_date,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type CreateSubscriptionInput struct {
	ServiceName string  `json:"service_name" binding:"required"`
	Price       int     `json:"price" binding:"required"`
	UserID      string  `json:"user_id" binding:"required"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     *string `json:"end_date"`
}

type UpdateSubscriptionInput struct {
	ServiceName *string `json:"service_name"`
	Price       *int    `json:"price"`
	EndDate     *string `json:"end_date"`
}
