package main

import (
	"log"
	"net/http"
	"time"

	"github.com/IshanHunt77/llm-gateway/internal/cache"
	"github.com/IshanHunt77/llm-gateway/internal/config"
	"github.com/IshanHunt77/llm-gateway/internal/middleware"
	"github.com/IshanHunt77/llm-gateway/internal/provider"
	"github.com/IshanHunt77/llm-gateway/internal/proxy"
	"github.com/IshanHunt77/llm-gateway/internal/ratelimit"
	"github.com/IshanHunt77/llm-gateway/internal/retry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	target, err := cfg.DefaultProviderURL()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Fatal(http.ListenAndServe(":8080", provider.Handler()))
	}() //background

	h, err := proxy.New(target,retry.New(http.DefaultTransport,3,100*time.Millisecond))
	if err != nil {
		log.Fatal(err)
	}
	c := cache.New()
	b := ratelimit.New(cfg.RateLimit.Capacity, cfg.RateLimit.RefillRate)
	c.Upsert("/", "cached response!")

	log.Fatal(http.ListenAndServe(cfg.GatewayPort, middleware.ServerHeader(middleware.Logging(middleware.RateLimit(b)(middleware.Caching(c)(h))))))

}
