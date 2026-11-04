package clients

import (
	"jaxpay-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterClientModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	acl := cryptography.AccessMiddleware
	g := router.Group("/clients", cryptography.JWTMiddleware(db))
	g.GET("", acl.Handler(module, "read"), c.GetAll)
	g.GET("/:id", acl.Handler(module, "read"), c.GetByID)
	g.POST("", acl.Handler(module, "create"), c.Create)
	g.PATCH("/:id", acl.Handler(module, "update"), c.Update)
	g.POST("/:id/rotate-secret", acl.Handler(module, "update"), c.RotateSecret)
	g.DELETE("/:id", acl.Handler(module, "delete"), c.Remove)
}
