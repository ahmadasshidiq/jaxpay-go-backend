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
	TrxID            string  `json:"trxId"`
	PaymentRequestID string  `json:"paymentRequestId"`
	PaidAmount       *amount `json:"paidAmount"`
	TotalAmount      *amount `json:"totalAmount"`
	TrxDateTime      string  `json:"trxDateTime"`
	AdditionalInfo   struct {
		TransactionStatus string `json:"transactionStatus"`
		MemberCode        string `json:"memberCode"`
		DebitFrom         string `json:"debitFrom"`
		DebitFromName     string `json:"debitFromName"`
		DebitFromBank     string `json:"debitFromBank"`
		CreditTo          string `json:"creditTo"`
		CreditToName      string `json:"creditToName"`
		CreditToBank      string `json:"creditToBank"`
		ProductCode       string `json:"productCode"`
		ProductValue      string `json:"productValue"`
		Message           struct {
			ChannelFlag string `json:"CHANNEL_FLAG"`
		} `json:"message"`
		FeeType       string `json:"feeType"`
		TxFee         string `json:"txFee"`
		PaymentRef    string `json:"paymentRef"`
		PaymentRemark string `json:"paymentRemark"`
		RRN           string `json:"rrn"`
		ApprovalCode  string `json:"approvalCode"`
		Token         string `json:"token"`
		UserID        string `json:"userId"`
		TraceNumber   string `json:"traceNumber"`
		TerminalID    string `json:"terminalId"`
		CardType      string `json:"cardType"`
		CardExpiry    string `json:"cardExpiry"`
	} `json:"additionalInfo"`
	BillDetails []struct {
		BillNo string `json:"billNo"`
	} `json:"billDetails"`
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

	code, rc, msg := 200, "2002500", "Success"
	switch {
	case err == nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, rc, msg = 401, "4012500", "Unauthorized Signature"
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

	if rc != "2002500" {
		return code, snapErrorBody{ResponseCode: rc, ResponseMessage: msg}
	}
	var name, desc string
	total := amount{Currency: "IDR"}
	if in.TotalAmount != nil {
		total = *in.TotalAmount
	}
	if b := ev.Bill; b != nil {
		name, desc = b.CustomerName, trunc(b.Description, 18)
		if in.TotalAmount == nil {
			total = formatAmount(b.Amount)
		}
	}
	if desc == "" {
		desc = "Pembayaran"
	}
	h := sha256.Sum256([]byte(in.PaymentRequestID))
	return code, vaPaymentBody{
		ResponseCode:    rc,
		ResponseMessage: msg,
		VirtualAccountData: vaPaymentData{
			PartnerServiceID:   in.PartnerServiceID,
			CustomerNo:         in.CustomerNo,
			VirtualAccountNo:   in.VirtualAccountNo,
			VirtualAccountName: name,
			PaymentRequestID:   in.PaymentRequestID,
			TotalAmount:        total,
			BillDetails:        []inquiryBillDetail{{BillDescription: billDescription{English: desc, Indonesia: desc}}},
		},
		AdditionalInfo: reconcileInfo{
			ReconcileID:       hex.EncodeToString(h[:16]),
			ReconcileDatetime: time.Now().In(wib).Format(tsLayout),
		},
	}
}

type vaPaymentBody struct {
	ResponseCode       string        `json:"responseCode"`
	ResponseMessage    string        `json:"responseMessage"`
	VirtualAccountData vaPaymentData `json:"virtualAccountData"`
	AdditionalInfo     reconcileInfo `json:"additionalInfo"`
}

type vaPaymentData struct {
	PartnerServiceID   string              `json:"partnerServiceId"`
	CustomerNo         string              `json:"customerNo"`
	VirtualAccountNo   string              `json:"virtualAccountNo"`
	VirtualAccountName string              `json:"virtualAccountName"`
	PaymentRequestID   string              `json:"paymentRequestId"`
	TotalAmount        amount              `json:"totalAmount"`
	BillDetails        []inquiryBillDetail `json:"billDetails"`
}

type reconcileInfo struct {
	ReconcileID       string `json:"reconcileId"`
	ReconcileDatetime string `json:"reconcileDatetime"`
}
