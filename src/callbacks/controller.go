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

// Handle godoc
// @Summary      Callback dari provider
// @Description  Endpoint publik untuk provider: /espay/qris/notify. Signature diverifikasi oleh adapter provider; balasan mengikuti format provider.
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
// @Router       /callbacks/{provider}/qris/notify [post]
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
