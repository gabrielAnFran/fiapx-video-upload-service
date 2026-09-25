package middleware

import (
	"net/http"
	"strings"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gin-gonic/gin"
)

const UserIDKey = "user_id"

// AuthRequired validates the Bearer JWT in the Authorization header and, on
// success, stores the authenticated user id in the gin context. The secret
// is passed in explicitly (sourced from cfg.JWTSecret by the caller) rather
// than read from a package-level var.
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorized(c, "Authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			unauthorized(c, "invalid authorization header format, expected: Bearer <token>")
			return
		}

		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			unauthorized(c, "invalid or expired token")
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func unauthorized(c *gin.Context, detail string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewProblem(
		http.StatusUnauthorized,
		"Unauthorized",
		detail,
		c.Request.URL.Path,
	))
}
