package provider

import (
	"io"
	"net/http"
)

func Handler() http.Handler {
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from provider/mock.go")
	}

	// http.HandleFunc("/", h1)

	// log.Fatal(http.ListenAndServe(host, nil))
	return http.HandlerFunc(h1)
}
