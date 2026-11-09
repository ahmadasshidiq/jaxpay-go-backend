package cryptography

import (
	"net/http"
	"strings"
	"time"

	"jaxpay-go-backend/libs/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AdminClaims struct {
	Access []string `json:"access"`
	jwt.RegisteredClaims
}

func IssueAdminToken(userID string, access []string, ttl time.Duration) (string, error) {
	c := AdminClaims{
		Access: access,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"jaxpay-admin"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(config.Get().JWTSecret))
}

func JWTMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(x *gin.Context) {
		raw, ok := strings.CutPrefix(x.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			abort(x, http.StatusUnauthorized, "missing bearer token")
			return
		}
		var c AdminClaims
		_, e := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
			return []byte(config.Get().JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithAudience("jaxpay-admin"), jwt.WithExpirationRequired())
		if e != nil {
			abort(x, http.StatusUnauthorized, "invalid token")
			return
		}
		x.Set("user_id", c.Subject)
		x.Set("access", c.Access)
		x.Next()
	}
}

type accessMiddleware struct{}

var AccessMiddleware accessMiddleware

func (accessMiddleware) Handler(module, action string) gin.HandlerFunc {
	return func(x *gin.Context) {
		access, _ := x.Get("access")
		list, _ := access.([]string)
		for _, a := range list {
			if a == "*" || a == module+":*" || a == module+":"+action {
				x.Next()
				return
			}
		}
		abort(x, http.StatusForbidden, "forbidden")
	}
}

func abort(x *gin.Context, code int, msg string) {
	x.AbortWithStatusJSON(code, gin.H{"success": false, "error": msg})
}
