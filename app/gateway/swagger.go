package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

//go:embed swagger.html
var swaggerPage []byte

func swaggerSigned(x *gin.Context) {
	x.Data(http.StatusOK, "text/html; charset=utf-8", swaggerPage)
}

func swaggerSpec(x *gin.Context) {
	doc, e := swag.ReadDoc()
	if e != nil {
		x.Status(http.StatusInternalServerError)
		return
	}
	x.Data(http.StatusOK, "application/json; charset=utf-8", rootSNAPPaths([]byte(doc)))
}

// rootSNAPPaths serves SNAP paths (/api/webhooks/...) from the root while keeping
// every other path under the original basePath.
func rootSNAPPaths(doc []byte) []byte {
	var spec map[string]any
	if json.Unmarshal(doc, &spec) != nil {
		return doc
	}
	base, _ := spec["basePath"].(string)
	paths, _ := spec["paths"].(map[string]any)
	if base == "" || base == "/" || paths == nil {
		return doc
	}
	out := make(map[string]any, len(paths))
	for p, v := range paths {
		if strings.HasPrefix(p, "/api/webhooks/") {
			out[p] = v
		} else {
			out[strings.TrimRight(base, "/")+p] = v
		}
	}
	spec["paths"] = out
	spec["basePath"] = "/"
	b, e := json.Marshal(spec)
	if e != nil {
		return doc
	}
	return b
}
