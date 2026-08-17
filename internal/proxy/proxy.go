package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

)

func New(target string,round http.RoundTripper) (http.Handler, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	revproxy :=  httputil.NewSingleHostReverseProxy(u)
	 revproxy.Transport = round
	return revproxy,nil
}
