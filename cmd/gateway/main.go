package main

import (
	"log"
	"net/http"

	"github.com/IshanHunt77/llm-gateway/internal/config"
	"github.com/IshanHunt77/llm-gateway/internal/provider"
	"github.com/IshanHunt77/llm-gateway/internal/proxy"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	 log.Printf("%+v", cfg)
	target, err := cfg.DefaultProviderURL()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Fatal(http.ListenAndServe(":8080", provider.Handler()))
	}() //background

	h, err := proxy.New(target)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal((http.ListenAndServe(cfg.GatewayPort, h)))

}
