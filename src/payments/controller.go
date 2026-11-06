package payments

import (
	"errors"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const module = "payments"

type PaymentController struct {
	Service *PaymentService
}

func NewController(s *PaymentService) *PaymentController {
	return &PaymentController{Service: s}
}

func clientID(x *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(x.GetString("client_id"))
	return id
}

func respondServiceError(x *gin.Context, e error) {
	switch {
	case errors.Is(e, errRouteNotFound), errors.Is(e, providers.ErrNotSupported), errors.Is(e, errInvalidBank):
		helpers.RespondError(x, module, 422, e)
	case errors.Is(e, errNotPending), errors.Is(e, errRefConflict):
		helpers.RespondError(x, module, 409, e)
	case errors.Is(e, errProviderFailure):
		helpers.RespondError(x, module, 502, e)
	default:
		helpers.RespondError(x, module, 500, e)
	}
}

// Create godoc
// @Summary      Buat transaksi pembayaran
// @Description  Membuat VA / QRIS. Untuk method "va", bank_code (3 digit kode bank Espay) wajib diisi. Idempotent berdasarkan reference_id (request ulang mengembalikan payment yang sama dengan status 200).
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        body body CreatePaymentDTO true "Payment"
// @Success      201 {object} helpers.Response{data=PaymentResponse}
// @Success      200 {object} helpers.Response{data=PaymentResponse}
// @Failure      400 {object} helpers.Response
// @Failure      409 {object} helpers.Response
// @Failure      422 {object} helpers.Response
// @Failure      502 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments [post]
func (c *PaymentController) Create(x *gin.Context) {
	var d CreatePaymentDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	client := x.MustGet("client").(*models.Client)
	v, created, e := c.Service.create(x.Request.Context(), client, d)
	if e != nil {
		respondServiceError(x, e)
		return
	}
	code := 200
	if created {
		code = 201
	}
	helpers.RespondSuccess(x, module, code, toResponse(v))
}

// VABanks godoc
// @Summary      Daftar bank Virtual Account
// @Description  Bank VA yang didukung provider route VA aktif milik client. Gunakan bank_code saat membuat payment method "va".
// @Tags         Payments
// @Produce      json
// @Success      200 {object} helpers.Response{data=[]VABankResponse}
// @Failure      422 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments/va-banks [get]
func (c *PaymentController) VABanks(x *gin.Context) {
	v, e := c.Service.vaBanks(clientID(x))
	if e != nil {
		respondServiceError(x, e)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// GetAll godoc
// @Summary      List transaksi milik client
// @Tags         Payments
// @Produce      json
// @Param        page query int false "Page"
// @Param        limit query int false "Limit (maks 100)"
// @Param        sort query string false "Sort, mis. -created_at"
// @Param        status query string false "Status" Enums(PENDING, PAID, EXPIRED, FAILED, CANCELLED)
// @Param        method query string false "Method" Enums(qris, va)
// @Param        reference_id query string false "Reference ID"
// @Success      200 {object} helpers.Response{data=helpers.PaginatedResult{items=[]PaymentResponse}}
// @Failure      400 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments [get]
func (c *PaymentController) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.getAll(clientID(x), d)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// GetByID godoc
// @Summary      Detail + status transaksi
// @Tags         Payments
// @Produce      json
// @Param        id path string true "Payment ID"
// @Success      200 {object} helpers.Response{data=PaymentResponse}
// @Failure      404 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments/{id} [get]
func (c *PaymentController) GetByID(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.getByID(clientID(x), id)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, toResponse(v))
}

// Cancel godoc
// @Summary      Batalkan transaksi
// @Description  Hanya untuk payment PENDING. Membatalkan QR di provider / menutup VA.
// @Tags         Payments
// @Produce      json
// @Param        id path string true "Payment ID"
// @Success      200 {object} helpers.Response{data=PaymentResponse}
// @Failure      404 {object} helpers.Response
// @Failure      409 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments/{id}/cancel [post]
func (c *PaymentController) Cancel(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.cancel(x.Request.Context(), clientID(x), id)
	if e != nil {
		respondServiceError(x, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, toResponse(v))
}

// CheckStatus godoc
// @Summary      Paksa cek status ke provider
// @Tags         Payments
// @Produce      json
// @Param        id path string true "Payment ID"
// @Success      200 {object} helpers.Response{data=PaymentResponse}
// @Failure      404 {object} helpers.Response
// @Failure      502 {object} helpers.Response
// @Security     ClientAPIKey
// @Router       /payments/{id}/check-status [post]
func (c *PaymentController) CheckStatus(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.checkStatus(x.Request.Context(), clientID(x), id)
	if e != nil {
		respondServiceError(x, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, toResponse(v))
}
