package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Client struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code          string         `gorm:"uniqueIndex;size:50" json:"code"`
	Name          string         `gorm:"size:255" json:"name"`
	APIKey        string         `gorm:"uniqueIndex;size:100" json:"api_key"`
	APISecretHash string         `gorm:"size:255" json:"-"`
	WebhookSecret string         `gorm:"size:255" json:"-"`
	CallbackURL   string         `gorm:"size:500" json:"callback_url"`
	VAPrefix      string         `gorm:"size:20" json:"va_prefix"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProviderRoute struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClientID  uuid.UUID      `gorm:"type:uuid;uniqueIndex:idx_client_method" json:"client_id"`
	Method    string         `gorm:"size:50;uniqueIndex:idx_client_method" json:"method"`
	Provider  string         `gorm:"size:50" json:"provider"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProviderBank struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Provider    string         `gorm:"size:50;uniqueIndex:idx_provider_product" json:"provider"`
	ProductCode string         `gorm:"size:50;uniqueIndex:idx_provider_product" json:"product_code"`
	Name        string         `gorm:"size:100" json:"name"`
	BankCode    string         `gorm:"size:10;index" json:"bank_code"`
	IsVA        bool           `gorm:"default:false" json:"is_va"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "PENDING"
	StatusPaid      PaymentStatus = "PAID"
	StatusExpired   PaymentStatus = "EXPIRED"
	StatusFailed    PaymentStatus = "FAILED"
	StatusCancelled PaymentStatus = "CANCELLED"
)

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClientID      uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_client_ref" json:"client_id"`
	ReferenceID   string         `gorm:"size:100;uniqueIndex:idx_client_ref" json:"reference_id"`
	Method        string         `gorm:"size:50" json:"method"`
	Provider      string         `gorm:"size:50;index" json:"provider"`
	ProviderRef   string         `gorm:"size:150;index" json:"provider_ref"`
	Amount        int64          `json:"amount"`
	Currency      string         `gorm:"size:3;default:IDR" json:"currency"`
	Status        PaymentStatus  `gorm:"size:20;index" json:"status"`
	VANumber      *string        `gorm:"size:50;index" json:"va_number"`
	BankCode      *string        `gorm:"size:10" json:"bank_code"`
	ProductCode   *string        `gorm:"size:50" json:"product_code"`
	ReturnURL     *string        `gorm:"size:255" json:"return_url"`
	RedirectURL   *string        `gorm:"type:text" json:"redirect_url"`
	QRString      *string        `gorm:"type:text" json:"qr_string"`
	CustomerName  string         `gorm:"size:255" json:"customer_name"`
	CustomerEmail string         `gorm:"size:255" json:"customer_email"`
	CustomerPhone string         `gorm:"size:20" json:"customer_phone"`
	Description   string         `gorm:"size:500" json:"description"`
	Metadata      datatypes.JSON `json:"metadata" swaggertype:"object"`
	ExpiredAt     *time.Time     `json:"expired_at"`
	PaidAt        *time.Time     `json:"paid_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type CallbackLog struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Provider  string         `gorm:"size:50;index" json:"provider"`
	PaymentID *uuid.UUID     `gorm:"type:uuid;index" json:"payment_id"`
	Headers   datatypes.JSON `json:"headers" swaggertype:"object"`
	Payload   datatypes.JSON `json:"payload" swaggertype:"object"`
	IsValid   bool           `json:"is_valid"`
	Error     string         `gorm:"type:text" json:"error"`
	CreatedAt time.Time      `json:"created_at"`
}

const (
	WebhookPending = "PENDING"
	WebhookSuccess = "SUCCESS"
	WebhookFailed  = "FAILED"
)

type WebhookLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PaymentID    uuid.UUID      `gorm:"type:uuid;index" json:"payment_id"`
	ClientID     uuid.UUID      `gorm:"type:uuid;index" json:"client_id"`
	Event        string         `gorm:"size:50" json:"event"`
	Payload      datatypes.JSON `json:"payload" swaggertype:"object"`
	Attempt      int            `json:"attempt"`
	ResponseCode int            `json:"response_code"`
	ResponseBody string         `gorm:"type:text" json:"response_body"`
	Status       string         `gorm:"size:20;index" json:"status"`
	NextRetryAt  *time.Time     `gorm:"index" json:"next_retry_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func AllModels() []any {
	return []any{&Client{}, &ProviderRoute{}, &ProviderBank{}, &Payment{}, &CallbackLog{}, &WebhookLog{}}
}
