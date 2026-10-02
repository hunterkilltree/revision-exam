# Test strategy

**Go (`go test -race ./...`)**
- `quiz`: table tests — valid file, missing `id`, duplicate ids, 1 option, 7 options, duplicate option keys, bad duration, non-JSON, oversize, multiple simultaneous errors.
- `session` hub: drive the actor with commands and a fake clock — state-machine transitions, illegal transitions rejected, answer overwrite, answer after finish rejected, expiry-after-finish no-op, tally correctness, no answers ⇒ all-zero tally, snapshot hides tally/question from clients at the right times.
- `httpapi`: `httptest` + WS client for auth (bad key → rejected), reconnect with same `clientId` keeps answer, malformed frames don't crash the hub.
- Concurrency: 200 simulated clients answering at once under `-race`.

**Frontend**
- Unit: `useCountdown` skew math; reducer that applies `state` messages; `BarChart` zero/rounding cases (Vitest + Testing Library).
- E2E (Playwright, Chromium is available): one admin context + three client contexts run a 2-question quiz; assertions on countdown, answered count, final bars; plus scenarios for client reload mid-question, admin reload mid-run, and late joiner.

**Edge-case checklist (phase 8)**: dropped Wi-Fi mid-question and answer changed after reconnect; join during `running`; admin reload during `running` and `results`; nobody answers; admin clicks Finish at the same instant as expiry; duplicate display names; same `clientId` in two tabs (last connection wins); question set with 1 question; 60 s timer on a backgrounded mobile tab (clock must be correct on return).
