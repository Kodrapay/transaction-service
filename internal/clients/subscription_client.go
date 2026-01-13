package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SubscriptionClient interface {
	CalculateFee(ctx context.Context, req *FeeQuoteRequest) (*FeeQuoteResponse, error)
}

type subscriptionClient struct {
	baseURL    string
	httpClient *http.Client
}

type FeeQuoteRequest struct {
	MerchantID    int64   `json:"merchant_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	Currency      string  `json:"currency"`
}

type FeeQuoteResponse struct {
	TotalFee          int64   `json:"total_fee"`           // Fee in kobo
	BaseAmount        int64   `json:"base_amount"`         // Original amount in kobo
	MerchantNet       int64   `json:"merchant_net"`        // Amount merchant receives in kobo
	Currency          string  `json:"currency"`
	FeePercentage     float64 `json:"fee_percentage"`
	FeeFlat           int64   `json:"fee_flat"`
	FeeCap            *int64  `json:"fee_cap,omitempty"`
	FeeCapped         bool    `json:"fee_capped"`
	PaymentMethod     string  `json:"payment_method"`
	TierName          string  `json:"tier_name"`
	UsedCustomPricing bool    `json:"used_custom_pricing"`
}

func NewSubscriptionClient(baseURL string) SubscriptionClient {
	return &subscriptionClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *subscriptionClient) CalculateFee(ctx context.Context, req *FeeQuoteRequest) (*FeeQuoteResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/fees/calculate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call subscription service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("subscription service returned status %d: %s", resp.StatusCode, string(body))
	}

	var feeResponse FeeQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&feeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &feeResponse, nil
}
