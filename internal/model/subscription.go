package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a user's online subscription record.
type Subscription struct {
	ID          int        `json:"id" db:"id"`
	ServiceName string     `json:"service_name" db:"service_name"`
	Price       int        `json:"price" db:"price"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	StartDate   string     `json:"start_date" db:"start_date"`
	EndDate     *string    `json:"end_date,omitempty" db:"end_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateSubscriptionRequest is the request body for creating a subscription.
type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name" example:"Yandex Plus"`
	Price       int    `json:"price" example:"400"`
	UserID      string `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string `json:"start_date" example:"07-2025"`
	EndDate     string `json:"end_date,omitempty" example:"12-2025"`
}

// UpdateSubscriptionRequest is the request body for updating a subscription.
type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty" example:"Yandex Plus"`
	Price       *int    `json:"price,omitempty" example:"500"`
	StartDate   *string `json:"start_date,omitempty" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
}

// TotalCostResponse is the response for the total cost calculation.
type TotalCostResponse struct {
	TotalCost int `json:"total_cost" example:"1200"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}
