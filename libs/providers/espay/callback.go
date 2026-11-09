package espay

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

const (
	endpointQRISNotify = "qris_notify"
	maxSkew            = 5 * time.Minute
)

type qrisNotify struct {
	OriginalReferenceNo        string  `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string  `json:"originalPartnerReferenceNo"`
	LatestTransactionStatus    string  `json:"latestTransactionStatus"`
	Amount                     *amount `json:"amount"`
	PaidTime                   string  `json:"paidTime"`
}

func (c *Client) ParseCallback(r *http.Request, body []byte) (*providers.CallbackEvent, error) {
	ev := &providers.CallbackEvent{Provider: name, Raw: body}
	if isSettlementPath(r.URL.Path) {
		return ev, c.parseSettlement(ev, body)
	}
	inquiry := isInquiryPath(r.URL.Path)
	switch {
	case inquiry:
		ev.Type, ev.Endpoint = providers.CallbackInquiry, endpointVAInquiry
	case strings.HasSuffix(r.URL.Path, "/payment"):
		ev.Type, ev.Endpoint = providers.CallbackPayment, endpointQRISNotify
		if isVAPayment(body) {
			ev.Endpoint = endpointVAPayment
		}
	default:
		return ev, providers.ErrInvalidRequest
	}

	ts := r.Header.Get("X-TIMESTAMP")
	if !freshTimestamp(ts) || !c.verify(r.Method, r.URL.Path, body, ts, r.Header.Get("X-SIGNATURE")) {
		return ev, providers.ErrInvalidSignature
	}
	if inquiry {
		return ev, c.parseInquiry(ev, body)
	}

	if ev.Endpoint == endpointVAPayment {
		return ev, c.parseVAPayment(ev, body)
	}

	var in qrisNotify
	if json.Unmarshal(body, &in) != nil || in.OriginalPartnerReferenceNo == "" || in.Amount == nil {
		return ev, providers.ErrInvalidRequest
	}
	amt, e := parseAmount(in.Amount.Value)
	if e != nil {
		return ev, providers.ErrInvalidRequest
	}
	ev.ProviderRef, ev.Amount = in.OriginalPartnerReferenceNo, amt
	ev.Status = mapStatus(in.LatestTransactionStatus)
	if ev.Status == models.StatusPaid {
		ev.PaidAt = parseTime(in.PaidTime)
	}
	return ev, nil
}

func (c *Client) CallbackResponse(ev *providers.CallbackEvent, err error) (int, any) {
	switch ev.Endpoint {
	case endpointVAInquiry:
		return inquiryResponse(ev, err)
	case endpointVAPayment:
		return vaPaymentResponse(ev, err)
	case endpointSettlement:
		return settlementResponse(ev, err)
	}
	code, rc, msg := 200, "2005200", "Successful"
	switch {
	case err == nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, rc, msg = 401, "4015200", "Unauthorized. [Signature]"
	case errors.Is(err, providers.ErrInvalidRequest):
		code, rc, msg = 400, "4005201", "Invalid Field Format"
	case errors.Is(err, providers.ErrPaymentNotFound):
		code, rc, msg = 404, "4045201", "Transaction Not Found"
	case errors.Is(err, providers.ErrAmountMismatch):
		code, rc, msg = 404, "4045213", "Invalid Amount"
	case errors.Is(err, providers.ErrPaymentExpired):
		code, rc, msg = 404, "4045219", "Transaction Expired"
	default:
		code, rc, msg = 500, "5005200", "General Error"
	}
	return code, map[string]string{"responseCode": rc, "responseMessage": msg}
}

func freshTimestamp(ts string) bool {
	t, e := parseTS(ts)
	if e != nil {
		return false
	}
	d := time.Since(t)
	return d < maxSkew && d > -maxSkew
}
