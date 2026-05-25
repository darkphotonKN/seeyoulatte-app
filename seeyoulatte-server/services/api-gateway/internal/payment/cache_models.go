package payment

// PaymentProcessorCacheData represents the cached payment processor data
type PaymentProcessorCacheData struct {
	CustomerData CustomerDataCache `json:"customer_data"`
	Payments     []*PaymentCache   `json:"payments"`
}

// PaymentCache represents cached payment information
type PaymentCache struct {
	IntentID string `json:"intent_id"`
	OrderID  string `json:"order_id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// CustomerDataCache represents cached customer data
type CustomerDataCache struct {
	CustomerID string `json:"customer_id"`
	Email      string `json:"email"`
	Created    int64  `json:"created,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}