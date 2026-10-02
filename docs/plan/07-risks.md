# Risks & mitigations

| Risk | Mitigation |
|---|---|
| Clock drift or backgrounded tabs freeze timers | Server-authoritative time + `serverNow` offset; recompute from timestamps on every render/visibility change, never decrement a counter. |
| Guessable host control | Separate high-entropy `hostKey`; admin role fixed at WS upgrade. |
| In-memory store loses sessions on restart | Accepted for v1 (single instance); store interface isolates a Redis implementation. Document that deploys end live quizzes — avoid deploying during an event. |
| Single instance limits scale | One hub goroutine per session is cheap; target ≈ 500 clients/session, tens of sessions. Horizontal scale would need sticky routing or Redis pub/sub — out of scope. |
| Prototype/requirement drift (e.g. timer option order "15/10/30/60") | Present options in ascending order (10/15/30/60) unless the prototype shows otherwise — confirm in phase 5 against the boards. |
| Abuse of public join link | Name length cap, message rate limit, per-session client cap. |
