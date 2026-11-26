package espay

import (
	"encoding/json"
	"errors"
	"strings"

	"jaxpay-go-backend/libs/providers"
)

const endpointVAInquiry = "va_inquiry"

type inquiryRequest struct {
	PartnerServiceID string `json:"partnerServiceId"`
	CustomerNo       string `json:"customerNo"`
	VirtualAccountNo string `json:"virtualAccountNo"`
	TrxDateInit      string `json:"trxDateInit"`
	InquiryRequestID string `json:"inquiryRequestId"`
}

func isInquiryPath(p string) bool { return strings.HasSuffix(p, "/inquiry") }

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

func inquiryResponse(ev *providers.CallbackEvent, err error) (int, any) {
	var in inquiryRequest
	_ = json.Unmarshal(ev.Raw, &in)

	code, rc, msg := 200, "2002400", "Success"
	switch {
	case err == nil && ev.Bill != nil:
	case errors.Is(err, providers.ErrInvalidSignature):
		code, rc, msg = 401, "4012400", "Unauthorized Signature"
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

	b := ev.Bill
	if rc != "2002400" || b == nil {
		return code, inquiryErrorBody{ResponseCode: rc, ResponseMessage: msg}
	}
	desc := trunc(b.Description, 18)
	if desc == "" {
		desc = "Pembayaran"
	}
	var info inquiryAdditionalInfo
	_ = json.Unmarshal(b.Metadata, &info)
	info.TransactionDate = b.CreatedAt.In(wib).Format(tsLayout)
	if info.Items == nil {
		info.Items = []inquiryItem{}
	}
	if info.DataMerchant.ListBills == nil {
		info.DataMerchant.ListBills = []inquiryBill{}
	}
	return code, inquiryBody{ResponseCode: rc, ResponseMessage: msg, VirtualAccountData: inquiryVADataFull{
		PartnerServiceID:    in.PartnerServiceID,
		CustomerNo:          in.CustomerNo,
		VirtualAccountNo:    in.VirtualAccountNo,
		VirtualAccountName:  b.CustomerName,
		VirtualAccountEmail: b.CustomerEmail,
		VirtualAccountPhone: b.CustomerPhone,
		InquiryRequestID:    in.InquiryRequestID,
		TotalAmount:         formatAmount(b.Amount),
		BillDetails:         []inquiryBillDetail{{BillDescription: billDescription{English: desc, Indonesia: desc}}},
		AdditionalInfo:      info,
	}}
}
