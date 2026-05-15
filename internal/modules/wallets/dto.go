package wallets

type CreditRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason"`
}

type WalletResponse struct {
	Wallet       Wallet        `json:"wallet"`
	Transactions []Transaction `json:"transactions"`
}
