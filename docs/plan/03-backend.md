# Backend (Go)

### 3.1 Layout

```
backend/
  cmd/server/main.go            # config, wiring, graceful shutdown
  internal/
    quiz/        data.go        # Question/Option types, Parse+Validate(data.json)
                 data_test.go
    session/     model.go       # Session, Timer, Status, Tally()
                 hub.go         # actor loop, commands, timer
                 hub_test.go
                 views.go       # AdminView / ClientView snapshots
    store/       store.go       # interface; memory.go (TTL eviction); redis.go (later)
    httpapi/     server.go      # router, CORS, POST /sessions, GET /sessions/{id}/exists
                 ws.go          # upgrade, read/write pumps, ping/pong
  go.mod
```

Dependencies: stdlib `net/http` (Go 1.22 routing), `github.com/coder/websocket` (or gorilla), `crypto/rand`. No framework.

### 3.2 HTTP API

| Endpoint | Purpose |
|---|---|
| `POST /sessions` (multipart `file`) | Validate `data.json`; create session; return `{ sessionId, hostKey, questionCount, questions:[{id,text}], hostLink, shareLink }`. On failure `422 { errors:[{path:"questions[2].options", message:"must have 2–6 items"}] }` — **all** errors, not just the first. |
| `POST /sessions/preview` (multipart `file`) | Same validation, creates nothing — powers the "preview imported question list" step (Build plan #1). |
| `GET /sessions/{id}` | `{ exists, ended }` so the join page can show "session not found" before opening a socket. |
| `GET /healthz` | Liveness. |
| `GET /ws/{id}` | WebSocket; `?key=` ⇒ admin, `?clientId=` ⇒ returning client. |

### 3.3 Validation rules (`quiz.Validate`)

Mirrors the data contract: `questions` is a non-empty array; each `id` non-empty string and unique; `text` non-empty; `options` 2–6 items each with non-empty `key` (unique within question) and `text`; `points` number ≥ 0 if present; `defaultDurationSec` ∈ {10,15,30,60} if present. Errors carry a JSON-path-style field name. Unknown fields are ignored. Enforce body size limit before parsing.

### 3.4 WebSocket protocol (JSON text frames)

Client → server:

| `type` | Role | Payload | Accepted when |
|---|---|---|---|
| `join` | client | `{ name, clientId? }` | any time |
| `answer` | client | `{ questionId, key }` | `status=running`, `questionId` is current, `key` is a valid option |
| `start` | admin | `{ durationSec }` ∈ {10,15,30,60} | `status=idle` |
| `finish` | admin | – | `status=running` |
| `next` | admin | – | `status=results` and more questions remain |
| `ping` | both | – | any |

Server → client:

| `type` | Payload |
|---|---|
| `state` | role-specific snapshot (below); sent on connect and after every mutation |
| `joined` | `{ clientId }` |
| `error` | `{ code, message }` (e.g. `locked`, `bad_option`, `forbidden`) — non-fatal |

Snapshot fields: `sessionId, serverNow, phase, currentQuestionIndex, questionCount, question {id,text,options[]}` (only when `running`/`finished`/`results`, hidden while `idle` for clients), `timer {durationSec, startedAt, status}`, `answeredCount, clientCount`, `myAnswer` (client only), `tally {A:3,B:5,…}` (only once finished), and for admin the per-question status list `[{id, text, status: not_started|running|finished}]`.

### 3.5 State machine

`idle → running → finished → results → (next) idle …` per question. `finished → results` is immediate and automatic (same command), so on the wire clients observe `running → results`; `finished` is kept in the model for the status chip. After the last question's results, `phase = complete`.

Invariants enforced in the hub and covered by tests: one answer slot per client per question (resubmit overwrites while running); no writes after status leaves `running`; `tally` is computed from `answers` on every read (never stored); `Expire` after manual `Finish` is a no-op.

### 3.6 Operational concerns

CORS allow-list from env; WebSocket origin check; per-connection read limit (4 KB) and rate limit (e.g. 10 msgs/s); write deadline + ping/pong (30 s) to reap dead sockets (these also drive `clientCount` accuracy — track `connected` separately from `joined`); background janitor evicting idle sessions; graceful shutdown closes sockets with a restart code so clients auto-reconnect. Config via env: `PORT`, `PUBLIC_BASE_URL`, `ALLOWED_ORIGINS`, `MAX_SESSIONS`, `SESSION_TTL`.
