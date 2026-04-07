package middleware

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"products/internal/gen"
	"products/internal/utils"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AuthMiddleware(pub *rsa.PublicKey, roles []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			makeInvalidTokenResponse(ctx)
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			makeInvalidTokenResponse(ctx)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, prefix)

		claims, err := utils.VerifyAccessToken(pub, tokenStr)
		if err != nil {
			if errors.Is(err, utils.ErrJWTExpired) {
				ctx.AbortWithStatusJSON(
					http.StatusUnauthorized,
					gen.UnauthorizedAccessToken{
						ErrorCode: gen.TOKENEXPIRED,
						Message:   "token expired",
					},
				)
				return
			}
			makeInvalidTokenResponse(ctx)
			return
		}

		if !slices.Contains(roles, claims["role"].(string)) {
			ctx.AbortWithStatusJSON(
				http.StatusForbidden,
				gen.AccessDenied{
					ErrorCode: gen.ACCESSDENIED,
					Message:   "access denied",
				},
			)
		}
		val, err := uuid.Parse(claims["sub"].(string))
		if err != nil {
			makeInvalidTokenResponse(ctx)
			return
		}
		ctx.Set("userID", val)
		ctx.Set("role", claims["role"])

		ctx.Next()
	}
}

func makeInvalidTokenResponse(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(
		http.StatusUnauthorized,
		gen.UnauthorizedAccessToken{
			ErrorCode: gen.TOKENINVALID,
			Message:   "invalid token",
		},
	)
}
