package payments

import (
	"context"
	"errors"
)

type StripeProvider struct {
	apiKey  string
	sandbox bool
}

func NewStripeProvider(apiKey string, sandbox bool) *StripeProvider {
	return &StripeProvider{
		apiKey:  apiKey,
		sandbox: sandbox,
	}
}

func (p *StripeProvider) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CustomerResult, error) {
	return &CustomerResult{
		ExternalCustomerID: "stripe-cust-placeholder",
		RawPayload:         nil,
	}, nil
}

func (p *StripeProvider) CreateCheckout(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error) {
	return &CheckoutResult{
		ExternalInvoiceID: "stripe-chkt-placeholder",
		InvoiceURL:        "https://checkout.stripe.com/pay/chkt-placeholder",
		RawPayload:         nil,
	}, nil
}

func (p *StripeProvider) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionResult, error) {
	return &SubscriptionResult{
		ExternalSubscriptionID: "stripe-sub-placeholder",
		ExternalInvoiceID:      "stripe-inv-placeholder",
		InvoiceURL:             "https://billing.stripe.com/sub-placeholder",
		RawPayload:             nil,
	}, nil
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, externalSubscriptionID string) error {
	return nil
}

func (p *StripeProvider) GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatusResult, error) {
	return &PaymentStatusResult{
		ExternalID: externalID,
		Status:     "PENDING",
	}, nil
}

func (p *StripeProvider) HandleWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return nil, errors.New("stripe webhook handling not implemented yet")
}
