package cryptography

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"

	"jaxpay-go-backend/libs/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxSkew = 5 * time.Minute

// ClientAuthMiddleware memverifikasi request dari app client:
//
//	X-API-Key:   api key client
//	X-Timestamp: unix seconds (selisih maks 5 menit)
//	X-Signature: hex(HMAC-SHA256(api_secret, timestamp + "." + METHOD + "." + request_uri + "." + body))
func ClientAuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(x *gin.Context) {
		key, ts, sig := x.GetHeader("X-API-Key"), x.GetHeader("X-Timestamp"), x.GetHeader("X-Signature")
		if key == "" || ts == "" || sig == "" {
			abort(x, http.StatusUnauthorized, "missing auth headers")
			return
		}
		sec, e := strconv.ParseInt(ts, 10, 64)
		if e != nil || absDur(time.Since(time.Unix(sec, 0))) > maxSkew {
			abort(x, http.StatusUnauthorized, "invalid timestamp")
			return
		}
		body, e := io.ReadAll(io.LimitReader(x.Request.Body, 1<<20))
		if e != nil {
			abort(x, http.StatusBadRequest, "cannot read body")
			return
		}
		x.Request.Body = io.NopCloser(bytes.NewReader(body))

		var c models.Client
		if e = db.Where("api_key = ? AND is_active = ?", key, true).First(&c).Error; e != nil {
			abort(x, http.StatusUnauthorized, "unauthorized")
			return
		}
		secret, e := Decrypt(c.APISecretHash)
		if e != nil {
			abort(x, http.StatusInternalServerError, "internal error")
			return
		}
		msg := ts + "." + x.Request.Method + "." + x.Request.URL.RequestURI() + "." + string(body)
		if !VerifyHMACSHA256(secret, msg, sig) {
			abort(x, http.StatusUnauthorized, "invalid signature")
			return
		}
		x.Set("client_id", c.ID.String())
		x.Set("client", &c)
		x.Next()
	}
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
