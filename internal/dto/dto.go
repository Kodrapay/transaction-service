package dto

import "time"

// TransactionCreateRequest DTO for creating a new transaction
type TransactionCreateRequest struct {
	Reference     string `json:"reference,omitempty"`
	MerchantID    int `json:"merchant_id"`
	CustomerEmail string `json:"customer_email,omitempty"`
	CustomerID    int `json:"customer_id"`
	CustomerName  string `json:"customer_name,omitempty"`
	Amount        float64  `json:"amount"` // currency units (e.g., NGN)
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status,omitempty"`
}

// TransactionResponse DTO for returning transaction information
type TransactionResponse struct {
	ID            int       `json:"id"`
	Reference     string    `json:"reference"`
	MerchantID    int       `json:"merchant_id"`
	CustomerEmail string    `json:"customer_email"`
	CustomerID    int       `json:"customer_id"`
	CustomerName  string    `json:"customer_name,omitempty"`
	Amount        float64     `json:"amount"` // currency units (e.g., NGN)
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// TransactionListResponse DTO for returning a list of transactions
type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Total        int                   `json:"total"`
}
