package wallets

type CreditRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0,lte=100000000"`
	Reason string  `json:"reason" binding:"omitempty,max=100"`
}

type WalletResponse struct {
	Wallet       Wallet        `json:"wallet"`
	Transactions []Transaction `json:"transactions"`
}

type WalletCreditResponse struct {
	Wallet Wallet `json:"wallet"`
}
