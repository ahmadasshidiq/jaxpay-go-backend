package espay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

const endpointVAPayment = "va_payment"

type vaPaymentRequest struct {
	PartnerServiceID string  `json:"partnerServiceId"`
	CustomerNo       string  `json:"customerNo"`
	VirtualAccountNo string  `json:"virtualAccountNo"`
	PaymentRequestID string  `json:"paymentRequestId"`
	TotalAmount      *amount `json:"totalAmount"`
	TrxDateTime      string  `json:"trxDateTime"`
	AdditionalInfo   struct {
		TransactionStatus string `json:"transactionStatus"`
	} `json:"additionalInfo"`
}

func isVAPayment(body []byte) bool {
	var in struct {
		VirtualAccountNo string `json:"virtualAccountNo"`
	}
	return json.Unmarshal(body, &in) == nil && in.VirtualAccountNo != ""
}

func (c *Client) parseVAPayment(ev *providers.CallbackEvent, body []byte) error {
	var in vaPaymentRequest
	if json.Unmarshal(body, &in) != nil || in.PaymentRequestID == "" || in.TotalAmount == nil {
		return providers.ErrInvalidRequest
	}
	st, ok := mapVAPaymentStatus(in.AdditionalInfo.TransactionStatus)
	if !ok {
		return providers.ErrInvalidRequest
	}
	amt, e := parseAmount(in.TotalAmount.Value)
	if e != nil {
		return providers.ErrInvalidRequest
	}
	ev.ProviderRef, ev.Amount, ev.Status = in.VirtualAccountNo, amt, st
	if st == models.StatusPaid {
		ev.PaidAt = parseTime(in.TrxDateTime)
	}
	return nil
}

func mapVAPaymentStatus(s string) (models.PaymentStatus, bool) {
	switch s {
	case "S":
		return models.StatusPaid, true
	case "F":
		return models.StatusFailed, true
	case "SP", "IP", "HP":
		return models.StatusPending, true
	}
	return "", false
}

func vaPaymentResponse(ev *providers.CallbackEvent, err error) (int, any) {
	var in vaPaymentRequest
	_ = json.Unmarshal(ev.Raw, &in)

	code, rc, msg := 200, "2002500", "Successful"
	switch {
	case err == nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, rc, msg = 401, "4012500", "Unauthorized. [Signature]"
	case errors.Is(err, providers.ErrInvalidRequest):
		code, rc, msg = 400, "4002501", "Invalid Field Format"
	case errors.Is(err, providers.ErrPaymentNotFound):
		code, rc, msg = 404, "4042512", "Bill not found"
	case errors.Is(err, providers.ErrAmountMismatch):
		code, rc, msg = 404, "4042513", "Invalid Amount"
	case errors.Is(err, providers.ErrPaymentExpired):
		code, rc, msg = 404, "4042519", "Invalid Bill/Virtual Account"
	default:
		code, rc, msg = 500, "5002500", "General Error"
	}

	va := map[string]any{
		"partnerServiceId": in.PartnerServiceID,
		"customerNo":       in.CustomerNo,
		"virtualAccountNo": in.VirtualAccountNo,
		"paymentRequestId": in.PaymentRequestID,
	}
	if in.TotalAmount != nil {
		va["totalAmount"] = in.TotalAmount
	}
	if b := ev.Bill; b != nil {
		desc := trunc(b.Description, 18)
		if desc == "" {
			desc = "Pembayaran"
		}
		va["virtualAccountName"] = b.CustomerName
		va["billDetails"] = []any{map[string]any{
			"billDescription": map[string]string{"english": desc, "indonesia": desc},
		}}
	}
	h := sha256.Sum256([]byte(in.PaymentRequestID))
	return code, map[string]any{
		"responseCode":       rc,
		"responseMessage":    msg,
		"virtualAccountData": va,
		"additionalInfo": map[string]string{
			"reconcileId":       hex.EncodeToString(h[:16]),
			"reconcileDatetime": time.Now().In(wib).Format(tsLayout),
		},
	}
}
