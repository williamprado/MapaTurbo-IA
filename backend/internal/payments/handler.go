package payments

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	cryptoPkg "mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/logger"
	"mapaturbo-ia/pkg/queue"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/validator"
	"go.uber.org/zap"
)

type Handler struct {
	db            *pgxpool.Pool
	queries       *database.Queries
	encryptionKey string
}

func NewHandler(db *pgxpool.Pool, encryptionKey string) *Handler {
	return &Handler{
		db:            db,
		queries:       database.New(db),
		encryptionKey: encryptionKey,
	}
}

type CheckoutRequest struct {
	PlanID      string `json:"planId" validate:"required"`
	Cycle       string `json:"cycle" validate:"required,oneof=monthly yearly"`
	BillingType string `json:"billingType" validate:"required,oneof=PIX BOLETO CREDIT_CARD"`
	Document    string `json:"document"` // CPF/CNPJ (required if customer not registered)
	Phone       string `json:"phone"`
}

func getAESKey(rawKey string) []byte {
	h := sha256.Sum256([]byte(rawKey))
	return h[:]
}

func (h *Handler) getProvider(ctx context.Context, slug string) (PaymentProvider, *database.GetPaymentProviderBySlugRow, error) {
	dbProvider, err := h.queries.GetPaymentProviderBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	if !dbProvider.IsActive {
		return nil, nil, errors.New("payment provider is inactive")
	}

	key := getAESKey(h.encryptionKey)
	var decryptedApiKey string
	if dbProvider.ApiKeySecure.Valid && dbProvider.ApiKeySecure.String != "" {
		decryptedApiKey, err = cryptoPkg.Decrypt(dbProvider.ApiKeySecure.String, key)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
	}

	var baseURL string
	if dbProvider.Mode == "sandbox" {
		baseURL = "https://sandbox.asaas.com/api/v3"
	} else {
		baseURL = "https://api.asaas.com/api/v3"
	}

	switch slug {
	case "asaas":
		return NewAsaasProvider(decryptedApiKey, baseURL), &dbProvider, nil
	case "stripe":
		return NewStripeProvider(decryptedApiKey, dbProvider.Mode == "sandbox"), &dbProvider, nil
	case "paypal":
		parts := strings.Split(decryptedApiKey, ":")
		clientID := ""
		clientSecret := ""
		if len(parts) > 0 {
			clientID = parts[0]
		}
		if len(parts) > 1 {
			clientSecret = parts[1]
		}
		return NewPaypalProvider(clientID, clientSecret, dbProvider.Mode == "sandbox"), &dbProvider, nil
	case "manual":
		return NewManualProvider(), &dbProvider, nil
	default:
		return nil, nil, fmt.Errorf("unknown payment provider: %s", slug)
	}
}

func (h *Handler) CreateCheckout(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// 1. Resolve organization context
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var userID pgtype.UUID
	_ = userID.Scan(userIDStr)

	// 2. Fetch plan details
	var planID pgtype.UUID
	if err := planID.Scan(req.PlanID); err != nil {
		response.BadRequest(c, "Invalid plan UUID format", nil)
		return
	}

	plan, err := h.queries.GetPlanByID(c.Request.Context(), planID)
	if err != nil {
		response.NotFound(c, "Selected plan not found")
		return
	}

	amount := plan.PriceMonthly
	if req.Cycle == "yearly" {
		amount = plan.PriceYearly
	}

	// 3. Instantiate payments provider (Asaas default)
	providerSlug := "asaas"
	prov, _, err := h.getProvider(c.Request.Context(), providerSlug)
	if err != nil {
		response.InternalServerError(c, "Failed to initialize payment gateway: "+err.Error())
		return
	}

	// 4. Resolve customer registry
	var extCustomer string
	dbCust, err := h.queries.GetPaymentCustomer(c.Request.Context(), database.GetPaymentCustomerParams{
		Provider:       providerSlug,
		OrganizationID: orgID,
	})
	if err != nil {
		// Register new customer in provider
		if req.Document == "" {
			response.BadRequest(c, "CPF/CNPJ is required for payment registration", nil)
			return
		}

		// Retrieve org name for customer details
		org, orgErr := h.queries.GetOrganizationByID(c.Request.Context(), orgID)
		if orgErr != nil {
			response.InternalServerError(c, "Failed to retrieve organization context")
			return
		}

		custResult, custErr := prov.CreateCustomer(c.Request.Context(), CreateCustomerInput{
			Name:     org.Name,
			Email:    c.GetString("email"),
			Document: req.Document,
			Phone:    req.Phone,
		})
		if custErr != nil {
			response.InternalServerError(c, "Failed to create customer on gateway: "+custErr.Error())
			return
		}

		extCustomer = custResult.ExternalCustomerID
		custPayload, _ := json.Marshal(custResult.RawPayload)

		// Store customer in database
		_, err = h.queries.CreatePaymentCustomer(c.Request.Context(), database.CreatePaymentCustomerParams{
			OrganizationID:     orgID,
			Provider:           providerSlug,
			ExternalCustomerID: extCustomer,
			Name:               pgtype.Text{String: org.Name, Valid: true},
			Email:              pgtype.Text{String: c.GetString("email"), Valid: true},
			Document:           pgtype.Text{String: req.Document, Valid: true},
			Phone:              pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
			Payload:            custPayload,
		})
		if err != nil {
			response.InternalServerError(c, "Failed to store customer mapping")
			return
		}
	} else {
		extCustomer = dbCust.ExternalCustomerID
	}

	var floatAmount float64
	if err := amount.Scan(&floatAmount); err != nil {
		response.InternalServerError(c, "Failed to parse plan price format")
		return
	}

	// 5. Generate local identifiers and invoke checkout API
	localInvoiceID := uuidToString(pgtype.UUID{Bytes: orgID.Bytes, Valid: true}) + "-" + string(time.Now().Format("02150405"))
	checkoutResult, err := prov.CreateCheckout(c.Request.Context(), CreateCheckoutInput{
		ExternalCustomerID: extCustomer,
		PlanName:           plan.Name,
		Amount:             floatAmount,
		Currency:           plan.Currency,
		BillingType:        req.BillingType,
		SuccessURL:         "http://localhost:5173/app/billing/success",
		CancelURL:          "http://localhost:5173/app/billing/cancel",
		ExternalInvoiceID:  localInvoiceID,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to generate charge on gateway: "+err.Error())
		return
	}

	// 6. DB transaction to write local subscription + invoice
	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to start database transaction")
		return
	}
	defer tx.Rollback(c.Request.Context())

	txQueries := h.queries.WithTx(tx)

	// Create/Update Local Subscription
	existingSub, err := txQueries.GetSubscriptionByOrg(c.Request.Context(), orgID)
	var subID pgtype.UUID
	expiresAt := time.Now().AddDate(0, 1, 0)
	if req.Cycle == "yearly" {
		expiresAt = time.Now().AddDate(1, 0, 0)
	}

	if err == nil {
		subRow, errVal := txQueries.UpdateSubscription(c.Request.Context(), database.UpdateSubscriptionParams{
			ID:                     existingSub.ID,
			PlanID:                 planID,
			Status:                 "PENDING",
			PaymentProvider:        providerSlug,
			ExternalSubscriptionID: pgtype.Text{String: checkoutResult.ExternalInvoiceID, Valid: true},
			CurrentPeriodStart:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			CurrentPeriodEnd:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		err = errVal
		if err == nil {
			subID = subRow.ID
		}
	} else {
		subRow, errVal := txQueries.CreateSubscription(c.Request.Context(), database.CreateSubscriptionParams{
			OrganizationID:         orgID,
			PlanID:                 planID,
			Status:                 "PENDING",
			PaymentProvider:        providerSlug,
			ExternalSubscriptionID: pgtype.Text{String: checkoutResult.ExternalInvoiceID, Valid: true},
			CurrentPeriodStart:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			CurrentPeriodEnd:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		err = errVal
		if err == nil {
			subID = subRow.ID
		}
	}

	if err != nil {
		response.InternalServerError(c, "Failed to update local subscription context")
		return
	}

	// Create Local Invoice
	inv, err := txQueries.CreateInvoice(c.Request.Context(), database.CreateInvoiceParams{
		OrganizationID:    orgID,
		SubscriptionID:    subID,
		Amount:            amount,
		Currency:          plan.Currency,
		Status:            "PENDING",
		ExternalInvoiceID: pgtype.Text{String: checkoutResult.ExternalInvoiceID, Valid: true},
		PdfUrl:            pgtype.Text{String: checkoutResult.BankSlipURL, Valid: checkoutResult.BankSlipURL != ""},
		DueDate:           pgtype.Timestamptz{Time: time.Now().Add(48 * time.Hour), Valid: true},
		BillingType:       pgtype.Text{String: req.BillingType, Valid: true},
		InvoiceUrl:        pgtype.Text{String: checkoutResult.InvoiceURL, Valid: checkoutResult.InvoiceURL != ""},
		BankSlipUrl:       pgtype.Text{String: checkoutResult.BankSlipURL, Valid: checkoutResult.BankSlipURL != ""},
		PixQrCode:         pgtype.Text{String: checkoutResult.PixQRCode, Valid: checkoutResult.PixQRCode != ""},
		PixCopyPaste:      pgtype.Text{String: checkoutResult.PixCopyPaste, Valid: checkoutResult.PixCopyPaste != ""},
	})
	if err != nil {
		response.InternalServerError(c, "Failed to register local invoice record")
		return
	}

	// Create Local Payment Transaction
	rawPayloadBytes, _ := json.Marshal(checkoutResult.RawPayload)
	_, err = txQueries.CreatePaymentTransaction(c.Request.Context(), database.CreatePaymentTransactionParams{
		OrganizationID:        orgID,
		InvoiceID:             pgtype.UUID{Bytes: inv.ID.Bytes, Valid: true},
		Amount:                amount,
		Provider:              providerSlug,
		ExternalTransactionID: pgtype.Text{String: checkoutResult.ExternalInvoiceID, Valid: true},
		Status:                "PENDING",
		PaymentMethod:         pgtype.Text{String: req.BillingType, Valid: true},
		Payload:               rawPayloadBytes,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to register local transaction record")
		return
	}

	// Commit Transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		response.InternalServerError(c, "Failed to commit local database records")
		return
	}

	// Log Audit
	meta, _ := json.Marshal(map[string]interface{}{
		"amount":       amount,
		"billing_type": req.BillingType,
		"plan_name":    plan.Name,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "PAYMENT_CHECKOUT_CREATED",
		EntityType:     "invoices",
		EntityID:       inv.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "Checkout session generated", gin.H{
		"invoice": inv,
		"url":     checkoutResult.InvoiceURL,
	})
}

func (h *Handler) HandleAsaasWebhook(c *gin.Context) {
	// 1. Fetch provider details
	dbProvider, err := h.queries.GetPaymentProviderBySlug(c.Request.Context(), "asaas")
	if err != nil {
		response.InternalServerError(c, "Asaas payment provider configuration missing")
		return
	}

	// 2. Read raw payload
	payloadBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Failed to read request body", err.Error())
		return
	}

	// 3. Verify signature token
	if dbProvider.WebhookSecretSecure.Valid && dbProvider.WebhookSecretSecure.String != "" {
		decryptedSecret, decErr := cryptoPkg.Decrypt(dbProvider.WebhookSecretSecure.String, getAESKey(h.encryptionKey))
		if decErr == nil && decryptedSecret != "" {
			receivedToken := c.GetHeader("asaas-access-token")
			if receivedToken != decryptedSecret {
				response.Unauthorized(c, "Invalid webhook token signature")
				return
			}
		}
	}

	// 4. Parse payload event name and payment ID
	var wrapper struct {
		Event   string `json:"event"`
		Payment struct {
			ID string `json:"id"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payloadBytes, &wrapper); err != nil {
		response.BadRequest(c, "Failed to parse webhook JSON structure", err.Error())
		return
	}

	if wrapper.Payment.ID == "" {
		response.BadRequest(c, "Invalid webhook payload: payment.id is missing", nil)
		return
	}

	// Use Payment.ID + Event as external_id for webhook_events to ensure idempotency per event change
	externalId := fmt.Sprintf("%s:%s", wrapper.Payment.ID, wrapper.Event)

	// Check if already registered
	_, err = h.queries.GetWebhookEventByExternalID(c.Request.Context(), database.GetWebhookEventByExternalIDParams{
		Provider:   "asaas",
		ExternalID: externalId,
	})
	if err == nil {
		// Idempotent bypass
		response.Success(c, http.StatusOK, "Webhook event already processed (idempotent)", nil)
		return
	}

	// 5. Store pending webhook event
	dbEvent, err := h.queries.CreateWebhookEvent(c.Request.Context(), database.CreateWebhookEventParams{
		Provider:   "asaas",
		EventType:  wrapper.Event,
		ExternalID: externalId,
		Payload:    payloadBytes,
		Status:     "PENDING",
	})
	if err != nil {
		response.InternalServerError(c, "Failed to record webhook event: "+err.Error())
		return
	}

	// 6. Queue the processing job on Redis queue (Opção B - assíncrona)
	taskPayload, _ := json.Marshal(gin.H{
		"id": uuidToString(dbEvent.ID),
	})
	_, err = queue.EnqueueTask("process_payment_webhook", taskPayload)
	if err != nil {
		// Fallback: log warning, but don't fail Asaas return so they don't loop webhook retries
		logger.Log.Error("Failed to queue webhook event processing task", zap.Error(err))
	}

	// Log audit event
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		Action:     "PAYMENT_WEBHOOK_RECEIVED",
		EntityType: "webhook_events",
		EntityID:   dbEvent.ID,
		Metadata:   payloadBytes,
		Ip:         pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:  pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Webhook event registered successfully", nil)
}

func (h *Handler) ListInvoices(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	var limit int32 = 10
	var offset int32 = 0

	invoices, err := h.queries.ListInvoicesByOrganization(c.Request.Context(), database.ListInvoicesByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve organization invoices")
		return
	}

	response.Success(c, http.StatusOK, "Invoices list", invoices)
}

func (h *Handler) ListPaymentProvidersAdmin(c *gin.Context) {
	providers, err := h.queries.ListPaymentProviders(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to list payment providers: "+err.Error())
		return
	}

	formatted := make([]map[string]interface{}, len(providers))
	for i, p := range providers {
		key := getAESKey(h.encryptionKey)
		var decApiKey string
		var decWebSecret string

		if p.ApiKeySecure.Valid && p.ApiKeySecure.String != "" {
			dec, err := cryptoPkg.Decrypt(p.ApiKeySecure.String, key)
			if err == nil {
				decApiKey = dec
			}
		}
		if p.WebhookSecretSecure.Valid && p.WebhookSecretSecure.String != "" {
			dec, err := cryptoPkg.Decrypt(p.WebhookSecretSecure.String, key)
			if err == nil {
				decWebSecret = dec
			}
		}

		formatted[i] = map[string]interface{}{
			"id":            uuidToString(p.ID),
			"name":          p.Name,
			"slug":          p.Slug,
			"apiKey":        maskAPIKey(decApiKey),
			"webhookSecret": maskAPIKey(decWebSecret),
			"isActive":      p.IsActive,
			"mode":          p.Mode,
			"createdAt":     p.CreatedAt,
		}
	}

	response.Success(c, http.StatusOK, "Payment providers list", formatted)
}

type UpdatePaymentProviderRequest struct {
	APIKey        string `json:"apiKey"`
	WebhookSecret string `json:"webhookSecret"`
	IsActive      *bool  `json:"isActive"`
	Mode          string `json:"mode"`
}

func (h *Handler) UpdatePaymentProviderAdmin(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid provider ID format", nil)
		return
	}

	var req UpdatePaymentProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON data", err.Error())
		return
	}

	// Fetch existing
	var p database.PaymentProvider
	err := h.db.QueryRow(c.Request.Context(), 
		"SELECT id, name, slug, api_key_secure, webhook_secret_secure, is_active, mode FROM payment_providers WHERE id = $1", 
		id,
	).Scan(&p.ID, &p.Name, &p.Slug, &p.ApiKeySecure, &p.WebhookSecretSecure, &p.IsActive, &p.Mode)
	if err != nil {
		response.NotFound(c, "Payment provider not found")
		return
	}

	key := getAESKey(h.encryptionKey)
	var encApiKey pgtype.Text = p.ApiKeySecure
	var encWebSecret pgtype.Text = p.WebhookSecretSecure

	if req.APIKey != "" {
		if !strings.HasPrefix(req.APIKey, "****") {
			enc, err := cryptoPkg.Encrypt(req.APIKey, key)
			if err != nil {
				response.InternalServerError(c, "Failed to encrypt API key")
				return
			}
			encApiKey = pgtype.Text{String: enc, Valid: true}
		}
	}

	if req.WebhookSecret != "" {
		if !strings.HasPrefix(req.WebhookSecret, "****") {
			enc, err := cryptoPkg.Encrypt(req.WebhookSecret, key)
			if err != nil {
				response.InternalServerError(c, "Failed to encrypt Webhook secret")
				return
			}
			encWebSecret = pgtype.Text{String: enc, Valid: true}
		}
	}

	isActiveVal := p.IsActive
	if req.IsActive != nil {
		isActiveVal = *req.IsActive
	}

	modeVal := p.Mode
	if req.Mode != "" {
		modeVal = req.Mode
	}

	_, err = h.queries.UpdatePaymentProvider(c.Request.Context(), database.UpdatePaymentProviderParams{
		ID:                  id,
		ApiKeySecure:        encApiKey,
		WebhookSecretSecure: encWebSecret,
		IsActive:            isActiveVal,
		Mode:                modeVal,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update payment provider details")
		return
	}

	// Audit log
	var userID pgtype.UUID
	userIDStr, exists := c.Get("user_id")
	if exists {
		_ = userID.Scan(userIDStr)
	}
	meta, _ := json.Marshal(map[string]string{"slug": p.Slug})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: userID,
		Action:      "PAYMENT_PROVIDER_UPDATED",
		EntityType:  "payment_providers",
		EntityID:    id,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Payment provider updated successfully", nil)
}

func (h *Handler) ListInvoicesAdmin(c *gin.Context) {
	invoices, err := h.queries.ListInvoicesDetailed(c.Request.Context(), database.ListInvoicesDetailedParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to query invoices: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Invoices list", invoices)
}

func (h *Handler) ListTransactionsAdmin(c *gin.Context) {
	txs, err := h.queries.ListPaymentTransactions(c.Request.Context(), database.ListPaymentTransactionsParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to query transactions: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Transactions list", txs)
}

func (h *Handler) ListWebhookEventsAdmin(c *gin.Context) {
	events, err := h.queries.ListWebhookEvents(c.Request.Context(), database.ListWebhookEventsParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to query webhook events: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Webhook events list", events)
}

// Helpers
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
