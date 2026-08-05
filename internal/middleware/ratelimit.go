package middleware

import (
	"net/http"

	"github.com/IshanHunt77/llm-gateway/internal/ratelimit"
)

func RateLimit(b *ratelimit.Bucket) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			got := b.Allow()
			if got {
				next.ServeHTTP(w,r)
			}else{
				http.Error(w,"429 Too Many Request",http.StatusTooManyRequests)
				return
			}
		})
	}
}