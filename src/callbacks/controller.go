package callbacks

import (
	"io"

	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
)

const module = "callbacks"

type CallbackController struct {
	Service *CallbackService
}

func NewController(s *CallbackService) *CallbackController {
	return &CallbackController{Service: s}
}

func readBody(x *gin.Context) ([]byte, bool) {
	b, e := io.ReadAll(io.LimitReader(x.Request.Body, 1<<20))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return nil, false
	}
	return b, true
}

// Inquiry godoc
// @Summary      Callback inquiry dari provider
// @Description  Provider menanyakan detail tagihan (mis. Espay service code 24). Signature diverifikasi oleh adapter provider; balasan mengikuti format provider.
// @Tags         Callbacks
// @Accept       json
// @Produce      json
// @Param        provider path string true "Provider" Enums(espay)
// @Param        X-TIMESTAMP header string true "Timestamp ISO-8601"
// @Param        X-SIGNATURE header string true "SHA256withRSA signature (public key provider)"
// @Param        body body object true "Payload provider"
// @Success      200 {object} SNAPResponse
// @Failure      401 {object} SNAPResponse
// @Failure      404 {object} SNAPResponse
// @Router       /callbacks/{provider}/inquiry [post]
func (c *CallbackController) Inquiry(x *gin.Context) { c.Handle(x) }

// Settlement godoc
// @Summary      Callback settlement dari provider
// @Description  Informasi pelunasan dana ke merchant (Espay Settlement Notification). Body form-urlencoded, divalidasi dengan password merchant dan signature sha1(md5(rq_uuid + rq_datetime + sender_id + receiver_id)). Tidak mengubah status payment.
// @Tags         Callbacks
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        provider path string true "Provider" Enums(espay)
// @Param        rq_uuid formData string true "Request ID"
// @Param        rq_datetime formData string true "yyyy-MM-dd hh:mi:ss"
// @Param        password formData string true "Password merchant"
// @Param        data formData string true "JSON: tx_id, payment_id, settlement_amount, settlement_date, settlement_remark"
// @Param        sender_id formData string false "Sender ID (default GOWORLDPG)"
// @Param        receiver_id formData string false "Receiver ID / comm_code (default ESPAY_MERCHANT_ID)"
// @Param        signature formData string true "Hash-based signature"
// @Success      200 {object} map[string]string
// @Router       /callbacks/{provider}/settlement [post]
func (c *CallbackController) Settlement(x *gin.Context) { c.Handle(x) }

// Handle godoc
// @Summary      Callback payment dari provider
// @Description  Notifikasi pembayaran / perubahan status dari provider. Signature diverifikasi oleh adapter provider; balasan mengikuti format provider.
// @Tags         Callbacks
// @Accept       json
// @Produce      json
// @Param        provider path string true "Provider" Enums(espay)
// @Param        X-TIMESTAMP header string true "Timestamp ISO-8601"
// @Param        X-SIGNATURE header string true "SHA256withRSA signature (public key provider)"
// @Param        body body object true "Payload provider"
// @Success      200 {object} SNAPResponse
// @Failure      401 {object} SNAPResponse
// @Failure      404 {object} SNAPResponse
// @Router       /callbacks/{provider}/payment [post]
func (c *CallbackController) Handle(x *gin.Context) {
	b, ok := readBody(x)
	if !ok {
		return
	}
	code, v, e := c.Service.handle(x.Param("provider"), x.Request, b)
	if e == errUnknownProvider {
		helpers.RespondError(x, module, 404, e)
		return
	}
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	x.JSON(code, v)
}
