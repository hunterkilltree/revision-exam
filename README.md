# Live Quiz Sessions

Go backend (WebSocket hub, in-memory store) + Next.js frontend. See `requirement-revision-exam.md` and `docs/plan/`.

```
make dev     # backend :8080, frontend :3000
make test    # go test -race + vitest
```

Backend env: `PORT` (8080), `PUBLIC_BASE_URL` (http://localhost:3000), `ALLOWED_ORIGINS` (defaults to the base URL), `MAX_SESSIONS` (20), `SESSION_TTL` (2h).
Frontend env: `NEXT_PUBLIC_API_URL` (http://localhost:8080).

End-to-end smoke test (needs both apps running and Playwright): `NODE_PATH=$(npm root -g) node e2e/smoke.mjs`.

Host link is `/admin/{id}?key={hostKey}`; the key is the only credential (no password).
