### Decisions
- 2026-06-03 — Rate limiting per API key (OpenSpec add-rate-limiting, inv-7) — incident: 40k invoices took the PDF workers down.
- 2026-06-06 — Logging upgrade (inv-9) waits until rate limiting ships — it touches every handler.

### Pitfalls
- The API key is in the `X-Api-Key` header, not `Authorization`.
