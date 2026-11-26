package callbacks

import (
	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterSNAPRoutes(router gin.IRouter, db *gorm.DB) {
	c := NewController(NewService(db))
	g := router.Group("/api/webhooks/espay/v1.0/transfer-va")
	g.POST("/payment", c.SNAPVAPayment)
	g.POST("/inquiry", c.SNAPVAInquiry)
}

type SNAPVAInquiryRequest struct {
	PartnerServiceID string `json:"partnerServiceId" example:"   Espay"`
	CustomerNo       string `json:"customerNo" example:"SGWJAXERTEKNOINDO"`
	VirtualAccountNo string `json:"virtualAccountNo" example:"JXPEA6FFAAB9345C9016C3F"`
	TrxDateInit      string `json:"trxDateInit" example:"2026-11-12T09:31:00+07:00"`
	InquiryRequestID string `json:"inquiryRequestId" example:"INQ-20261112-0006"`
}

type SNAPAmount struct {
	Value    string `json:"value" example:"10000.00"`
	Currency string `json:"currency" example:"IDR"`
}

type SNAPVAPaymentMessage struct {
	ChannelFlag string `json:"CHANNEL_FLAG" example:"W"`
}

type SNAPVAPaymentAdditionalInfo struct {
	TransactionStatus string               `json:"transactionStatus" example:"S"`
	MemberCode        string               `json:"memberCode" example:"SGWJAXERTEKNOINDO"`
	DebitFrom         string               `json:"debitFrom" example:"16095902XXXX"`
	DebitFromName     string               `json:"debitFromName" example:"BRI"`
	DebitFromBank     string               `json:"debitFromBank" example:"002"`
	CreditTo          string               `json:"creditTo" example:"665050XXXX"`
	CreditToName      string               `json:"creditToName" example:"BRI"`
	CreditToBank      string               `json:"creditToBank" example:"002"`
	ProductCode       string               `json:"productCode" example:"BRIATM"`
	ProductValue      string               `json:"productValue" example:"16095902XXXX"`
	Message           SNAPVAPaymentMessage `json:"message"`
	FeeType           string               `json:"feeType" example:"S"`
	TxFee             string               `json:"txFee" example:"4400"`
	PaymentRef        string               `json:"paymentRef" example:"ESP1661161958ZQEG"`
	PaymentRemark     string               `json:"paymentRemark" example:"Bayar"`
	RRN               string               `json:"rrn" example:"220822000288"`
	ApprovalCode      string               `json:"approvalCode" example:"220822000288"`
	Token             string               `json:"token" example:"2023011167618274122"`
	UserID            string               `json:"userId" example:"2306291049P89VWB"`
	TraceNumber       string               `json:"traceNumber" example:"7999999901"`
	TerminalID        string               `json:"terminalId" example:"10380579"`
	CardType          string               `json:"cardType" example:"credit"`
	CardExpiry        string               `json:"cardExpiry" example:"202612"`
}

type SNAPVAPaymentBillDetail struct {
	BillNo string `json:"billNo" example:"invoiceid0001"`
}

type SNAPVAPaymentRequest struct {
	PartnerServiceID string                      `json:"partnerServiceId" example:" Espay"`
	CustomerNo       string                      `json:"customerNo" example:"SGWJAXERTEKNOINDO"`
	VirtualAccountNo string                      `json:"virtualAccountNo" example:"JXPEA6FFAAB9345C9016C3F"`
	TrxID            string                      `json:"trxId" example:"ESP1661161958ZQEG"`
	PaymentRequestID string                      `json:"paymentRequestId" example:"PAY-20261112-0006"`
	PaidAmount       SNAPAmount                  `json:"paidAmount"`
	TotalAmount      SNAPAmount                  `json:"totalAmount"`
	TrxDateTime      string                      `json:"trxDateTime" example:"2026-11-12T09:35:00+07:00"`
	AdditionalInfo   SNAPVAPaymentAdditionalInfo `json:"additionalInfo"`
	BillDetails      []SNAPVAPaymentBillDetail   `json:"billDetails"`
}

// SNAPVAPayment godoc
// @Summary      Espay SNAP VA payment notification
// @Description  Path standar SNAP Espay untuk notifikasi pembayaran Virtual Account. Logika sama dengan /callbacks/espay/payment.
// @Tags         Callbacks
// @Accept       json
// @Produce      json
// @Param        X-TIMESTAMP header string true "Timestamp ISO-8601"
// @Param        X-SIGNATURE header string true "SHA256withRSA signature (public key provider)"
// @Param        body body SNAPVAPaymentRequest true "Payload provider"
// @Success      200 {object} SNAPResponse
// @Failure      401 {object} SNAPResponse
// @Router       /api/webhooks/espay/v1.0/transfer-va/payment [post]
func (c *CallbackController) SNAPVAPayment(x *gin.Context) { c.handleEspay(x) }

// SNAPVAInquiry godoc
// @Summary      Espay SNAP VA inquiry
// @Description  Path standar SNAP Espay untuk inquiry tagihan Virtual Account. Logika sama dengan /callbacks/espay/inquiry.
// @Tags         Callbacks
// @Accept       json
// @Produce      json
// @Param        X-TIMESTAMP header string true "Timestamp ISO-8601"
// @Param        X-SIGNATURE header string true "SHA256withRSA signature (public key provider)"
// @Param        body body SNAPVAInquiryRequest true "Payload provider"
// @Success      200 {object} SNAPResponse
// @Failure      401 {object} SNAPResponse
// @Router       /api/webhooks/espay/v1.0/transfer-va/inquiry [post]
func (c *CallbackController) SNAPVAInquiry(x *gin.Context) { c.handleEspay(x) }

func (c *CallbackController) handleEspay(x *gin.Context) {
	b, ok := readBody(x)
	if !ok {
		return
	}
	code, v, e := c.Service.handle("espay", x.Request, b)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	x.JSON(code, v)
}
