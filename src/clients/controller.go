package clients

import (
	"jaxpay-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const module = "clients"

type ClientController struct {
	Service *ClientService
}

func NewController(s *ClientService) *ClientController {
	return &ClientController{Service: s}
}

// GetAll godoc
// @Summary      List client app
// @Tags         Clients
// @Produce      json
// @Param        page query int false "Page"
// @Param        limit query int false "Limit"
// @Param        sort query string false "Sort, mis. -created_at"
// @Param        search query string false "Cari code / name"
// @Param        is_active query bool false "Filter aktif"
// @Success      200 {object} helpers.Response{data=helpers.PaginatedResult{items=[]models.Client}}
// @Security     BearerAuth
// @Router       /clients [get]
func (c *ClientController) GetAll(x *gin.Context) {
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
// @Summary      Detail client app
// @Tags         Clients
// @Produce      json
// @Param        id path string true "Client ID"
// @Success      200 {object} helpers.Response{data=models.Client}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /clients/{id} [get]
func (c *ClientController) GetByID(x *gin.Context) {
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
// @Summary      Daftarkan client app
// @Description  Mengembalikan api_key, api_secret dan webhook_secret. Secret hanya ditampilkan sekali.
// @Tags         Clients
// @Accept       json
// @Produce      json
// @Param        body body CreateDTO true "Client"
// @Success      201 {object} helpers.Response{data=CredentialsDTO}
// @Failure      400 {object} helpers.Response
// @Security     BearerAuth
// @Router       /clients [post]
func (c *ClientController) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.create(d)
	if e != nil {
		helpers.RespondError(x, module, 500, e)
		return
	}
	helpers.RespondSuccess(x, module, 201, v)
}

// Update godoc
// @Summary      Update client app
// @Tags         Clients
// @Accept       json
// @Produce      json
// @Param        id path string true "Client ID"
// @Param        body body UpdateDTO true "Field yang diubah"
// @Success      200 {object} helpers.Response{data=models.Client}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /clients/{id} [patch]
func (c *ClientController) Update(x *gin.Context) {
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
		helpers.RespondError(x, module, 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, module, 404, nil)
		return
	}
	helpers.RespondSuccess(x, module, 200, v)
}

// RotateSecret godoc
// @Summary      Rotasi API key & secret
// @Description  Credential lama langsung tidak berlaku.
// @Tags         Clients
// @Produce      json
// @Param        id path string true "Client ID"
// @Success      200 {object} helpers.Response{data=CredentialsDTO}
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /clients/{id}/rotate-secret [post]
func (c *ClientController) RotateSecret(x *gin.Context) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, module, 400, e)
		return
	}
	v, e := c.Service.rotateSecret(id)
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

// Remove godoc
// @Summary      Archive / hapus client app
// @Tags         Clients
// @Produce      json
// @Param        id path string true "Client ID"
// @Param        permanent query bool false "Hapus permanen"
// @Success      200 {object} helpers.Response
// @Failure      404 {object} helpers.Response
// @Security     BearerAuth
// @Router       /clients/{id} [delete]
func (c *ClientController) Remove(x *gin.Context) {
	helpers.HandleRemove(x, module, c.Service.archive, c.Service.delete)
}
