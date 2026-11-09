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
	Method      string `form:"method" binding:"omitempty,oneof=qris va direct_debit"`
	ReferenceID string `form:"reference_id"`
}

type CreatePaymentDTO struct {
	ReferenceID    string         `json:"reference_id" binding:"required,max=100" example:"INV-20261106-0001"`
	Method         string         `json:"method" binding:"required,oneof=qris va direct_debit" example:"va"`
	Amount         int64          `json:"amount" binding:"required,min=1" example:"150000"`
	CustomerName   string         `json:"customer_name" binding:"required,max=255" example:"Budi Santoso"`
	CustomerEmail  string         `json:"customer_email" binding:"omitempty,email" example:"budi.santoso@example.com"`
	CustomerPhone  string         `json:"customer_phone" binding:"omitempty,numeric,max=17" example:"081234567890"`
	BankCode       string         `json:"bank_code" binding:"required_if=Method va,omitempty,numeric,len=3" example:"014"`                                                  // wajib untuk method va
	ProductCode    string         `json:"product_code" binding:"required_if=Method direct_debit,omitempty,max=50" example:"GOPAYLINK"`                                      // wajib untuk method direct_debit
	ReturnURL      string         `json:"return_url" binding:"required_if=Method direct_debit,omitempty,url,max=255" example:"https://merchant.example.com/payment/return"` // wajib untuk method direct_debit
	Description    string         `json:"description" binding:"omitempty,max=500" example:"Pembayaran invoice INV-20261106-0001"`
	ExpiredMinutes int            `json:"expired_minutes" binding:"omitempty,min=5,max=10080" example:"60"`
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
	BankCode      *string              `json:"bank_code,omitempty"`
	ProductCode   *string              `json:"product_code,omitempty"`
	RedirectURL   *string              `json:"redirect_url,omitempty"`
	QRString      *string              `json:"qr_string,omitempty"`
	QRURL         *string              `json:"qr_url,omitempty"`
	CustomerName  string               `json:"customer_name"`
	CustomerEmail string               `json:"customer_email,omitempty"`
	Description   string               `json:"description,omitempty"`
	Metadata      datatypes.JSON       `json:"metadata,omitempty" swaggertype:"object"`
	ProviderData  datatypes.JSON       `json:"provider_data,omitempty" swaggertype:"object"`
	ExpiredAt     *time.Time           `json:"expired_at"`
	PaidAt        *time.Time           `json:"paid_at"`
	CreatedAt     time.Time            `json:"created_at"`
}

type VABankResponse struct {
	BankCode    string `json:"bank_code"`
	Name        string `json:"name"`
	ProductCode string `json:"product_code"`
}

func toResponse(p *models.Payment) PaymentResponse {
	return PaymentResponse{
		ID: p.ID, ReferenceID: p.ReferenceID, Method: p.Method, Provider: p.Provider,
		Amount: p.Amount, Currency: p.Currency, Status: p.Status, VANumber: p.VANumber, BankCode: p.BankCode, ProductCode: p.ProductCode, RedirectURL: p.RedirectURL, QRString: p.QRString, QRURL: p.QRURL,
		CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, Description: p.Description,
		Metadata: p.Metadata, ProviderData: p.ProviderData, ExpiredAt: p.ExpiredAt, PaidAt: p.PaidAt, CreatedAt: p.CreatedAt,
	}
}
