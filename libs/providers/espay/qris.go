package espay

import (
	"context"
	"encoding/json"
	"net/http"

	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

const (
	pathQRGenerate = "/api/v1.0/qr/qr-mpm-generate"
	pathQRQuery    = "/api/v1.0/qr/qr-mpm-query"
	pathQRCancel   = "/api/v1.0/qr/qr-mpm-cancel"
)

func (c *Client) CreateQRIS(ctx context.Context, req providers.CreateQRISRequest) (*providers.PaymentResult, error) {
	ref := req.ProviderRef
	if ref == "" {
		ref = providers.NewProviderRef()
	}
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
		ReferenceNo    string          `json:"referenceNo"`
		QRContent      string          `json:"qrContent"`
		QRURL          string          `json:"qrUrl"`
		AdditionalInfo json.RawMessage `json:"additionalInfo"`
	}
	if e := c.do(ctx, http.MethodPost, pathQRGenerate, in, &out); e != nil {
		return nil, e
	}
	if !out.ok() || out.QRContent == "" {
		return nil, out.err("qr-mpm-generate")
	}
	exp := req.ExpiredAt
	res := &providers.PaymentResult{ProviderRef: ref, QRString: &out.QRContent, ExpiredAt: &exp, ProviderData: out.AdditionalInfo}
	if out.QRURL != "" {
		res.QRURL = &out.QRURL
	}
	return res, nil
}

func (c *Client) CheckStatus(ctx context.Context, method, providerRef string) (*providers.StatusResult, error) {
	if method == "va" || method == "direct_debit" {
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
	switch method {
	case "va":
		return c.deleteVA(ctx, providerRef)
	case "direct_debit":
		return providers.ErrNotSupported
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

func mapStatus(s string) models.PaymentStatus {
	switch s {
	case "00":
		return models.StatusPaid
	case "05":
		return models.StatusCancelled
	case "06", "07":
		return models.StatusFailed
	}
	return models.StatusPending
}
