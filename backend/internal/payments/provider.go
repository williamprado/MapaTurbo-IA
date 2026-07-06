package payments

import "context"

type CreateCustomerInput struct {
	Name     string
	Email    string
	Document string // CPF or CNPJ
	Phone    string
}

type CustomerResult struct {
	ExternalCustomerID string
	RawPayload         interface{}
}

type CreateCheckoutInput struct {
	ExternalCustomerID string
	PlanName           string
	Amount             float64
	Currency           string
	BillingType        string // PIX, BOLETO, CREDIT_CARD
	SuccessURL         string
	CancelURL          string
	ExternalInvoiceID  string
}

type CheckoutResult struct {
	ExternalInvoiceID string
	InvoiceURL        string
	BankSlipURL       string
	PixQRCode         string
	PixCopyPaste      string
	RawPayload        interface{}
}

type CreateSubscriptionInput struct {
	ExternalCustomerID string
	PlanName           string
	Amount             float64
	Currency           string
	BillingType        string
	Cycle              string // WEEKLY, MONTHLY, SEMIANNUALLY, ANNUALLY
	ExternalInvoiceID  string
}

type SubscriptionResult struct {
	ExternalSubscriptionID string
	ExternalInvoiceID      string
	InvoiceURL             string
	PixQRCode              string
	PixCopyPaste           string
	RawPayload             interface{}
}

type PaymentStatusResult struct {
	ExternalID string
	Status     string // PAID, PENDING, FAILED, CANCELED
}

type WebhookResult struct {
	EventName              string // PAYMENT_CONFIRMED, PAYMENT_OVERDUE, etc.
	ExternalInvoiceID      string
	ExternalSubscriptionID string
	Amount                 float64
	BillingType            string
	PaymentMethod          string
	PaidAt                 string
	RawPayload             interface{}
}

type PaymentProvider interface {
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CustomerResult, error)
	CreateCheckout(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error)
	CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionResult, error)
	CancelSubscription(ctx context.Context, externalSubscriptionID string) error
	GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatusResult, error)
	HandleWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)
}
