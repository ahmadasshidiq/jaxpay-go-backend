package helpers

import (
	"encoding/json"
	"errors"
	"time"

	"jaxpay-go-backend/libs/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrStateConflict = errors.New("payment is no longer pending")

func EventFor(s models.PaymentStatus) string {
	switch s {
	case models.StatusPaid:
		return "payment.paid"
	case models.StatusExpired:
		return "payment.expired"
	case models.StatusFailed:
		return "payment.failed"
	case models.StatusCancelled:
		return "payment.cancelled"
	}
	return "payment.updated"
}

// TransitionPayment memindahkan payment PENDING → status final dan insert WebhookLog
// (outbox) dalam satu transaksi. Return ErrStateConflict jika payment sudah tidak PENDING.
func TransitionPayment(db *gorm.DB, p *models.Payment, to models.PaymentStatus, paidAt *time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": to, "updated_at": time.Now()}
		if paidAt != nil {
			updates["paid_at"] = *paidAt
		}
		r := tx.Model(&models.Payment{}).
			Where("id = ? AND status = ?", p.ID, models.StatusPending).
			Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrStateConflict
		}
		p.Status = to
		if paidAt != nil {
			p.PaidAt = paidAt
		}
		wl, e := NewWebhookLog(p, EventFor(to))
		if e != nil {
			return e
		}
		return tx.Create(wl).Error
	})
}

func NewWebhookLog(p *models.Payment, event string) (*models.WebhookLog, error) {
	meta := json.RawMessage(p.Metadata)
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}
	body, e := json.Marshal(map[string]any{
		"event": event,
		"data": map[string]any{
			"id":           p.ID,
			"reference_id": p.ReferenceID,
			"method":       p.Method,
			"amount":       p.Amount,
			"status":       p.Status,
			"paid_at":      p.PaidAt,
			"metadata":     meta,
		},
	})
	if e != nil {
		return nil, e
	}
	now := time.Now()
	return &models.WebhookLog{
		PaymentID:   p.ID,
		ClientID:    p.ClientID,
		Event:       event,
		Payload:     datatypes.JSON(body),
		Status:      models.WebhookPending,
		NextRetryAt: &now,
	}, nil
}
