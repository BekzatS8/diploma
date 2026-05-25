package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const yookassaPaymentsURL = "https://api.yookassa.ru/v3/payments"

type YooKassaConfig struct {
	ShopID    string
	SecretKey string
	ReturnURL string
}

type YooKassaProvider struct {
	shopID    string
	secretKey string
	returnURL string
	client    *http.Client
}

func NewYooKassa(cfg YooKassaConfig) (*YooKassaProvider, error) {
	shopID := strings.TrimSpace(cfg.ShopID)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	if shopID == "" || secretKey == "" {
		return nil, fmt.Errorf("yookassa: shop id and secret key are required")
	}
	return &YooKassaProvider{
		shopID:    shopID,
		secretKey: secretKey,
		returnURL: strings.TrimSpace(cfg.ReturnURL),
		client:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type yookassaCreatePaymentRequest struct {
	Amount       yookassaAmount       `json:"amount"`
	Capture      bool                 `json:"capture"`
	Confirmation yookassaConfirmation `json:"confirmation"`
	Description  string               `json:"description,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
}

type yookassaAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type yookassaConfirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url,omitempty"`
}

type yookassaCreatePaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

type yookassaErrorResponse struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter"`
}

func (p *YooKassaProvider) CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResponse, error) {
	if req.AmountCents <= 0 {
		return ChargeResponse{}, fmt.Errorf("yookassa: amount must be positive")
	}
	currency := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if currency == "" {
		currency = "RUB"
	}

	payload := yookassaCreatePaymentRequest{
		Amount: yookassaAmount{
			Value:    formatAmount(req.AmountCents),
			Currency: currency,
		},
		Capture: true,
		Confirmation: yookassaConfirmation{
			Type:      "redirect",
			ReturnURL: p.returnURL,
		},
		Description: req.Description,
		Metadata: map[string]string{
			"order_id": req.OrderID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("yookassa: marshal create payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, yookassaPaymentsURL, bytes.NewReader(body))
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("yookassa: build create payment request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", uuid.NewString())
	httpReq.SetBasicAuth(p.shopID, p.secretKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("yookassa: create payment request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("yookassa: read create payment response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr yookassaErrorResponse
		if decodeErr := json.Unmarshal(raw, &apiErr); decodeErr == nil && apiErr.Description != "" {
			return ChargeResponse{}, fmt.Errorf("yookassa: create payment failed: %s (%s)", apiErr.Description, apiErr.Code)
		}
		return ChargeResponse{}, fmt.Errorf("yookassa: create payment failed with HTTP %d", resp.StatusCode)
	}

	var out yookassaCreatePaymentResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return ChargeResponse{}, fmt.Errorf("yookassa: decode create payment response: %w", err)
	}
	if out.ID == "" || out.Confirmation.ConfirmationURL == "" {
		return ChargeResponse{}, fmt.Errorf("yookassa: create payment response missing payment id or confirmation url")
	}
	return ChargeResponse{
		TransactionID: out.ID,
		Status:        out.Status,
		RedirectURL:   out.Confirmation.ConfirmationURL,
	}, nil
}

func (p *YooKassaProvider) VerifyWebhook(_ context.Context, _ []byte, _ string) error {
	return nil
}

func formatAmount(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
