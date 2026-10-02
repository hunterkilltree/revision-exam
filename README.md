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

## Docker Hub → Render

```
DOCKERHUB_USER=me API_URL=https://quiz-api.onrender.com ./scripts/docker-publish.sh [tag]
```

Builds `linux/amd64` images for `backend/` and `frontend/`, pushes them to Docker Hub and prints the Render settings.
Deploy the backend first to learn its URL, then build the frontend with it as `API_URL` (it is inlined at build time).
Set the backend's `PUBLIC_BASE_URL` and `ALLOWED_ORIGINS` to the frontend's Render URL. Use `TARGET=backend|frontend` to push one image.
