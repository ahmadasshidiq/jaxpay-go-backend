package main

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

//go:embed swagger.html
var swaggerPage []byte

// swaggerSigned = Swagger UI dengan requestInterceptor yang menandatangani request client (HMAC).
func swaggerSigned(x *gin.Context) {
	x.Data(http.StatusOK, "text/html; charset=utf-8", swaggerPage)
}

// swaggerSpec menyajikan spesifikasi OpenAPI hasil swag init (docs/docs.go).
func swaggerSpec(x *gin.Context) {
	doc, e := swag.ReadDoc()
	if e != nil {
		x.Status(http.StatusInternalServerError)
		return
	}
	x.Data(http.StatusOK, "application/json; charset=utf-8", []byte(doc))
}
