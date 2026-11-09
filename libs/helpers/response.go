package helpers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Response struct {
	Success bool   `json:"success"`
	Module  string `json:"module"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func RespondSuccess(x *gin.Context, module string, code int, data any) {
	x.JSON(code, Response{Success: true, Module: module, Data: data})
}

func RespondError(x *gin.Context, module string, code int, e error) {
	msg := http.StatusText(code)
	if e != nil {
		if code >= 500 {
			log.Printf("[%s] %d: %v", module, code, e)
		} else {
			msg = e.Error()
		}
	}
	x.AbortWithStatusJSON(code, Response{Success: false, Module: module, Error: msg})
}

func HandleRemove(x *gin.Context, module string, archive, remove func(id uuid.UUID) (bool, error)) {
	id, e := uuid.Parse(x.Param("id"))
	if e != nil {
		RespondError(x, module, 400, e)
		return
	}
	fn := archive
	if x.Query("permanent") == "true" {
		fn = remove
	}
	ok, e := fn(id)
	if e != nil {
		RespondError(x, module, 500, e)
		return
	}
	if !ok {
		RespondError(x, module, 404, nil)
		return
	}
	RespondSuccess(x, module, 200, id)
}
