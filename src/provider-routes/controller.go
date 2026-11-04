package providerroutes

import (
	"errors"
	"strings"

	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const module = "provider-routes"

type ProviderRouteController struct {
	Service *ProviderRouteService
}

func NewController(s *ProviderRouteService) *ProviderRouteController {
	return &ProviderRouteController{Service: s}
}

func respondWriteError(x *gin.Context, e error) {
	switch {
	case errors.Is(e, errClientNotFound), strings.Contains(e.Error(), "not registered"):
		helpers.RespondError(x, module, 400, e)
	case strings.Contains(e.Error(), "duplicate key"):
		helpers.RespondError(x, module, 409, errors.New("route for this client and method already exists"))
	default:
		helpers.RespondError(x, module, 500, e)
	}
}

// GetAll godoc
// @Summary      List mapping client + method → provider
// @Tags         Provider Routes
// @Produce      json
// @Param        page query int false "Page"
// @Param        limit query int false "Limit"
// @Param        sort query string false "Sort"
// @Param        client_id query string false "Client ID"
// @Param        method query string false "Method"
// @Param        provider query string false "Provider"
// @Success      200 {object} helpers.Response{data=helpers.PaginatedResult{items=[]models.ProviderRoute}}
// @Security     BearerAuth
// @Router       /provider-routes [get]
func (c *ProviderRouteController) GetAll(x *gin.Context) {
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
// @Summary      Detail provider route
// @Tags         Provider Routes
// @Produce      json
// @Param        id path string true "Route ID"
// @Success      200 {object} helpers.Response{data=models.ProviderRoute}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-routes/{id} [get]
func (c *ProviderRouteController) GetByID(x *gin.Context) {
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
// @Summary      Buat provider route
// @Tags         Provider Routes
// @Accept       json
// @Produce      json
// @Param        body body CreateDTO true "Route"
// @Success      201 {object} helpers.Response
// @Failure      400 {object} helpers.Response
// @Failure      409 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-routes [post]
func (c *ProviderRouteController) Create(x *gin.Context) {
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
// @Summary      Update provider route
// @Tags         Provider Routes
// @Accept       json
// @Produce      json
// @Param        id path string true "Route ID"
// @Param        body body UpdateDTO true "Field yang diubah"
// @Success      200 {object} helpers.Response{data=models.ProviderRoute}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-routes/{id} [patch]
func (c *ProviderRouteController) Update(x *gin.Context) {
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
// @Summary      Hapus provider route
// @Tags         Provider Routes
// @Produce      json
// @Param        id path string true "Route ID"
// @Success      200 {object} helpers.Response
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /provider-routes/{id} [delete]
func (c *ProviderRouteController) Remove(x *gin.Context) {
	helpers.HandleRemove(x, module, c.Service.archive, c.Service.delete)
}
