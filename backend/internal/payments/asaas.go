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
)

type AsaasProvider struct {
	apiKey  string
	baseURL string
}

func NewAsaasProvider(apiKey, baseURL string) *AsaasProvider {
	if baseURL == "" {
		baseURL = "https://sandbox.asaas.com/api/v3"
	}
	return &AsaasProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (p *AsaasProvider) doRequest(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := fmt.Sprintf("%s%s", p.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("access_token", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("asaas api error: status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	if out != nil {
		return json.Unmarshal(respBytes, out)
	}
	return nil
}

type AsaasCreateCustomerReq struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	CpfCnpj string `json:"cpfCnpj"`
	Phone   string `json:"phone,omitempty"`
}

type AsaasCustomerResp struct {
	ID string `json:"id"`
}

func (p *AsaasProvider) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CustomerResult, error) {
	reqBody := AsaasCreateCustomerReq{
		Name:    input.Name,
		Email:   input.Email,
		CpfCnpj: input.Document,
		Phone:   input.Phone,
	}

	var respBody AsaasCustomerResp
	err := p.doRequest(ctx, "POST", "/customers", reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	return &CustomerResult{
		ExternalCustomerID: respBody.ID,
		RawPayload:         respBody,
	}, nil
}

type AsaasCreatePaymentReq struct {
	Customer          string  `json:"customer"`
	BillingType       string  `json:"billingType"`
	Value             float64 `json:"value"`
	DueDate           string  `json:"dueDate"`
	ExternalReference string  `json:"externalReference,omitempty"`
}

type AsaasPaymentResp struct {
	ID         string `json:"id"`
	InvoiceUrl string `json:"invoiceUrl"`
	BankSlipUrl string `json:"bankSlipUrl"`
	Status     string `json:"status"`
}

type AsaasPixQrCodeResp struct {
	Success      bool   `json:"success"`
	EncodedImage string `json:"encodedImage"`
	Payload      string `json:"payload"`
}

func (p *AsaasProvider) CreateCheckout(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error) {
	dueDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	reqBody := AsaasCreatePaymentReq{
		Customer:          input.ExternalCustomerID,
		BillingType:       input.BillingType,
		Value:             input.Amount,
		DueDate:           dueDate,
		ExternalReference: input.ExternalInvoiceID,
	}

	var respBody AsaasPaymentResp
	err := p.doRequest(ctx, "POST", "/payments", reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	result := &CheckoutResult{
		ExternalInvoiceID: respBody.ID,
		InvoiceURL:        respBody.InvoiceUrl,
		BankSlipURL:       respBody.BankSlipUrl,
		RawPayload:        respBody,
	}

	if input.BillingType == "PIX" {
		var pixResp AsaasPixQrCodeResp
		err = p.doRequest(ctx, "GET", fmt.Sprintf("/payments/%s/pixQrCode", respBody.ID), nil, &pixResp)
		if err == nil && pixResp.Success {
			result.PixQRCode = pixResp.EncodedImage
			result.PixCopyPaste = pixResp.Payload
		}
	}

	return result, nil
}

type AsaasCreateSubscriptionReq struct {
	Customer          string  `json:"customer"`
	BillingType       string  `json:"billingType"`
	Value             float64 `json:"value"`
	NextDueDate       string  `json:"nextDueDate"`
	Cycle             string  `json:"cycle"`
	ExternalReference string  `json:"externalReference,omitempty"`
}

type AsaasSubscriptionResp struct {
	ID string `json:"id"`
}

func (p *AsaasProvider) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionResult, error) {
	nextDueDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	reqBody := AsaasCreateSubscriptionReq{
		Customer:          input.ExternalCustomerID,
		BillingType:       input.BillingType,
		Value:             input.Amount,
		NextDueDate:       nextDueDate,
		Cycle:             input.Cycle,
		ExternalReference: input.ExternalInvoiceID,
	}

	var respBody AsaasSubscriptionResp
	err := p.doRequest(ctx, "POST", "/subscriptions", reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	type AsaasListPaymentsResp struct {
		Data []AsaasPaymentResp `json:"data"`
	}
	var listResp AsaasListPaymentsResp
	var extInvoiceID string
	var invoiceURL string

	err = p.doRequest(ctx, "GET", fmt.Sprintf("/payments?subscription=%s", respBody.ID), nil, &listResp)
	if err == nil && len(listResp.Data) > 0 {
		firstPayment := listResp.Data[0]
		extInvoiceID = firstPayment.ID
		invoiceURL = firstPayment.InvoiceUrl
	} else {
		extInvoiceID = "sub-inv-" + respBody.ID
	}

	result := &SubscriptionResult{
		ExternalSubscriptionID: respBody.ID,
		ExternalInvoiceID:      extInvoiceID,
		InvoiceURL:             invoiceURL,
		RawPayload:             respBody,
	}

	if input.BillingType == "PIX" && extInvoiceID != "" && !strings.HasPrefix(extInvoiceID, "sub-inv-") {
		var pixResp AsaasPixQrCodeResp
		err = p.doRequest(ctx, "GET", fmt.Sprintf("/payments/%s/pixQrCode", extInvoiceID), nil, &pixResp)
		if err == nil && pixResp.Success {
			result.PixQRCode = pixResp.EncodedImage
			result.PixCopyPaste = pixResp.Payload
		}
	}

	return result, nil
}

func (p *AsaasProvider) CancelSubscription(ctx context.Context, externalSubscriptionID string) error {
	return p.doRequest(ctx, "DELETE", fmt.Sprintf("/subscriptions/%s", externalSubscriptionID), nil, nil)
}

func (p *AsaasProvider) GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatusResult, error) {
	var respBody AsaasPaymentResp
	err := p.doRequest(ctx, "GET", fmt.Sprintf("/payments/%s", externalID), nil, &respBody)
	if err != nil {
		return nil, err
	}

	var status string
	switch respBody.Status {
	case "RECEIVED", "CONFIRMED":
		status = "PAID"
	case "OVERDUE":
		status = "FAILED"
	case "REFUNDED", "CHARGEBACK_REQUESTED":
		status = "CANCELED"
	default:
		status = "PENDING"
	}

	return &PaymentStatusResult{
		ExternalID: externalID,
		Status:     status,
	}, nil
}

type AsaasWebhookPayload struct {
	Event   string `json:"event"`
	Payment struct {
		ID                string  `json:"id"`
		Customer          string  `json:"customer"`
		Subscription      string  `json:"subscription"`
		Value             float64 `json:"value"`
		BillingType       string  `json:"billingType"`
		Status            string  `json:"status"`
		ExternalReference string  `json:"externalReference"`
		ClientPaymentDate string  `json:"clientPaymentDate"`
	} `json:"payment"`
}

func (p *AsaasProvider) HandleWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	var wrapper AsaasWebhookPayload
	if err := json.Unmarshal(payload, &wrapper); err != nil {
		return nil, err
	}

	var eventName string
	switch wrapper.Event {
	case "PAYMENT_CONFIRMED", "PAYMENT_RECEIVED":
		eventName = "PAYMENT_CONFIRMED"
	case "PAYMENT_OVERDUE":
		eventName = "PAYMENT_OVERDUE"
	case "PAYMENT_DELETED":
		eventName = "PAYMENT_CANCELED"
	default:
		eventName = wrapper.Event
	}

	return &WebhookResult{
		EventName:              eventName,
		ExternalInvoiceID:      wrapper.Payment.ID,
		ExternalSubscriptionID: wrapper.Payment.Subscription,
		Amount:                 wrapper.Payment.Value,
		BillingType:            wrapper.Payment.BillingType,
		PaymentMethod:          wrapper.Payment.BillingType,
		PaidAt:                 wrapper.Payment.ClientPaymentDate,
		RawPayload:             wrapper,
	}, nil
}
