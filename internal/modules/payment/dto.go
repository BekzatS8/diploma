package payment

type CreatePaymentRequest struct {
	OrderID string  `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
}

type CreatePaymentResponse struct {
	TransactionID     string  `json:"transaction_id"`
	YooKassaPaymentID string  `json:"yookassa_payment_id"`
	Status            string  `json:"status"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	ConfirmationURL   string  `json:"confirmation_url"`
}
