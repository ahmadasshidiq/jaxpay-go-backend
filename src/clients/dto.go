package clients

import "github.com/google/uuid"

type DefaultFindDTO struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort"`
	Search   string `form:"search"`
	IsActive *bool  `form:"is_active"`
}

type CreateDTO struct {
	Code        string `json:"code" binding:"required,max=50,alphanum,lowercase"`
	Name        string `json:"name" binding:"required,max=255"`
	CallbackURL string `json:"callback_url" binding:"required,url,max=500"`
	VAPrefix    string `json:"va_prefix" binding:"omitempty,numeric,max=10"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateDTO struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	CallbackURL *string `json:"callback_url" binding:"omitempty,url,max=500"`
	VAPrefix    *string `json:"va_prefix" binding:"omitempty,numeric,max=10"`
	IsActive    *bool   `json:"is_active"`
}

// CredentialsDTO hanya dikembalikan sekali saat create / rotate-secret.
type CredentialsDTO struct {
	ID            uuid.UUID `json:"id"`
	APIKey        string    `json:"api_key"`
	APISecret     string    `json:"api_secret"`
	WebhookSecret string    `json:"webhook_secret"`
}
