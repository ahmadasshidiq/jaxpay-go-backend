package main

import (
	"net/http"

	_ "jaxpay-go-backend/docs"
	"jaxpay-go-backend/libs/config"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), CORS())

	r.GET("/health", func(x *gin.Context) {
		if sqlDB, e := db.DB(); e != nil || sqlDB.Ping() != nil {
			x.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		x.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1", RateLimiter(cfg.RateLimitPerMinute))
	RegisterAllModules(api, db)
	return r
}
