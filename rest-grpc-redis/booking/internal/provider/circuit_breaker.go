package provider

import (
	"booking/internal/gen/api"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type BreakerState string

var (
	Closed BreakerState = "closed"
	Open   BreakerState = "open"
	Half   BreakerState = "half"
)

type CircuitBreaker struct {
	sync.Mutex
	errs int
	state BreakerState
	timeout time.Duration
	lastErrorTime time.Time
}

func (cb *CircuitBreaker) Allow() bool {
	cb.Lock()
	defer cb.Unlock()
	if cb.state == Open {
		if time.Since(cb.lastErrorTime) > cb.timeout {
			cb.state = Half
			log.Println("Circuit breaker is HALF OPEN")
		} else {
			return false
		}
	}
	return true
}

func CircuitBreakerMiddleware(timeout time.Duration) gin.HandlerFunc {
	cb := &CircuitBreaker{
		state: Closed,
		timeout: timeout,
	}

	return func(ctx *gin.Context) {
		if !cb.Allow() {
			ctx.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				api.ErrorResponse{
					ErrorCode: "UNAVAILABLE",
					Message:   "Service Unavailable",
				},
			)
			return
		}
		ctx.Next()
		cb.Lock()
		defer cb.Unlock()
		if _, ok := ctx.Get("GRPC_ERROR"); ok {
			cb.errs++
			if cb.errs >= 3 {
				cb.state = Open
				cb.lastErrorTime = time.Now()
				log.Println("Circuit breaker is OPEN")
			}
		} else {
			cb.errs = 0
			if cb.state == Half {
				cb.state = Closed
				log.Println("Circuit breaker is CLOSE")
			} 
		}
	}
}
