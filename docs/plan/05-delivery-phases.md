# Delivery phases

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
