package payments

import (
	"context"
	"errors"
)

type PaypalProvider struct {
	clientID     string
	clientSecret string
	sandbox      bool
}

func NewPaypalProvider(clientID, clientSecret string, sandbox bool) *PaypalProvider {
	return &PaypalProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		sandbox:      sandbox,
	}
}

func (p *PaypalProvider) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CustomerResult, error) {
	return &CustomerResult{
		ExternalCustomerID: "paypal-cust-placeholder",
		RawPayload:         nil,
	}, nil
}

func (p *PaypalProvider) CreateCheckout(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error) {
	return &CheckoutResult{
		ExternalInvoiceID: "paypal-chkt-placeholder",
		InvoiceURL:        "https://www.paypal.com/checkout/chkt-placeholder",
		RawPayload:         nil,
	}, nil
}

func (p *PaypalProvider) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionResult, error) {
	return &SubscriptionResult{
		ExternalSubscriptionID: "paypal-sub-placeholder",
		ExternalInvoiceID:      "paypal-inv-placeholder",
		InvoiceURL:             "https://www.paypal.com/billing/sub-placeholder",
		RawPayload:             nil,
	}, nil
}

func (p *PaypalProvider) CancelSubscription(ctx context.Context, externalSubscriptionID string) error {
	return nil
}

func (p *PaypalProvider) GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatusResult, error) {
	return &PaymentStatusResult{
		ExternalID: externalID,
		Status:     "PENDING",
	}, nil
}

func (p *PaypalProvider) HandleWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return nil, errors.New("paypal webhook handling not implemented yet")
}
