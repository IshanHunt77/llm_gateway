package retry

import (
	"math/rand"
	"net/http"
	"time"
)


type RetryTransport struct {
	transport http.RoundTripper
	maxAttempts int
	baseDelay time.Duration
}

func New(trspt http.RoundTripper,maxAtt int,basDel time.Duration) *RetryTransport{
	return &RetryTransport{transport: trspt,maxAttempts: maxAtt,baseDelay: basDel}
}

func (rt *RetryTransport) RoundTrip(req *http.Request)(*http.Response,error){
	var resp *http.Response
	var err error
	for i:=0;i<rt.maxAttempts;i++ {
		resp,err = rt.transport.RoundTrip(req)
		if err==nil && resp.StatusCode < 500 {
			return resp,err
		}else{
			delay := rt.baseDelay * time.Duration(1<<i)
			jittDelay := rand.Float64()*float64(delay)
			time.Sleep(time.Duration(jittDelay))
		}
	}
	return resp,err
}