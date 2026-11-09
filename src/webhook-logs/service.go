package webhooklogs

import (
	"errors"
	"time"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WebhookLogService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *WebhookLogService {
	return &WebhookLogService{DB: db}
}

func (s *WebhookLogService) getAll(d DefaultFindDTO) (*helpers.PaginatedResult, error) {
	filters := map[string]any{
		"webhook_logs.payment_id.eq": d.PaymentID,
		"webhook_logs.client_id.eq":  d.ClientID,
		"webhook_logs.event.eq":      d.Event,
		"webhook_logs.status.eq":     d.Status,
	}
	var v []models.WebhookLog
	return helpers.BuildPaginatedQuery(s.DB.Model(&models.WebhookLog{}), d.Page, d.Limit, d.Sort,
		"webhook_logs.created_at DESC", filters, &v)
}

func (s *WebhookLogService) getByID(id uuid.UUID) (*models.WebhookLog, error) {
	var v models.WebhookLog
	e := s.DB.First(&v, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &v, nil
}

func (s *WebhookLogService) retry(id uuid.UUID) (*models.WebhookLog, error) {
	v, e := s.getByID(id)
	if e != nil || v == nil {
		return v, e
	}
	now := time.Now()
	e = s.DB.Model(v).Updates(map[string]any{"status": models.WebhookPending, "attempt": 0, "next_retry_at": now}).Error
	if e != nil {
		return nil, e
	}
	return s.getByID(id)
}
