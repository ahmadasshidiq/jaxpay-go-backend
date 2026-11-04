package espay

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"jaxpay-go-backend/libs/providers"
)

const endpointSettlement = "settlement"

// settlementData = field "data" (JSON string di dalam form-urlencoded).
type settlementData struct {
	TxID             string `json:"tx_id"`
	PaymentID        string `json:"payment_id"`
	SettlementAmount string `json:"settlement_amount"`
	SettlementDate   string `json:"settlement_date"`
	SettlementRemark string `json:"settlement_remark"`
}

func isSettlementPath(p string) bool { return strings.HasSuffix(p, "/settlement") }

// parseSettlement: body application/x-www-form-urlencoded, divalidasi dengan password merchant.
// TODO: verifikasi field "signature" (Hash-Based Signature) setelah rumusnya dari dokumentasi Espay tersedia.
func (c *Client) parseSettlement(ev *providers.CallbackEvent, body []byte) error {
	ev.Type, ev.Endpoint = providers.CallbackSettlement, endpointSettlement
	f, e := url.ParseQuery(string(body))
	if e != nil || f.Get("rq_uuid") == "" {
		return providers.ErrInvalidRequest
	}
	if c.cfg.Password == "" || subtle.ConstantTimeCompare([]byte(f.Get("password")), []byte(c.cfg.Password)) != 1 {
		return providers.ErrInvalidSignature
	}
	var d settlementData
	if json.Unmarshal([]byte(f.Get("data")), &d) != nil {
		return providers.ErrInvalidRequest
	}
	ev.ProviderRef = d.PaymentID
	ev.Amount, _ = parseAmount(d.SettlementAmount)
	return nil
}

func settlementResponse(ev *providers.CallbackEvent, err error) (int, any) {
	code, msg := "0000", "Success"
	switch {
	case err == nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, msg = "0001", "Invalid password"
	case errors.Is(err, providers.ErrInvalidRequest):
		code, msg = "0002", "Invalid request"
	default:
		code, msg = "0099", "General error"
	}
	return 200, map[string]string{
		"error_code":    code,
		"error_message": msg,
		"date_settle":   time.Now().In(wib).Format("2006-01-02 15:04:05"),
	}
}
