# Decisions that need pinning down before coding

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
