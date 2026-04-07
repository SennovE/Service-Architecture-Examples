package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type logFormat struct {
	RequestID   string    `json:"request_id"`
	Method      string    `json:"method"`
	Endpoint    string    `json:"endpoint"`
	StatusCode  int       `json:"status_code"`
	DurationMS  int       `json:"duration_ms"`
	UserID      uuid.UUID `json:"user_id"`
	Timestamp   string    `json:"timestamp"`
	RequestBody any       `json:"request_body,omitempty"`
}

var sensitiveData = []string{"password", "refresh_token"}

func LogRequest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		info := logFormat{
			RequestID: ctx.GetString("X-Request-Id"),
			Method:    ctx.Request.Method,
			Endpoint:  ctx.Request.RequestURI,
			Timestamp: startTime.Format(time.RFC3339),
		}

		if shouldLogRequestBody(ctx.Request.Method) {
			if raw, ok := getBody(ctx); ok {
				info.RequestBody = maskPasswords(raw)
			}
		}

		ctx.Next()

		info.StatusCode = ctx.Writer.Status()
		info.DurationMS = int(time.Since(startTime).Milliseconds())
		if userID, ok := ctx.Get("userID"); ok {
			if id, ok := userID.(uuid.UUID); ok {
				info.UserID = id
			}
		}

		b, _ := json.Marshal(info)
		_, _ = os.Stdout.Write(append(b, '\n'))
	}
}

func shouldLogRequestBody(m string) bool {
	switch strings.ToUpper(m) {
	case "POST", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

func getBody(ctx *gin.Context) ([]byte, bool) {
	if ctx.Request.Body == nil {
		return nil, false
	}
	raw, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
		return nil, false
	}
	defer func() { ctx.Request.Body = io.NopCloser(bytes.NewBuffer(raw)) }()
	return raw, true
}

func maskPasswords(raw []byte) any {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return strings.TrimSpace(string(raw))
	}
	for _, k := range sensitiveData {
		if _, ok := m[k]; ok {
			m[k] = "*****"
		}
	}
	return m
}
