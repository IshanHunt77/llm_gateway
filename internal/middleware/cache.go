package middleware

import (
	"log/slog"
	"net/http"

	"github.com/IshanHunt77/llm-gateway/internal/cache"
)

func Caching(c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			key := r.URL.Path
			_,ok:=c.Read(key)
			if !ok {
				slog.Info("miss")
			}else{
				slog.Info("hit")	
			}
			
			next.ServeHTTP(w,r)
		})
	}
}