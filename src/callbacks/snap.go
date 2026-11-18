package callbacks

import (
	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterSNAPRoutes registers Espay SNAP standard paths at the router root.
func RegisterSNAPRoutes(router gin.IRouter, db *gorm.DB) {
	c := NewController(NewService(db))
	g := router.Group("/api/webhooks/espay/v1.0/transfer-va")
	g.POST("/payment", c.SNAPVAPayment)
	g.POST("/inquiry", c.SNAPVAInquiry)
}

// SNAPVAInquiryRequest is the Espay SNAP VA inquiry body (docs only).
type SNAPVAInquiryRequest struct {
	PartnerServiceID string `json:"partnerServiceId" example:"   Espay"`
	CustomerNo       string `json:"customerNo" example:"SGWJAXERTEKNOINDO"`
	VirtualAccountNo string `json:"virtualAccountNo" example:"JXPEA6FFAAB9345C9016C3F"`
	TrxDateInit      string `json:"trxDateInit" example:"2026-11-12T09:31:00+07:00"`
	InquiryRequestID string `json:"inquiryRequestId" example:"INQ-20261112-0006"`
}

// SNAPAmount is a SNAP amount object (docs only).
type SNAPAmount struct {
	Value    string `json:"value" example:"10000.00"`
	Currency string `json:"currency" example:"IDR"`
}

// SNAPVAPaymentAdditionalInfo holds the Espay transaction status (docs only).
type SNAPVAPaymentAdditionalInfo struct {
	TransactionStatus string `json:"transactionStatus" example:"S"`
}

// SNAPVAPaymentRequest is the Espay SNAP VA payment notification body (docs only).
type SNAPVAPaymentRequest struct {
	PartnerServiceID string                      `json:"partnerServiceId" example:"   Espay"`
	CustomerNo       string                      `json:"customerNo" example:"SGWJAXERTEKNOINDO"`
	VirtualAccountNo string                      `json:"virtualAccountNo" example:"JXPEA6FFAAB9345C9016C3F"`
	PaymentRequestID string                      `json:"paymentRequestId" example:"PAY-20261112-0006"`
	TotalAmount      SNAPAmount                  `json:"totalAmount"`
	TrxDateTime      string                      `json:"trxDateTime" example:"2026-11-12T09:35:00+07:00"`
	AdditionalInfo   SNAPVAPaymentAdditionalInfo `json:"additionalInfo"`
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
