package espay

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"jaxpay-go-backend/libs/providers"
)

const endpointSettlement = "settlement"

type settlementData struct {
	TxID             string `json:"tx_id"`
	PaymentID        string `json:"payment_id"`
	SettlementAmount string `json:"settlement_amount"`
	SettlementDate   string `json:"settlement_date"`
	SettlementRemark string `json:"settlement_remark"`
}

func isSettlementPath(p string) bool { return strings.HasSuffix(p, "/settlement") }

const defaultSettlementSender = "GOWORLDPG"

func settlementSignature(rqUUID, rqDatetime, senderID, receiverID string) string {
	m := md5.Sum([]byte(rqUUID + rqDatetime + senderID + receiverID))
	s := sha1.Sum([]byte(hex.EncodeToString(m[:])))
	return hex.EncodeToString(s[:])
}

func (c *Client) parseSettlement(ev *providers.CallbackEvent, body []byte) error {
	ev.Type, ev.Endpoint = providers.CallbackSettlement, endpointSettlement
	f, e := url.ParseQuery(string(body))
	if e != nil || f.Get("rq_uuid") == "" {
		return providers.ErrInvalidRequest
	}
	if c.cfg.Password == "" || subtle.ConstantTimeCompare([]byte(f.Get("password")), []byte(c.cfg.Password)) != 1 {
		return providers.ErrInvalidSignature
	}
	sender, receiver := f.Get("sender_id"), f.Get("receiver_id")
	if sender == "" {
		sender = defaultSettlementSender
	}
	if receiver == "" {
		receiver = c.cfg.MerchantID
	}
	want := settlementSignature(f.Get("rq_uuid"), f.Get("rq_datetime"), sender, receiver)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(f.Get("signature"))), []byte(want)) != 1 {
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
		code, msg = "0001", "Invalid password or signature"
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
