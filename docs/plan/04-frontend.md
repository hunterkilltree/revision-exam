# Frontend (Next.js, App Router, TypeScript)

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
