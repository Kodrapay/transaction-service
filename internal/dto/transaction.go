package dto

type TransactionCreateRequest struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	CustomerID    string `json:"customer_id,omitempty"`    // legacy field (email)
	CustomerEmail string `json:"customer_email,omitempty"` // preferred email field
	MerchantID    string `json:"merchant_id"`
	Description   string `json:"description"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Reference     string `json:"reference,omitempty"`
}

type TransactionResponse struct {
	ID            string `json:"id"`
	Reference     string `json:"reference"`
	MerchantID    string `json:"merchant_id"`
	CustomerEmail string `json:"customer_email,omitempty"`
	CustomerName  string `json:"customer_name,omitempty"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	Description   string `json:"description,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
}
