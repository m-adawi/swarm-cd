package web

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var apiToken string

func init() {
	apiToken = os.Getenv("SWARMCD_API_TOKEN")
}

// MutationAPIEnabled reports whether the mutation API is active.
// The mutation API requires a bearer token set via SWARMCD_API_TOKEN.
func MutationAPIEnabled() bool {
	return apiToken != ""
}

// authMiddleware returns a Gin handler that enforces bearer token auth.
// If no token is configured, all mutation requests are rejected with 403.
func authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !MutationAPIEnabled() {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "mutation API is disabled (SWARMCD_API_TOKEN not set)",
			})
			return
		}

		header := ctx.GetHeader("Authorization")
		if header == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid Authorization header format (expected 'Bearer <token>')",
			})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiToken)) != 1 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		ctx.Next()
	}
}
