package models

import "time"

type Transaction struct {
	ID           string    `json:"id"`
	Reference    string    `json:"reference"`
	MerchantID   string    `json:"merchant_id"`
	CustomerEmail string   `json:"customer_email,omitempty"`
	CustomerName string    `json:"customer_name,omitempty"`
	Amount       int64     `json:"amount"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	PaymentMethod string   `json:"payment_method,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
