package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(x *gin.Context) {
		h := x.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Timestamp, X-Signature")
		h.Set("Access-Control-Max-Age", "86400")
		if x.Request.Method == http.MethodOptions {
			x.AbortWithStatus(http.StatusNoContent)
			return
		}
		x.Next()
	}
}
