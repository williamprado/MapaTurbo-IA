package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/logger"
	"go.uber.org/zap"
)

type Worker struct {
	db      *pgxpool.Pool
	queries *database.Queries
}

func NewWorker(db *pgxpool.Pool) *Worker {
	return &Worker{
		db:      db,
		queries: database.New(db),
	}
}

func (w *Worker) ProcessPaymentWebhookTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.Error("Failed to parse payment webhook task payload", zap.Error(err))
		return nil // Return nil so it doesn't loop retrying bad JSON payloads
	}

	eventIDStr, exists := payload["id"]
	if !exists || eventIDStr == "" {
		logger.Log.Error("Missing id in task payload")
		return nil
	}

	var eventID pgtype.UUID
	if err := eventID.Scan(eventIDStr); err != nil {
		logger.Log.Error("Invalid event ID UUID format", zap.Error(err))
		return nil
	}

	// 1. Fetch webhook event by database ID
	// Our task payload carries the database ID!
	// So we need a query to get the webhook event by database ID!
	// Let's check: did we define GetWebhookEventByID or similar? No, billing.sql had GetWebhookEventByExternalID.
	// But wait! We can just query by database ID directly using standard sql or add it!
	// Let's look up using the database connection pool:
	// db.QueryRow(ctx, "SELECT id, provider, event_type, external_id, payload, status FROM webhook_events WHERE id = $1", eventID).Scan(...)
	// That is extremely simple and requires no SQLC rebuild!
	var dbEvent struct {
		ID         pgtype.UUID
		Provider   string
		EventType  string
		ExternalID string
		Payload    []byte
		Status     string
	}
	err := w.db.QueryRow(ctx, 
		"SELECT id, provider, event_type, external_id, payload, status FROM webhook_events WHERE id = $1", 
		eventID,
	).Scan(&dbEvent.ID, &dbEvent.Provider, &dbEvent.EventType, &dbEvent.ExternalID, &dbEvent.Payload, &dbEvent.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Log.Error("Webhook event not found in database", zap.String("id", eventIDStr))
			return nil
		}
		return err
	}

	if dbEvent.Status == "PROCESSED" {
		logger.Log.Info("Webhook event already processed (idempotent skip)", zap.String("id", eventIDStr))
		return nil
	}

	// 2. Parse event payload
	var webhookPayload struct {
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
	if err := json.Unmarshal(dbEvent.Payload, &webhookPayload); err != nil {
		w.markEventFailed(ctx, eventID, "failed to parse event payload: "+err.Error())
		return nil
	}

	paymentID := webhookPayload.Payment.ID
	if paymentID == "" {
		w.markEventFailed(ctx, eventID, "invalid payload: payment.id is missing")
		return nil
	}

	// 3. Process inside a SQL transaction
	tx, err := w.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txQueries := w.queries.WithTx(tx)
	_ = txQueries

	// Local mapping
	var invoice database.Invoice
	invoiceErr := tx.QueryRow(ctx,
		"SELECT id, organization_id, subscription_id, amount, status FROM invoices WHERE external_invoice_id = $1 LIMIT 1",
		paymentID,
	).Scan(&invoice.ID, &invoice.OrganizationID, &invoice.SubscriptionID, &invoice.Amount, &invoice.Status)

	isConfirmedEvent := webhookPayload.Event == "PAYMENT_CONFIRMED" || webhookPayload.Event == "PAYMENT_RECEIVED"
	isOverdueEvent := webhookPayload.Event == "PAYMENT_OVERDUE"

	if invoiceErr == nil {
		if isConfirmedEvent {
			if invoice.Status != "PAID" {
				// Mark Invoice Paid
				_, err = tx.Exec(ctx, "UPDATE invoices SET status = 'PAID', paid_at = NOW(), updated_at = NOW() WHERE id = $1", invoice.ID)
				if err != nil {
					return fmt.Errorf("failed to update invoice status: %w", err)
				}

				// Mark Subscription Active
				if invoice.SubscriptionID.Valid {
					expiresAt := time.Now().AddDate(0, 1, 0)
					_, err = tx.Exec(ctx, "UPDATE subscriptions SET status = 'ACTIVE', current_period_start = NOW(), current_period_end = $2, updated_at = NOW() WHERE id = $1", invoice.SubscriptionID, expiresAt)
					if err != nil {
						return fmt.Errorf("failed to update subscription status: %w", err)
					}
				}

				// Mark Payment Transaction Paid
				_, err = tx.Exec(ctx, "UPDATE payment_transactions SET status = 'PAID', paid_at = NOW() WHERE invoice_id = $1", invoice.ID)
				if err != nil {
					return fmt.Errorf("failed to update transaction status: %w", err)
				}

				// Grant Credits
				var plan database.Plan
				err = tx.QueryRow(ctx,
					"SELECT p.credits_monthly, p.name FROM plans p JOIN subscriptions s ON s.plan_id = p.id WHERE s.id = $1",
					invoice.SubscriptionID,
				).Scan(&plan.CreditsMonthly, &plan.Name)
				if err == nil {
					// Initialize Balance
					_, _ = tx.Exec(ctx, "INSERT INTO ai_credit_balances (organization_id, balance, updated_at) VALUES ($1, 0, NOW()) ON CONFLICT (organization_id) DO NOTHING", invoice.OrganizationID)
					
					// Add Balance
					_, err = tx.Exec(ctx, "UPDATE ai_credit_balances SET balance = balance + $2, updated_at = NOW() WHERE organization_id = $1", invoice.OrganizationID, plan.CreditsMonthly)
					if err != nil {
						return fmt.Errorf("failed to update credit balance: %w", err)
					}

					// Create Credit Transaction
					metaBytes, _ := json.Marshal(map[string]interface{}{
						"invoice_id":             uuidToString(invoice.ID),
						"external_transaction_id": paymentID,
					})
					_, err = tx.Exec(ctx,
						"INSERT INTO ai_credit_transactions (organization_id, amount, type, description, metadata) VALUES ($1, $2, 'ADD', $3, $4)",
						invoice.OrganizationID, plan.CreditsMonthly, "Créditos mensais do plano (Asaas Webhook)", metaBytes,
					)
					if err != nil {
						return fmt.Errorf("failed to log credit transaction: %w", err)
					}
				}

				// Audit Logs
				var actorUserID pgtype.UUID // webhook is a system operation, actor remains null
				metaAudit, _ := json.Marshal(map[string]string{
					"invoice_id":          uuidToString(invoice.ID),
					"external_payment_id": paymentID,
				})
				
				// Audit - Payment Confirmed
				_, _ = tx.Exec(ctx,
					"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'PAYMENT_CONFIRMED', 'invoices', $3, $4)",
					actorUserID, invoice.OrganizationID, invoice.ID, metaAudit,
				)

				// Audit - Subscription Activated
				if invoice.SubscriptionID.Valid {
					_, _ = tx.Exec(ctx,
						"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'SUBSCRIPTION_ACTIVATED', 'subscriptions', $3, $4)",
						actorUserID, invoice.OrganizationID, invoice.SubscriptionID, metaAudit,
					)
				}

				// Audit - AI Credits Granted
				_, _ = tx.Exec(ctx,
					"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'AI_CREDITS_GRANTED', 'ai_credit_balances', $3, $4)",
					actorUserID, invoice.OrganizationID, invoice.OrganizationID, metaAudit,
				)
			}
		} else if isOverdueEvent {
			// Mark Invoice Failed
			_, _ = tx.Exec(ctx, "UPDATE invoices SET status = 'FAILED', updated_at = NOW() WHERE id = $1", invoice.ID)
			
			// Mark Subscription Past Due
			if invoice.SubscriptionID.Valid {
				_, _ = tx.Exec(ctx, "UPDATE subscriptions SET status = 'PAST_DUE', updated_at = NOW() WHERE id = $1", invoice.SubscriptionID)
			}

			// Mark Transaction Failed
			_, _ = tx.Exec(ctx, "UPDATE payment_transactions SET status = 'FAILED', failed_at = NOW() WHERE invoice_id = $1", invoice.ID)

			// Audit Log - Payment Overdue
			var actorUserID pgtype.UUID
			metaAudit, _ := json.Marshal(map[string]string{
				"invoice_id":          uuidToString(invoice.ID),
				"external_payment_id": paymentID,
			})
			_, _ = tx.Exec(ctx,
				"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'PAYMENT_OVERDUE', 'invoices', $3, $4)",
				actorUserID, invoice.OrganizationID, invoice.ID, metaAudit,
			)
		}
	} else {
		// Log warning if payment invoice reference not found in local db, but proceed to mark processed
		logger.Log.Warn("Webhook received for unknown invoice reference", zap.String("payment_id", paymentID))
	}

	// Update webhook event to processed
	_, err = tx.Exec(ctx,
		"UPDATE webhook_events SET status = 'PROCESSED', processed_at = NOW() WHERE id = $1",
		eventID,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Log.Info("Webhook event processed successfully", zap.String("id", eventIDStr), zap.String("action", webhookPayload.Event))
	return nil
}

func (w *Worker) markEventFailed(ctx context.Context, id pgtype.UUID, errMsg string) {
	_, err := w.db.Exec(ctx,
		"UPDATE webhook_events SET status = 'FAILED', error = $2, processed_at = NOW() WHERE id = $1",
		id, errMsg,
	)
	if err != nil {
		logger.Log.Error("Failed to update webhook event status to FAILED", zap.Error(err))
	}
}
