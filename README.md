# LLM Gateway

A self-hosted, **standard-library-first** gateway written in Go that sits between your applications and LLM providers (Groq, Ollama, …). It centralizes the cross-cutting concerns of every LLM call — caching, routing, reliability, security, and observability — so apps don't have to reimplement them.

> Think **LiteLLM / Portkey / Cloudflare AI Gateway**, built from scratch in Go to learn backend infrastructure deeply.

---

## Why

Calling an LLM directly from every app breaks in the same 6 places, everywhere:

- **Scattered keys** — every app holds its own API key; rotation/leaks become a nightmare.
- **Repeated cost & latency** — the same prompt is paid for and waited on again and again.
- **No failover** — if a provider goes down, the app goes down.
- **No safety brake** — one bug can blow up the bill with unlimited calls.
- **No visibility** — no metrics on cost, latency, or token usage.
- **Tight coupling** — switching providers means rewriting app code.

**Core idea:** when a problem repeats everywhere, don't solve it everywhere — centralize it in one place. That place is the **Gateway**.

---

## Architecture

```
                        ┌───────────────────────── Gateway ──────────────────────────┐
   Client               │                                                            │
  "generate"  ─────────▶│  Logging ──▶ Cache check ──▶ Router ──▶ Reverse Proxy  ─────┼──▶  Provider
                        │     │            │  │                                       │      (Groq /
                        │     │       HIT  │  │ MISS                                  │       Ollama /
   Response  ◀──────────┼─────┴────────────┘  └───────────────────────────────◀──────┼──────  Mock)
                        │              (served from cache, provider untouched)        │
                        └────────────────────────────────────────────────────────────┘
```

- **Logging** — records method, path, and latency for every request (`log/slog`).
- **Cache check** — on a cache **HIT**, the stored response is returned immediately and the provider is never called; on a **MISS**, the request flows on and the response is stored on the way back.
- **Router** — picks the target provider from configuration (no code change needed to switch).
- **Reverse Proxy** — forwards the request to the chosen provider and streams the response back.
- **Provider** — the actual LLM backend (a mock provider is used for deterministic local testing).

---

## Features

**Implemented**

- **Reverse-proxy gateway** exposing a single entry point, built entirely on the Go standard library (`net/http`, `net/http/httputil`) — zero external dependencies.
- **Config-driven multi-provider routing** — providers, URLs, and ports live in `config.json`; API keys are injected from environment variables (never committed).
- **Composable middleware chain** — cross-cutting concerns are `func(http.Handler) http.Handler` layers, chained in order.
- **Structured request logging** via `log/slog`.
- **Thread-safe in-memory cache** — a `sync.RWMutex`-guarded store with SHA-256 keys, ready to short-circuit duplicate calls.

**On the roadmap**

- Token-bucket **rate limiting**
- **Retries** with exponential backoff + jitter
- **Circuit breaker** (per provider) and **fallback** to a healthy provider
- **SSE streaming** (token-by-token passthrough)
- **Observability** — Prometheus metrics (latency, tokens, cost, cache hit-rate)
- **Load testing** to publish real throughput/latency numbers

---

## Project Structure

```
llm-gateway/
├── cmd/gateway/main.go        # entry point: wiring + server startup
├── internal/
│   ├── config/                # load providers/keys/ports; env-based secrets
│   ├── provider/              # LLM backends (mock; Groq/Ollama planned)
│   ├── proxy/                 # reverse-proxy handler
│   ├── middleware/            # logging, caching, (rate-limit/auth planned)
│   └── cache/                 # thread-safe in-memory store (RWMutex + map)
├── config.json                # runtime configuration (no secrets)
└── go.mod
```

- **`cmd/`** holds only wiring; **`internal/`** holds the real logic (and is protected from external imports by Go's `internal` rule).
- Each `internal` package has a **single responsibility** — a clean separation of concerns.

---

## Getting Started

**Prerequisites:** Go 1.24+

```bash
# 1. Clone
git clone https://github.com/IshanHunt77/llm_gateway.git
cd llm_gateway

# 2. (Optional) provide a provider API key via environment
export GROQ_API_KEY=your_key_here     # PowerShell: $env:GROQ_API_KEY="your_key_here"

# 3. Run (starts the mock provider + the gateway)
go run ./cmd/gateway

# 4. In another terminal, hit the gateway
curl http://localhost:9090/
```

### Configuration (`config.json`)

```json
{
  "gateway_port": ":9090",
  "default_provider": "mock_provider",
  "providers": [
    { "name": "mock_provider", "base_url": "http://localhost:8080" }
  ]
}
```

- Change the port or add/switch providers here — **no code changes required**.
- API keys are read from environment variables named `<PROVIDER_NAME>_API_KEY` (uppercased), never stored in this file.

---

## Testing

```bash
go test ./...            # run all tests
go test -race ./...      # run with the data-race detector (requires a 64-bit C compiler)
```

The cache package ships with concurrency tests that exercise the `sync.RWMutex`-guarded store under parallel writes.

---

## Roadmap (build phases)

| Phase | Focus | Status |
|-------|-------|--------|
| 0 | Passthrough reverse proxy | ✅ Done |
| 1 | Config + multi-provider routing + env secrets | ✅ Done |
| 2 | Middleware chain (logging) | ✅ Done |
| 3 | Caching (SHA-256 key, RWMutex store) | 🔨 In progress |
| 4 | Rate limiting (token bucket) | ⏳ Planned |
| 5 | Retries + backoff with jitter | ⏳ Planned |
| 6 | Circuit breaker + fallback | ⏳ Planned |
| 7 | SSE streaming | ⏳ Planned |
| 8 | Observability (Prometheus) | ⏳ Planned |
| 9 | Load testing | ⏳ Planned |

---

## Tech Stack

**Go** · `net/http` · `net/http/httputil` (reverse proxy) · goroutines · `sync.RWMutex` · `crypto/sha256` · `encoding/json` · `log/slog` · REST
