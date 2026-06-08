# Last 20 Commits — Status Report

## Context
Review of the most recent 20 commits on `main` (2026-06-04 → 2026-06-08). The
work forms one coherent push: harden the embedder RAG chat (retrieval grounding
+ intent gating), add model-provider selection/config UX, give operators debug
visibility, and trim page-load cost. Reviewing to summarize what shipped and
flag the loose ends worth doing next.

## What's been done

### 1. Retrieval quality / RAG grounding (core thread)
- **#246 Add chat intent guard** (`d49ee13`) — classify query intent before retrieving.
- **#252 Gate visual retrieval by query intent** (`220988b`) — skip visual/image retrieval unless the query asks for it (`vector_retrieve.go`).
- **#254 Guard chat against weak retrieval context** (`8489b7d`) — lexical grounding filter; canned fallback when retrieved chunks don't lexically match the query, LLM not called.
- **`efc9dbf` Improve lexical search** — fixed two regressions from #254: pure-stopword/short queries no longer force the fallback, and explicitly record-scoped queries skip grounding. Removed dead `hasLexicalGrounding`; `groundedChunks(terms, chunks)` computes terms once. Added 2 tests.

### 2. Model provider / config UX
- **#248 Show chat model provider status** (`34e4dd8`) — provider/model badge in `ChatPage.tsx`.
- **#249 Add prompt model selector** (`4000a0d`) — per-request model override (`modelConfig.ts`, `ChatPage.tsx`).
- **#250 Add model provider connection test** (`ccba82e`) — backend test endpoint (`transport/models.go` + test) wired through Traefik and `ConfigPage.tsx`.

### 3. Debug / observability
- **#251 Add retrieval debug panel** (`025f4ae`) — `ChatDebug` events surface query, topK, ranked chunks + scores end-to-end (domain → postgres → service → transport → UI).

### 4. Performance
- **#253 Optimize dashboard and domain loading** (`db3dd12`) — new `ui/src/lib/dashboard.ts` data layer, slimmed `DashboardPage.tsx`.

### 5. UX clarity / docs / cleanup
- **#247 Clarify chat context counts** (`d2b8f54`).
- **`4c44340` Document UI setup and local development** (README + ui/README).
- **`08f2d04` Clean up chat test fixtures**.

## What's next (open threads from this work)

1. **Word-boundary grounding** — `groundedChunks` uses `strings.Contains`, so term `art` matches `start`/`cart`. Weak chunks slip the guard. Switch to token/word-boundary match. (`internal/embedder/service/chat.go`)
2. **Short codes dropped** — `meaningfulQueryTerms` drops tokens `<3` chars, losing `AI`, `Q3`, `H2`, `ML`. Contradicts `ragSystemPrompt` promise to handle short keywords/codes. Allow short alphanumeric codes.
3. **Score-based grounding over lexical** — a reranker already runs; consider a score threshold instead of (or alongside) lexical overlap — more robust than substring matching.
4. **General-knowledge fallback toggle** — #254 removed the old "answer from general knowledge when retrieval is empty" path. Consider a config flag so deployments can re-enable it.

## Findings — bugs in shipped code (not future work)

### F1 (HIGH / security) — SSRF on the real chat path
The `test-connection` endpoint validates `base_url` against an allowlist, but the
actual chat path does not.
- `internal/embedder/api/transport/chat.go:25` — `Model *domain.ModelConfig` (incl. `BaseURL`, `APIKey`, `Provider`) decoded straight from the request body.
- `internal/embedder/service/chat.go:78-89` — `modelCfg.BaseURL` passed verbatim into `s.factory(llm.Config{BaseURL: ...})`, **no `allowedExternalModelURL` check**.
- `cmd/embedder/main.go:332` — factory wired in prod; builds an openai/ollama client and POSTs to that URL.
- Impact: any authenticated tenant user sets `model.base_url` to `http://169.254.169.254/...` (cloud metadata), `http://localhost:<port>`, or any internal service → server-side request. Same input the test endpoint guards.
- Fix: factor `allowedExternalModelURL` out of `transport/models.go` and call it in `chatService.Chat` (or in the factory) before building the per-request client; reject non-configured ollama hosts too.

### F2 (MEDIUM) — substring-match false positives (recurring anti-pattern)
`hasVisualIntent` (`internal/embedder/service/vector_retrieve.go`, commit `220988b`) uses `strings.Contains`:
- `"imagine the rollout"` contains `"image"` → visual retrieval fires wrongly.
- `"scandal report"` contains `"scan"` → fires.
Same `strings.Contains` anti-pattern as the lexical grounding (`art`⊂`start`, future-work #1). Now in two files — extract a shared word-boundary helper rather than fixing twice.

### F3 (MINOR) — dropped error is undebuggable server-side
`contextError(message string, _ error)` (`internal/embedder/api/transport/models.go`) discards the wrapped error entirely. Fine for client-facing text, but nothing logs the real cause → connection failures undebuggable. `slog` the dropped err.

## Verification (for any next-step work)
- `go test ./internal/embedder/service/` — grounding/guard unit tests.
- `go test ./internal/embedder/...` — full embedder suite.
- Manual: ask a short-code query (`Q3 revenue`) and a record-scoped paraphrase; confirm LLM is called and grounded answer returns (not the canned refusal).
