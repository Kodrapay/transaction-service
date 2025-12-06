package dto

import "time"

// TransactionCreateRequest DTO for creating a new transaction
type TransactionCreateRequest struct {
	Reference     string `json:"reference,omitempty"`
	MerchantID    string `json:"merchant_id"`
	CustomerEmail string `json:"customer_email,omitempty"`
	CustomerID    string `json:"customer_id"` // Added CustomerID
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status,omitempty"`
}

// TransactionResponse DTO for returning transaction information
type TransactionResponse struct {
	ID            string    `json:"id"`
	Reference     string    `json:"reference"`
	MerchantID    string    `json:"merchant_id"`
	CustomerEmail string    `json:"customer_email"`
	CustomerID    string    `json:"customer_id"` // Added CustomerID
	CustomerName  string    `json:"customer_name,omitempty"`
	Amount        int64     `json:"amount"`
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
