package payments

import (
	"time"

	"jaxpay-go-backend/libs/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type DefaultFindDTO struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort        string `form:"sort"`
	Status      string `form:"status" binding:"omitempty,oneof=PENDING PAID EXPIRED FAILED CANCELLED"`
	Method      string `form:"method" binding:"omitempty,oneof=qris"`
	ReferenceID string `form:"reference_id"`
}

type CreatePaymentDTO struct {
	ReferenceID    string         `json:"reference_id" binding:"required,max=100"`
	Method         string         `json:"method" binding:"required,oneof=qris"`
	Amount         int64          `json:"amount" binding:"required,min=1"`
	CustomerName   string         `json:"customer_name" binding:"required,max=255"`
	CustomerEmail  string         `json:"customer_email" binding:"omitempty,email"`
	Description    string         `json:"description" binding:"omitempty,max=500"`
	ExpiredMinutes int            `json:"expired_minutes" binding:"omitempty,min=5,max=10080"`
	Metadata       map[string]any `json:"metadata" binding:"omitempty"`
}

type PaymentResponse struct {
	ID            uuid.UUID            `json:"id"`
	ReferenceID   string               `json:"reference_id"`
	Method        string               `json:"method"`
	Provider      string               `json:"provider"`
	Amount        int64                `json:"amount"`
	Currency      string               `json:"currency"`
	Status        models.PaymentStatus `json:"status"`
	VANumber      *string              `json:"va_number,omitempty"`
	QRString      *string              `json:"qr_string,omitempty"`
	CustomerName  string               `json:"customer_name"`
	CustomerEmail string               `json:"customer_email,omitempty"`
	Description   string               `json:"description,omitempty"`
	Metadata      datatypes.JSON       `json:"metadata,omitempty" swaggertype:"object"`
	ExpiredAt     *time.Time           `json:"expired_at"`
	PaidAt        *time.Time           `json:"paid_at"`
	CreatedAt     time.Time            `json:"created_at"`
}

func toResponse(p *models.Payment) PaymentResponse {
	return PaymentResponse{
		ID: p.ID, ReferenceID: p.ReferenceID, Method: p.Method, Provider: p.Provider,
		Amount: p.Amount, Currency: p.Currency, Status: p.Status, VANumber: p.VANumber, QRString: p.QRString,
		CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, Description: p.Description,
		Metadata: p.Metadata, ExpiredAt: p.ExpiredAt, PaidAt: p.PaidAt, CreatedAt: p.CreatedAt,
	}
}
