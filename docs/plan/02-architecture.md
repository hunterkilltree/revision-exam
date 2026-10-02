# Architecture

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
