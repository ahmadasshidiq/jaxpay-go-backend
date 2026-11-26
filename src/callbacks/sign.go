package callbacks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/providers"

	"github.com/gin-gonic/gin"
)

type callbackSigner interface {
	SignCallback(method, path string, body []byte) (ts, sig, sts string, err error)
}

type SignRequest struct {
	Method string          `json:"method" example:"POST"`
	Path   string          `json:"path" binding:"required" example:"/api/webhooks/espay/v1.0/transfer-va/payment"`
	Body   json.RawMessage `json:"body" swaggertype:"object"`
}

type SignResponse struct {
	XTimestamp   string `json:"x_timestamp" example:"2026-11-26T10:00:00+07:00"`
	XSignature   string `json:"x_signature"`
	StringToSign string `json:"string_to_sign"`
}

// Sign godoc
// @Summary      Generate X-TIMESTAMP & X-SIGNATURE untuk simulasi callback SNAP
// @Description  Hanya tersedia di non-production. Signature dibuat dengan private key merchant; callback non-production menerima signature ini selain signature Espay. Kirim body callback persis (atau versi minified-nya) bersama header yang dihasilkan, dalam 5 menit.
// @Tags         Callbacks
// @Accept       json
// @Produce      json
// @Param        provider path string true "Provider" Enums(espay)
// @Param        body body SignRequest true "Method, path, dan body callback"
// @Success      200 {object} SignResponse
// @Router       /callbacks/{provider}/sign [post]
func (c *CallbackController) Sign(x *gin.Context) {
	var in SignRequest
	if e := x.ShouldBindJSON(&in); e != nil {
		helpers.RespondError(x, module, http.StatusBadRequest, e)
		return
	}
	p, e := providers.Get(x.Param("provider"))
	if e != nil {
		helpers.RespondError(x, module, http.StatusNotFound, e)
		return
	}
	s, ok := p.(callbackSigner)
	if !ok {
		helpers.RespondError(x, module, http.StatusBadRequest, errors.New("provider does not support callback signing"))
		return
	}
	method := strings.ToUpper(in.Method)
	if method == "" {
		method = http.MethodPost
	}
	ts, sig, sts, e := s.SignCallback(method, in.Path, in.Body)
	if e != nil {
		helpers.RespondError(x, module, http.StatusForbidden, e)
		return
	}
	x.JSON(http.StatusOK, SignResponse{XTimestamp: ts, XSignature: sig, StringToSign: sts})
}
