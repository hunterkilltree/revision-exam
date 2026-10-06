# Live Quiz Sessions

A project plan for a two-role live quiz app: an admin creates a session from an uploaded data.json, shares a join link, runs each question on an admin-chosen timer (15s / 10s / 30s / 1 min, or stopped early), and clients answer live before a results chart appears on that same question screen.

## Roles & flow

&#91;embedded content: session flow · 7 steps, 1 loop\]

Each question repeats steps 3 through 7 until every question in the set has run.

## Data contract

The admin console loads one `data.json` file per session; nothing about the question set is hardcoded, so swapping the file changes the quiz.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `questions` | array | Yes | One entry per question; array order is play order. |
| `questions[].id` | string | Yes | Stable unique id — the key used for answers and tallies. |
| `questions[].text` | string | Yes | The question prompt shown to admin and clients. |
| `questions[].options` | array | Yes | 2–6 choices, each `{ "key": "A", "text": "..." }`. |
| `questions[].correctKey` | string | No | Key of the correct option; must match one of `options[].key`. Shown (highlighted) on the results screen for admin and clients once the question is locked, never while it is running. |
| `questions[].points` | number | No | Carried through for later scoring; unused by the live-tally feature itself. |
| `questions[].defaultDurationSec` | number | No | Pre-selects a timer option in the admin console (15/10/30/60); admin can still override before starting. |

Minimal example:

```json
{
  "questions": [
    {
      "id": "q1",
      "text": "Which option best describes X?",
      "options": [
        { "key": "A", "text": "Option A" },
        { "key": "B", "text": "Option B" },
        { "key": "C", "text": "Option C" }
      ],
      "points": 10,
      "defaultDurationSec": 30
    }
  ]
}
```

An upload that fails this shape (missing `id`, fewer than 2 options, duplicate ids) is rejected at import time with the specific field named, rather than failing partway through a live session.

## Question lifecycle

&#91;embedded content: question states · 4 states, 1 loop\]

Idle and Running are the only two states a client can act on; Finished and Results happen automatically, in order, with no admin input.

## Session runtime model

Everything that changes while a session is live sits in one session record, keyed by a session id; the admin console and every client view read the same record.

| Field | Type | Who writes it | Purpose |
| --- | --- | --- | --- |
| `sessionId` | string | created at session start | Identifies the host link and the share link. |
| `dataSource` | reference to the uploaded `data.json` | admin, at creation | Which question set this session plays. |
| `currentQuestionIndex` | integer | admin | Which question is on screen right now. |
| `timer.durationSec` | 15 \| 10 \| 30 \| 60 | admin | Chosen just before each question starts. |
| `timer.startedAt` | timestamp or null | admin (Start) | Null while idle; set the moment the countdown begins. |
| `timer.status` | `idle` \| `running` \| `finished` | admin (Start / Finish) or auto on expiry | Drives what clients are allowed to do right now. |
| `clients` | map of `clientId → displayName` | client, on join | Who has joined via the share link. |
| `answers` | map of `questionId → { clientId: optionKey }` | client, once per question while `timer.status = running` | Raw submissions — one slot per client per question; a resubmission before lock overwrites the previous one. |
| `tally` | derived, not stored | computed on read from `answers` | Per-option counts, drawn as the results chart. |

`tally` is never written directly — it is recomputed from `answers` whenever a question's results are shown, so admin and clients can never see a stale count.

## Screens

### Admin console

- Session setup: upload a `data.json`, see the imported question count, and get back two links — a host link (opens the console) and a share link (what clients use to join).
- Question list: every question down the left, the current one highlighted, a status chip per question (not started / running / finished).
- Live question panel: question text and options, a duration selector (15s, 10s, 30s, 1 min) enabled only while idle, a Start button that locks in the chosen duration and begins the countdown, a Finish early button that ends the countdown immediately, and a live “N of M answered” count while the timer runs.
- Results panel: once a question finishes (by timer or Finish), the same question screen switches to a bar chart of how many clients picked each option, plus a Next question button.

### Client view

- Join screen: open the share link, enter a display name, wait for the admin to start the session.
- Waiting state: between questions, a plain “waiting for the next question” screen with the question number counter.
- Answering state: once the admin starts the timer, the question and its options appear with a live countdown; tapping an option submits it immediately (one submission per question, changeable only while the timer is still running).
- Locked state: after Finish or timeout, options disable and the same results chart the admin sees appears, so clients see how the group answered before the next question begins.

## Real-time sync

Two viable ways to keep the admin console and every client in sync; the choice affects hosting, latency and how much backend work this needs.

| Approach | How sync works | Latency | Hosting | Build effort |
| --- | --- | --- | --- | --- |
| Artifact-hosted (shared store) | Admin and clients poll one shared session record on an interval (roughly every 1–2s) | \~1–2s, fine for a quiz timer | None — runs as a Claude artifact, re-deploys replace the page without losing the session | Low — no server to write, deploy or scale |
| Dedicated backend (e.g. Node + WebSocket) | A server pushes state to every connected client the instant it changes | Near-instant | You run and maintain a server (self-hosted or managed) | Higher — server, auth, deployment and scaling all become your responsibility |

For the scale this brief implies — one room of clients answering live — the Artifact-hosted approach meets every requirement (share link, admin control, live tally) with the least to build and maintain. The dedicated-backend path is worth it only if the app must run on its own domain, independent of claude.ai, or needs to support many simultaneous large rooms.

## Technical stack

The backend is Go and the frontend is Next.js, which settles the real-time approach above: this is the dedicated-backend path, built specifically for this project rather than run as a Claude artifact.

| Layer | Technology | Responsibility |
| --- | --- | --- |
| Backend | Go | Owns the session record, the timer engine, data.json validation, and the WebSocket hub that pushes state to every client. |
| Frontend | Next.js | One app, two routes — the admin console and the client join/answer view — plus the results chart UI. |
| Transport | WebSocket (Go ↔ Next.js) | Pushes timer ticks, question changes and live tallies to every connected client the instant they change, replacing the polling model described above. |

Shape of the two apps:

- Go backend: `POST /sessions` validates an uploaded `data.json` against the contract above and returns a session id plus both links; a `/ws/:sessionId` endpoint upgrades each admin and client connection to a WebSocket and is the only writer of the session record; an in-memory store is enough for a single backend instance, Redis if the service needs to survive a restart or run on more than one instance.
- Next.js frontend: a `/admin/[sessionId]` route for the host console and a `/join/[sessionId]` route for clients, both holding no session state of their own — everything shown comes from the WebSocket feed, so a page refresh just reconnects and replays the current state.

## Build plan

1. Data loader & validation — parse an uploaded `data.json` against the contract above, reject malformed files with a specific field-level error, preview the imported question list before a session starts.
2. Session creation & links — the admin action that turns a loaded file into a session, returning a host link and a share link.
3. Admin console shell — question list, status chips, navigation between questions, no timer logic yet.
4. Client join & waiting state — the share link opens a join screen, assigns a client id, shows a waiting screen until the admin starts a question.
5. Timer engine — duration selector, Start, live countdown, Finish early, auto-expiry; driven from the shared session record so every viewer sees the same clock.
6. Answer capture — a client's tap locks in one answer per question while running; taps after lock are rejected client-side.
7. Live tally & chart — a bar chart of option counts, shown on the same question screen for both roles once a question finishes.
8. Polish & edge cases — reconnecting after a dropped connection, a client joining mid-question, an admin reloading the console mid-session, a question nobody answered.

## Decisions

- Admin identity — no password; holding the host link is enough to control a session.
- Chart visibility — both admin and clients see the live results chart once a question finishes.
- Concurrent sessions — one session at a time per admin.
- Changing an answer — a client can change their pick only while the timer is still running; once it stops, their last pick before lock is final.
- Data retention — no export; the live tally is the only output needed.
