package middleware

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/IshanHunt77/llm-gateway/internal/cache"
)


type wrapper struct {
	 http.ResponseWriter
	buff bytes.Buffer
}

func (rw *wrapper) Write(b []byte) (int,error) {
					rw.buff.Write(b)
					val,err:=rw.ResponseWriter.Write(b)
					return val,err
				}

func Caching(c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			key := r.URL.Path
			val,ok:=c.Read(key)
			if !ok {
				wrap := wrapper{ResponseWriter: w}
				next.ServeHTTP(&wrap,r)
				c.Upsert(key,wrap.buff.String())
				slog.Info("miss")
			}else{
				w.Write([]byte(val))
				slog.Info("Hit")
				return
				
			}
			
		})
	}
}