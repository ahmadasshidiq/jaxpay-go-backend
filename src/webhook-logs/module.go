package webhooklogs

import (
	"jaxpay-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterWebhookLogModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	acl := cryptography.AccessMiddleware
	g := router.Group("/webhook-logs", cryptography.JWTMiddleware(db))
	g.GET("", acl.Handler(module, "read"), c.GetAll)
	g.GET("/:id", acl.Handler(module, "read"), c.GetByID)
	g.POST("/:id/retry", acl.Handler(module, "update"), c.Retry)
}
