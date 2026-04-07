package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMIddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rID := ctx.GetHeader("X-Request-Id")
		if rID == "" {
			rID = uuid.NewString()
		}
		ctx.Set("X-Request-Id", rID)
		ctx.Writer.Header().Set("X-Request-Id", rID)
		ctx.Next()
	}
}
