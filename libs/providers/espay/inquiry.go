package espay

import (
	"encoding/json"
	"errors"
	"strings"

	"jaxpay-go-backend/libs/providers"
)

const endpointVAInquiry = "va_inquiry"

// inquiryRequest = body Inquiry (service code 24) dari Espay.
type inquiryRequest struct {
	PartnerServiceID string `json:"partnerServiceId"`
	CustomerNo       string `json:"customerNo"`
	VirtualAccountNo string `json:"virtualAccountNo"`
	TrxDateInit      string `json:"trxDateInit"`
	InquiryRequestID string `json:"inquiryRequestId"`
}

func isInquiryPath(p string) bool { return strings.HasSuffix(p, "/inquiry") }

// parseInquiry: virtualAccountNo = id transaksi yang kita kirim ke Espay (provider_ref).
func (c *Client) parseInquiry(ev *providers.CallbackEvent, body []byte) error {
	ev.Type, ev.Endpoint = providers.CallbackInquiry, endpointVAInquiry
	var in inquiryRequest
	if json.Unmarshal(body, &in) != nil || in.VirtualAccountNo == "" || in.InquiryRequestID == "" {
		return providers.ErrInvalidRequest
	}
	ev.ProviderRef = in.VirtualAccountNo
	return nil
}

func trunc(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// inquiryResponse membentuk balasan SNAP service code 24.
func inquiryResponse(ev *providers.CallbackEvent, err error) (int, any) {
	var in inquiryRequest
	_ = json.Unmarshal(ev.Raw, &in)

	code, rc, msg := 200, "2002400", "Successful"
	switch {
	case err == nil && ev.Bill != nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, rc, msg = 401, "4012400", "Unauthorized. [Signature]"
	case errors.Is(err, providers.ErrInvalidRequest):
		code, rc, msg = 400, "4002401", "Invalid Field Format"
	case errors.Is(err, providers.ErrPaymentNotFound):
		code, rc, msg = 404, "4042412", "Bill not found"
	case errors.Is(err, providers.ErrAlreadyPaid):
		code, rc, msg = 404, "4042414", "Paid Bill"
	case errors.Is(err, providers.ErrPaymentExpired):
		code, rc, msg = 404, "4042419", "Invalid Bill/Virtual Account"
	default:
		code, rc, msg = 500, "5002400", "General Error"
	}

	va := map[string]any{
		"partnerServiceId": in.PartnerServiceID,
		"customerNo":       in.CustomerNo,
		"virtualAccountNo": in.VirtualAccountNo,
		"inquiryRequestId": in.InquiryRequestID,
	}
	if b := ev.Bill; rc == "2002400" {
		desc := trunc(b.Description, 18)
		if desc == "" {
			desc = "Pembayaran"
		}
		info := map[string]any{"transactionDate": b.CreatedAt.In(wib).Format(tsLayout)}
		if b.ExpiredAt != nil {
			info["expiredDatetime"] = b.ExpiredAt.In(wib).Format(tsLayout)
		}
		va["virtualAccountName"] = b.CustomerName
		va["virtualAccountEmail"] = b.CustomerEmail
		va["totalAmount"] = formatAmount(b.Amount)
		va["billDetails"] = []any{map[string]any{
			"billDescription": map[string]string{"english": desc, "indonesia": desc},
		}}
		va["additionalInfo"] = info
	}
	return code, map[string]any{"responseCode": rc, "responseMessage": msg, "virtualAccountData": va}
}
