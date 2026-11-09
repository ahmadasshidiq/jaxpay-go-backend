package espay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

const (
	pathQRGenerate = "/api/v1.0/qr/qr-mpm-generate"
	pathQRQuery    = "/api/v1.0/qr/qr-mpm-query"
	pathQRCancel   = "/api/v1.0/qr/qr-mpm-cancel"
)

func (c *Client) CreateQRIS(ctx context.Context, req providers.CreateQRISRequest) (*providers.PaymentResult, error) {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	ref := "JXP" + strings.ToUpper(hex.EncodeToString(b))
	// struct (bukan map) agar urutan field sama dengan dokumentasi Espay
	in := struct {
		PartnerReferenceNo string            `json:"partnerReferenceNo"`
		MerchantID         string            `json:"merchantId"`
		Amount             amount            `json:"amount"`
		AdditionalInfo     map[string]string `json:"additionalInfo"`
		ValidityPeriod     string            `json:"validityPeriod,omitempty"`
	}{ref, c.cfg.MerchantID, formatAmount(req.Amount), map[string]string{"productCode": "QRIS"}, ""}
	if !req.ExpiredAt.IsZero() {
		in.ValidityPeriod = req.ExpiredAt.In(wib).Format(tsLayout)
	}
	var out struct {
		snapResponse
		ReferenceNo string `json:"referenceNo"`
		QRContent   string `json:"qrContent"`
	}
	if e := c.do(ctx, http.MethodPost, pathQRGenerate, in, &out); e != nil {
		return nil, e
	}
	if !out.ok() || out.QRContent == "" {
		return nil, out.err("qr-mpm-generate")
	}
	exp := req.ExpiredAt
	return &providers.PaymentResult{ProviderRef: ref, QRString: &out.QRContent, ExpiredAt: &exp}, nil
}

func (c *Client) CheckStatus(ctx context.Context, method, providerRef string) (*providers.StatusResult, error) {
	if method == "va" {
		return c.checkVAStatus(ctx, providerRef)
	}
	in := map[string]any{
		"originalPartnerReferenceNo": providerRef,
		"serviceCode":                "47",
		"merchantId":                 c.cfg.MerchantID,
	}
	var out struct {
		snapResponse
		LatestTransactionStatus string  `json:"latestTransactionStatus"`
		PaidTime                string  `json:"paidTime"`
		Amount                  *amount `json:"amount"`
	}
	if e := c.do(ctx, http.MethodPost, pathQRQuery, in, &out); e != nil {
		return nil, e
	}
	if !out.ok() {
		return nil, out.err("qr-mpm-query")
	}
	r := &providers.StatusResult{ProviderRef: providerRef, Status: mapStatus(out.LatestTransactionStatus)}
	if out.Amount != nil {
		r.Amount, _ = parseAmount(out.Amount.Value)
	}
	if r.Status == models.StatusPaid {
		r.PaidAt = parseTime(out.PaidTime)
	}
	return r, nil
}

func (c *Client) Cancel(ctx context.Context, method, providerRef string) error {
	if method == "va" {
		return c.deleteVA(ctx, providerRef)
	}
	in := map[string]any{
		"originalPartnerReferenceNo": providerRef,
		"merchantId":                 c.cfg.MerchantID,
		"reason":                     "cancelled by merchant",
	}
	var out snapResponse
	if e := c.do(ctx, http.MethodPost, pathQRCancel, in, &out); e != nil {
		return e
	}
	if !out.ok() {
		return out.err("qr-mpm-cancel")
	}
	return nil
}

// mapStatus memetakan latestTransactionStatus SNAP ke status internal.
func mapStatus(s string) models.PaymentStatus {
	switch s {
	case "00":
		return models.StatusPaid
	case "05":
		return models.StatusCancelled
	case "06", "07":
		return models.StatusFailed
	}
	return models.StatusPending // 01 initiated, 02 paying, 03 pending, dst.
}
