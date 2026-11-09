package main

import (
	"net/http"

	_ "jaxpay-go-backend/docs"
	"jaxpay-go-backend/libs/config"

	"github.com/gin-gonic/gin"
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
	if cfg.AppEnv != "production" {
		r.GET("/docs", swaggerSigned)
		r.GET("/swagger/doc.json", swaggerSpec)
	}

	api := r.Group("/api/v1", RateLimiter(cfg.RateLimitPerMinute))
	RegisterAllModules(api, db)
	return r
}
