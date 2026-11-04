package webhooklogs

import (
	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const module = "webhook-logs"

type WebhookLogController struct {
	Service *WebhookLogService
}

func NewController(s *WebhookLogService) *WebhookLogController {
	return &WebhookLogController{Service: s}
}

// GetAll godoc
// @Summary      Riwayat pengiriman webhook
// @Tags         Webhook Logs
// @Produce      json
// @Param        page query int false "Page"
// @Param        limit query int false "Limit"
// @Param        sort query string false "Sort"
// @Param        payment_id query string false "Payment ID"
// @Param        client_id query string false "Client ID"
// @Param        event query string false "Event"
// @Param        status query string false "Status" Enums(PENDING, SUCCESS, FAILED)
// @Success      200 {object} helpers.Response{data=helpers.PaginatedResult{items=[]models.WebhookLog}}
// @Security     BearerAuth
// @Router       /webhook-logs [get]
func (c *WebhookLogController) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.getAll(d)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// GetByID godoc
// @Summary      Detail webhook log
// @Tags         Webhook Logs
// @Produce      json
// @Param        id path string true "Webhook log ID"
// @Success      200 {object} helpers.Response{data=models.WebhookLog}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /webhook-logs/{id} [get]
func (c *WebhookLogController) GetByID(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.getByID(id)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// Retry godoc
// @Summary      Retry manual pengiriman webhook
// @Tags         Webhook Logs
// @Produce      json
// @Param        id path string true "Webhook log ID"
// @Success      200 {object} helpers.Response{data=models.WebhookLog}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /webhook-logs/{id}/retry [post]
func (c *WebhookLogController) Retry(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.retry(id)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}
