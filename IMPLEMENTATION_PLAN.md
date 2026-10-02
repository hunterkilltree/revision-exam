# Implementation Plan — Live Quiz Sessions

Source documents: `requirement-revision-exam.md` (requirements, data contract, decisions) and `prototype-ui.html` (11 screens: 3 admin at 1440×900, 4 client mobile at 390×844, 4 client desktop at 1440×900).

Stack (fixed by the requirements): **Go backend + WebSocket**, **Next.js frontend**, in-memory store (Redis optional later).

---

## 1. Decisions that need pinning down before coding

The requirements leave a few gaps. Proposed defaults are in bold; flag any you disagree with.

| # | Gap | Proposed default |
|---|-----|------------------|
| 1 | "Holding the host link is enough" — but the share link carries the same `sessionId`, so a session-id-only host URL is guessable by any client. | **Host link = `/admin/{sessionId}?key={hostKey}`** where `hostKey` is a random 128-bit secret returned only by `POST /sessions`. Share link = `/join/{sessionId}` (no secret). Still no password. |
| 2 | "One session at a time per admin" with no accounts. | **Enforce per browser**: the setup screen stores the active host link in `localStorage`; if one exists and is live, it offers "Resume" / "End and start new". Server also caps total concurrent sessions (config, default 20) and expires idle ones. |
| 3 | Who computes the clock? | **Server is authoritative.** It stores `startedAt` and `durationSec`, expires the question itself, and sends `serverNow` with every snapshot so clients correct for clock skew. Clients only render. |
| 4 | Are live counts visible to clients while running? | **No.** Clients only get `answeredCount / clientCount`. Per-option counts are sent only after the question finishes (prevents herding; matches "results appear after lock"). |
| 5 | Question order / navigation. | **Linear, play order = array order.** Admin can click earlier question in the list only to *view* its status/results, not replay it. Next question is enabled only from `results`. |
| 6 | Client identity / reconnect. | **`clientId` (UUID) generated server-side on join, stored in `localStorage`**; reconnect with the same id resumes the same slot and answers. Display names need not be unique (id is the key). |
| 7 | Join mid-session / mid-question. | **Allowed.** Client receives the current snapshot; if `running` they can answer with the remaining time. |
| 8 | Session end. | **After the last question's results, admin sees a "Quiz complete" state**; session is kept 2 h for late reconnects, then evicted. No export (per decisions). |
| 9 | Limits. | **File ≤ 1 MB, ≤ 200 questions, name ≤ 40 chars, ≤ 500 clients/session.** |
| 10 | `points` | Validated and stored, never used (as specified). |

---

## 2. Architecture

```
 Browser (admin)  ──HTTP POST /sessions──▶ ┐
 Browser (admin)  ──WS /ws/{id}?key=…───▶  │  Go service
 Browser (client) ──WS /ws/{id}─────────▶  │   ├─ http handlers (validate, create)
                                            │   ├─ Hub per session  (single goroutine = only writer)
 Next.js (SSR shell, no session state) ◀─── │   ├─ Timer (time.AfterFunc inside the hub)
                                            │   └─ Store (in-memory map; interface for Redis)
```

**Single-writer hub per session.** Each session owns one goroutine with an inbound `chan Command`. Every mutation (join, answer, start, finish, next, timer expiry) is a command processed serially, so there are no locks around session state and no races between "Finish early" and timer expiry. After each mutation the hub broadcasts a fresh state view to every connection.

**Why not broadcast diffs:** the state is tiny (≤ 500 clients, ≤ 6 options). Sending a full role-specific snapshot on every change makes reconnect trivial ("reconnect = receive snapshot") and removes diff-ordering bugs. Optimise only if measurement says so.

**Timer.** `Start` sets `startedAt = now`, `status = running`, and schedules `time.AfterFunc(duration)` that posts a `Expire{questionIndex, epoch}` command. The command is ignored if the question was already finished (epoch mismatch). No per-second ticks are pushed; clients count down locally from `startedAt + durationSec − serverNow`. A resync snapshot is sent on any state change and every ~10 s as a heartbeat.

**Role separation.** Admin connections must present `key` matching the session's `hostKey` (constant-time compare). Role is fixed at upgrade time; commands are authorised per role in the hub (clients may only `join`/`answer`).

---

## 3. Backend (Go)

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

---

## 4. Frontend (Next.js, App Router, TypeScript)

### 4.1 Layout

```
frontend/
  app/
    layout.tsx                  # fonts, global tokens
    page.tsx                    # Admin · Session setup (upload, preview, create, show links)
    admin/[sessionId]/page.tsx  # Admin console (reads ?key=)
    join/[sessionId]/page.tsx   # Client flow
  components/
    admin/  QuestionList, QuestionPanel, DurationSelector, ResultsPanel, LinkCopy, UploadDropzone
    client/ JoinForm, WaitingView, AnsweringView, LockedResultsView
    shared/ Countdown, BarChart, StatusChip, ConnectionBanner
  lib/
    socket.ts      # useSession(): connect, auto-reconnect w/ backoff, exposes state + send
    clock.ts       # useCountdown(startedAt, durationSec, serverNow) with skew offset
    types.ts       # mirrors the Go snapshot/command types
  styles/tokens.css
```

No session state is held in the frontend beyond what the last `state` message says (per the requirements): a refresh simply reconnects and receives the snapshot. The only `localStorage` use is `clientId`, display name, and the admin's active host link.

### 4.2 Screens ↔ prototype

| Prototype board | Route / component | Notes |
|---|---|---|
| Admin · Session setup | `/` | Upload → validate → show imported count + question list preview → "Create session" → host & share link with copy buttons. Field-level errors listed inline. |
| Admin · Question running | `/admin/[id]` `QuestionPanel` | Left list with status chips; duration selector (disabled unless `idle`); Start / Finish early; countdown; "N of M answered". |
| Admin · Results | same route, `ResultsPanel` | Same screen swaps to bar chart + **Next question** (becomes "Quiz complete" on the last). |
| Client · Join (mobile + desktop) | `/join/[id]` `JoinForm` | Name input; error if session missing/ended. |
| Client · Waiting | `WaitingView` | "Waiting for the next question", question n of N. |
| Client · Answering | `AnsweringView` | Options as large tap targets, live countdown, selected option highlighted and changeable until lock. |
| Client · Locked + results | `LockedResultsView` | Options disabled, own pick marked, same bar chart as admin. |

Mobile (390 px) and desktop (1440 px) client variants are **one responsive component set**, not two code paths.

### 4.3 Design tokens (extracted from the prototype)

Palette: ink `#16161A`, secondary text `#3A3A44` / `#5B5B66` / `#75757F`, accent blue `#2F3BE0` (dark `#1E27A8`, tint `#E6E8FD`), warning/timer orange `#C2410C`, surfaces `#FFFFFF`, `#FAF9F6`, `#F4F3EF`, `#EFEDE6`, `#E7E6E0`, borders `#DDDBD3` / `#CFCDC4`. Fonts: **Bricolage Grotesque** (display), **IBM Plex Sans** (body), **IBM Plex Mono** (timer/numbers) via `next/font/google`. Put these in `tokens.css` as CSS variables; read exact spacing/radii from the prototype boards (the bundle extracts cleanly, one HTML per board) while building each component.

### 4.4 Behaviours worth specifying up front

- **Countdown** derives from `startedAt + durationSec − (serverNow + localElapsedSinceMessage)`; clamps at 0; shows the orange state in the last 5 s. When it hits 0 the UI waits for the server's `results` state rather than self-transitioning (avoids flicker if the server is a hair late); locks inputs locally at 0 to match the "taps after lock are rejected client-side" requirement.
- **Optimistic answer**: highlight immediately, reconcile with `myAnswer` from the next snapshot; on `locked` error revert and show a toast.
- **Connection banner** ("Reconnecting…") with exponential backoff 0.5 s → 5 s; on reconnect re-send `join` with stored `clientId`.
- **Bar chart**: pure SVG/CSS (no chart library needed for ≤ 6 bars), shows count and %, labelled by option key + text, highlights the viewer's pick on client screens, handles all-zero (nobody answered) with an explicit "No answers" message.
- Accessibility: options are buttons with `aria-pressed`, countdown has an `aria-live="polite"` announcement at 10/5 s, charts have a text table fallback, colour is never the only state indicator.

---

## 5. Delivery phases

Maps to the 8 steps in the requirements' build plan, regrouped so each phase ends in something demonstrable.

| Phase | Scope (requirements step) | Exit criteria |
|---|---|---|
| **0 — Scaffolding** | Repo layout, `go.mod`, Next.js app, tokens/fonts, Makefile / `docker-compose` for local run, CI (go vet+test, eslint, tsc, next build) | `make dev` runs both apps; CI green |
| **1 — Data loader** (#1) | `quiz.Validate` + table tests; `/sessions/preview`; upload UI with field-level errors and preview | Malformed fixtures each produce the named field error; valid file previews |
| **2 — Sessions & links** (#2) | `POST /sessions`, store, `hostKey`, link display, `GET /sessions/{id}`, resume-existing-session logic | Links open correct routes; wrong `key` rejected |
| **3 — Realtime skeleton + Admin shell** (#3) | WS upgrade, hub actor, snapshot broadcast, admin question list with status chips | Admin console renders live state from socket; refresh restores it |
| **4 — Client join & waiting** (#4) | `join`, `clientId`, join page, waiting view, `clientCount` on admin | Two browser tabs join; admin sees count change |
| **5 — Timer engine** (#5) | Duration selector, Start, expiry, Finish early, skew-corrected countdown | Admin and 3 clients show same clock within ~0.5 s; Finish vs. expiry race test passes |
| **6 — Answer capture** (#6) | `answer` command, overwrite-while-running, lock, "N of M answered" | Answers after lock rejected server-side and ignored client-side |
| **7 — Tally & chart** (#7) | Computed tally on finish, shared `BarChart`, results panel + locked view, Next question, quiz-complete state | Full multi-question run end-to-end with correct counts |
| **8 — Polish & edge cases** (#8) | Reconnect, join mid-question, admin reload mid-run, zero-answer question, janitor/TTL, rate limits, a11y pass, mobile QA | Edge-case checklist (§6) passes |
| **9 — Ship** | Dockerfiles, env docs, deploy target, README, load check (~200 clients) | Deployed instance used for a dry-run quiz |

Phases 1–2 are backend-first and unblock the frontend; from phase 3 on, build each feature backend-then-UI in one vertical slice.

---

## 6. Test strategy

**Go (`go test -race ./...`)**
- `quiz`: table tests — valid file, missing `id`, duplicate ids, 1 option, 7 options, duplicate option keys, bad duration, non-JSON, oversize, multiple simultaneous errors.
- `session` hub: drive the actor with commands and a fake clock — state-machine transitions, illegal transitions rejected, answer overwrite, answer after finish rejected, expiry-after-finish no-op, tally correctness, no answers ⇒ all-zero tally, snapshot hides tally/question from clients at the right times.
- `httpapi`: `httptest` + WS client for auth (bad key → rejected), reconnect with same `clientId` keeps answer, malformed frames don't crash the hub.
- Concurrency: 200 simulated clients answering at once under `-race`.

**Frontend**
- Unit: `useCountdown` skew math; reducer that applies `state` messages; `BarChart` zero/rounding cases (Vitest + Testing Library).
- E2E (Playwright, Chromium is available): one admin context + three client contexts run a 2-question quiz; assertions on countdown, answered count, final bars; plus scenarios for client reload mid-question, admin reload mid-run, and late joiner.

**Edge-case checklist (phase 8)**: dropped Wi-Fi mid-question and answer changed after reconnect; join during `running`; admin reload during `running` and `results`; nobody answers; admin clicks Finish at the same instant as expiry; duplicate display names; same `clientId` in two tabs (last connection wins); question set with 1 question; 60 s timer on a backgrounded mobile tab (clock must be correct on return).

---

## 7. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Clock drift or backgrounded tabs freeze timers | Server-authoritative time + `serverNow` offset; recompute from timestamps on every render/visibility change, never decrement a counter. |
| Guessable host control | Separate high-entropy `hostKey`; admin role fixed at WS upgrade. |
| In-memory store loses sessions on restart | Accepted for v1 (single instance); store interface isolates a Redis implementation. Document that deploys end live quizzes — avoid deploying during an event. |
| Single instance limits scale | One hub goroutine per session is cheap; target ≈ 500 clients/session, tens of sessions. Horizontal scale would need sticky routing or Redis pub/sub — out of scope. |
| Prototype/requirement drift (e.g. timer option order "15/10/30/60") | Present options in ascending order (10/15/30/60) unless the prototype shows otherwise — confirm in phase 5 against the boards. |
| Abuse of public join link | Name length cap, message rate limit, per-session client cap. |

---

## 8. Out of scope (per requirements)

Admin accounts/passwords, results export or persistence, scoring/leaderboard (`points` is carried only), multiple concurrent sessions per admin, replaying earlier questions.

## 9. Open questions for you

1. OK with the **hostKey-in-URL** approach (gap #1)? It keeps "no password" but avoids a guessable console.
2. Where will this be deployed (single VM / Fly / Cloud Run / etc.)? Affects phase 9 and whether Redis is worth adding.
3. Monorepo with `backend/` and `frontend/` folders as above — acceptable?
