package main

import (
	"log"
	"net/http"

	"github.com/IshanHunt77/llm-gateway/internal/provider"
	"github.com/IshanHunt77/llm-gateway/internal/proxy"
)

func main() {
	go func() {
		log.Fatal(http.ListenAndServe(":8080", provider.Handler()))
	}()//background 

	h, err := proxy.New("http://localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal((http.ListenAndServe(":9090", h)))
	
}
