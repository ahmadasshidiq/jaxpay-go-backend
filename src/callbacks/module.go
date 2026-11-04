package callbacks

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Route callback provider: publik tanpa JWT, verifikasi signature dilakukan di adapter provider.
func RegisterCallbackModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	g := router.Group("/callbacks/:provider")
	g.POST("/payment", c.Handle)
	g.POST("/notify", c.Handle)
	g.POST("/inquiry", c.Inquiry)
}
