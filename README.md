-
1️⃣ Flow diagram — Cache middleware kaise kaam karega

        Client request (POST, body = {"prompt":"capital of India?"})
                       │
                       ▼
        ┌─────────────────────────────┐
        │   Logging middleware          │  (Phase 2 — already hai)
        └─────────────────────────────┘
                       │
                       ▼
   ┌───────────────────────────────────────────────┐
   │        CACHE MIDDLEWARE  ← ye hum banayenge       │
   │                                                   │
   │  Step 1: body padho                               │
   │  Step 2: body ka sha256 hash nikaalo = KEY        │
   │          "capital of India?" → "a3f9c2..." (64char)│
   │  Step 3: map[KEY] exist karta hai?                │
   │                                                   │
   │     ┌── HAAN (HIT) ──────────────────────────  ┐   │
   │     │  stored response seedha client ko likho  │   │
   │     │  next.ServeHTTP CALL HI MAT KARO         │   │───► return ✅
   │     │  (proxy/provider ko chhua bhi nahi)      │   │     (fast, free)
   │     └──────────────────────────────────────────┘   │
   │                                                    │
   │     └── NAHI (MISS) ─────────────────────────┐     │
   │        3a: body WAPAS bharo (padh liya tha)    │    │
   │        3b: next.ServeHTTP(w, r) ────────────── ┼────┼──► proxy → provider
   │        3c: jawab CAPTURE karo (RW wrap)        │    │      (asli LLM call)
   │        3d: map[KEY] = jawab  (write-lock)      │◄───┼──── response
   │        3e: client ko jawab do                  │    │
   │        └───────────────────────────────────────┘    │
   └───────────────────────────────────────────────┘