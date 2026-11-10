package main

import (
	"net/http"
	"strings"

	"jaxpay-go-backend/libs/config"

	"github.com/gin-gonic/gin"
)

func CORS(c config.CORSConfig) gin.HandlerFunc {
	allowAll := false
	origins := map[string]bool{}
	for _, o := range c.AllowOrigins {
		if o == "*" {
			allowAll = true
		}
		origins[o] = true
	}
	methods := strings.Join(c.AllowMethods, ", ")
	headers := strings.Join(c.AllowHeaders, ", ")

	return func(x *gin.Context) {
		h := x.Writer.Header()
		origin := x.Request.Header.Get("Origin")
		switch {
		case allowAll:
			h.Set("Access-Control-Allow-Origin", "*")
		case origin != "" && origins[origin]:
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
		}
		h.Set("Access-Control-Allow-Methods", methods)
		h.Set("Access-Control-Allow-Headers", headers)
		h.Set("Access-Control-Max-Age", "86400")
		if x.Request.Method == http.MethodOptions {
			x.AbortWithStatus(http.StatusNoContent)
			return
		}
		x.Next()
	}
}
