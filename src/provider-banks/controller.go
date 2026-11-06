package providerbanks

import (
	"errors"
	"strings"

	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const module = "provider-banks"

type ProviderBankController struct {
	Service *ProviderBankService
}

func NewController(s *ProviderBankService) *ProviderBankController {
	return &ProviderBankController{Service: s}
}

func respondWriteError(x *gin.Context, e error) {
	switch {
	case strings.Contains(e.Error(), "duplicate key"):
		helpers.RespondError(x, module, 409, errors.New("product_code already exists for this provider"))
	default:
		helpers.RespondError(x, module, 500, e)
	}
}

// GetAll godoc
// @Summary      List master bank / produk provider
// @Tags         Provider Banks
// @Produce      json
// @Param        page query int false "Page"
// @Param        limit query int false "Limit"
// @Param        sort query string false "Sort"
// @Param        provider query string false "Provider, mis. espay"
// @Param        bank_code query string false "Kode bank"
// @Param        is_va query bool false "Hanya produk VA"
// @Param        is_active query bool false "Status aktif"
// @Param        search query string false "Cari nama"
// @Success      200 {object} helpers.Response{data=helpers.PaginatedResult{items=[]models.ProviderBank}}
// @Security     BearerAuth
// @Router       /provider-banks [get]
func (c *ProviderBankController) GetAll(x *gin.Context) {
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
// @Summary      Detail master bank
// @Tags         Provider Banks
// @Produce      json
// @Param        id path string true "Bank ID"
// @Success      200 {object} helpers.Response{data=models.ProviderBank}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-banks/{id} [get]
func (c *ProviderBankController) GetByID(x *gin.Context) {
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

// Create godoc
// @Summary      Tambah master bank
// @Tags         Provider Banks
// @Accept       json
// @Produce      json
// @Param        body body CreateDTO true "Bank"
// @Success      201 {object} helpers.Response
// @Failure      400 {object} helpers.Response
// @Failure      409 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-banks [post]
func (c *ProviderBankController) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.create(d)
	if e != nil {
		respondWriteError(x, e)
		return
	}
	helpers.RespondSuccess(x, module, 201, v.ID)
}

// Update godoc
// @Summary      Update master bank
// @Tags         Provider Banks
// @Accept       json
// @Produce      json
// @Param        id path string true "Bank ID"
// @Param        body body UpdateDTO true "Field yang diubah"
// @Success      200 {object} helpers.Response{data=models.ProviderBank}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-banks/{id} [patch]
func (c *ProviderBankController) Update(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	var d UpdateDTO
	if e = x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.update(id, d)
	if e != nil {
		respondWriteError(x, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// Remove godoc
// @Summary      Hapus master bank
// @Tags         Provider Banks
// @Produce      json
// @Param        id path string true "Bank ID"
// @Success      200 {object} helpers.Response
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-banks/{id} [delete]
func (c *ProviderBankController) Remove(x *gin.Context) {
	helpers.HandleRemove(x, module, c.Service.archive, c.Service.delete)
}
