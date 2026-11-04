package espay

import (
	"context"

	"jaxpay-go-backend/libs/providers"
)

// CreateVA belum diimplementasikan untuk Espay (fase 1 hanya QRIS).
func (c *Client) CreateVA(ctx context.Context, req providers.CreateVARequest) (*providers.PaymentResult, error) {
	return nil, providers.ErrNotSupported
}
