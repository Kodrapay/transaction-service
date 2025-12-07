package models

import "time"

type Transaction struct {
	ID            int       `json:"id"`
	Reference     string    `json:"reference"` // Changed from int to string
	MerchantID    int       `json:"merchant_id"`
	CustomerEmail string    `json:"customer_email,omitempty"`
	CustomerID    int       `json:"customer_id,omitempty"` // Added CustomerID
	CustomerName  string    `json:"customer_name,omitempty"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
