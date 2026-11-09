package callbacks

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterCallbackModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	g := router.Group("/callbacks/:provider")
	g.POST("/payment", c.Handle)
	g.POST("/notify", c.Handle)
	g.POST("/inquiry", c.Inquiry)
	g.POST("/settlement", c.Settlement)
}
