package payments

import (
	"context"
	"errors"
)

type ManualProvider struct{}

func NewManualProvider() *ManualProvider {
	return &ManualProvider{}
}

func (p *ManualProvider) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CustomerResult, error) {
	return &CustomerResult{
		ExternalCustomerID: "manual-customer",
		RawPayload:         nil,
	}, nil
}

func (p *ManualProvider) CreateCheckout(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error) {
	return &CheckoutResult{
		ExternalInvoiceID: "manual-checkout",
		InvoiceURL:        "",
		RawPayload:         nil,
	}, nil
}

func (p *ManualProvider) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionResult, error) {
	return &SubscriptionResult{
		ExternalSubscriptionID: "manual-sub",
		ExternalInvoiceID:      "manual-inv",
		InvoiceURL:             "",
		RawPayload:             nil,
	}, nil
}

func (p *ManualProvider) CancelSubscription(ctx context.Context, externalSubscriptionID string) error {
	return nil
}

func (p *ManualProvider) GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatusResult, error) {
	return &PaymentStatusResult{
		ExternalID: externalID,
		Status:     "PAID",
	}, nil
}

func (p *ManualProvider) HandleWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return nil, errors.New("manual webhook not supported")
}
