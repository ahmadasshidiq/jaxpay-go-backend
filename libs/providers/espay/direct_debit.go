package espay

import (
	"context"
	"net/http"

	"jaxpay-go-backend/libs/providers"
)

const pathDirectDebit = "/apimerchant/v1.0/debit/payment-host-to-host"

// CreateDirectDebit = Payment Host to Host (service code 54): menghasilkan URL Espay Checkout Page.
func (c *Client) CreateDirectDebit(ctx context.Context, req providers.CreateDirectDebitRequest) (*providers.PaymentResult, error) {
	total := formatAmount(req.Amount)
	in := map[string]any{
		"partnerReferenceNo": req.ProviderRef,
		"merchantId":         c.cfg.MerchantID,
		"amount":             total,
		"urlParam":           map[string]string{"url": req.ReturnURL, "type": "PAY_RETURN", "isDeeplink": "N"},
		"payOptionDetails": map[string]any{
			"payMethod":   req.BankCode,
			"payOption":   req.ProductCode,
			"transAmount": total,
			"feeAmount":   formatAmount(0),
		},
		"additionalInfo": map[string]string{
			"payType":     "REDIRECT",
			"productCode": req.ProductCode,
			"userName":    trunc(req.CustomerName, 64),
			"userEmail":   trunc(req.CustomerEmail, 64),
			"userPhone":   trunc(req.CustomerPhone, 16),
		},
	}
	if !req.ExpiredAt.IsZero() {
		in["validUpTo"] = req.ExpiredAt.In(wib).Format(tsLayout)
	}
	var out struct {
		snapResponse
		WebRedirectURL string `json:"webRedirectUrl"`
	}
	if e := c.doAt(ctx, c.merchantURL(), http.MethodPost, pathDirectDebit, in, &out); e != nil {
		return nil, e
	}
	if !out.ok() || out.WebRedirectURL == "" {
		return nil, out.err("payment-host-to-host")
	}
	res := &providers.PaymentResult{ProviderRef: req.ProviderRef, RedirectURL: &out.WebRedirectURL}
	if !req.ExpiredAt.IsZero() {
		res.ExpiredAt = &req.ExpiredAt
	}
	return res, nil
}
