package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IshanHunt77/llm-gateway/internal/ratelimit"
)


func TestRateLimit(t *testing.T){
	b:= ratelimit.New(2,1)
	dummyHandler:= http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.WriteHeader(http.StatusOK)
	})
	handler := RateLimit(b)(dummyHandler)

	expected := []int{200,200,429}

	for i,want := range expected {
		req:=httptest.NewRequest(http.MethodGet,"/",nil)
		rec:=httptest.NewRecorder()
		handler.ServeHTTP(rec,req)
		got := rec.Code
		if got != want {
			t.Errorf("call %d: got %v, want %v", i+1, got, want)
		}
	}
}