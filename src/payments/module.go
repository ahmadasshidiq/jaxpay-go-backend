package payments

import (
	"jaxpay-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterPaymentModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	g := router.Group("/payments", cryptography.ClientAuthMiddleware(db))
	g.POST("", c.Create)
	g.GET("", c.GetAll)
	g.GET("/:id", c.GetByID)
	g.POST("/:id/cancel", c.Cancel)
	g.POST("/:id/check-status", c.CheckStatus)
}
