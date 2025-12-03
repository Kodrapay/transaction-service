package dto

type TransactionCreateRequest struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	CustomerID  string `json:"customer_id"`
	MerchantID  string `json:"merchant_id"`
	Description string `json:"description"`
}

type TransactionResponse struct {
	Reference string `json:"reference"`
	Status    string `json:"status"`
}
