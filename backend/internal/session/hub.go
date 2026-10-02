package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"quizlive/internal/quiz"
)

const MaxClients = 500

// Sink is one live connection as seen by the hub. Send must not block.
type Sink interface {
	// Send queues a frame; returning false means the peer is too slow/dead.
	Send(frame []byte) bool
	Close()
}

// Conn is the hub's bookkeeping for a connection.
type Conn struct {
	Admin    bool
	ClientID string // set once joined
	Sink     Sink
}

// Msg is an inbound frame.
type Msg struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	ClientID    string `json:"clientId,omitempty"`
	QuestionID  string `json:"questionId,omitempty"`
	Key         string `json:"key,omitempty"`
	DurationSec int    `json:"durationSec,omitempty"`
}

type Hub struct {
	ID      string
	HostKey string

	clock Clock
	rec   *record
	conns map[*Conn]struct{}
	cmds  chan func()
	done  chan struct{}
	last  time.Time // last activity, hub goroutine only
}

func NewHub(id, hostKey string, set *quiz.Set, clock Clock) *Hub {
	if clock == nil {
		clock = RealClock
	}
	st := make([]QStatus, len(set.Questions))
	for i := range st {
		st[i] = NotStarted
	}
	h := &Hub{
		ID: id, HostKey: hostKey, clock: clock,
		rec: &record{
			set: set, phase: PhaseIdle, statuses: st,
			timer:   Timer{DurationSec: 30, Status: "idle"},
			clients: map[string]string{}, answers: map[string]map[string]string{},
		},
		conns: map[*Conn]struct{}{},
		cmds:  make(chan func(), 256),
		done:  make(chan struct{}),
	}
	h.last = clock.Now()
	if d := set.Questions[0].DefaultDurationSec; d != nil {
		h.rec.timer.DurationSec = *d
	}
	go h.loop()
	return h
}

func (h *Hub) loop() {
	for {
		select {
		case f := <-h.cmds:
			f()
		case <-h.done:
			return
		}
	}
}

// do runs f on the hub goroutine; it is the only way to touch session state.
func (h *Hub) do(f func()) {
	select {
	case h.cmds <- func() { h.last = h.clock.Now(); f() }:
	case <-h.done:
	}
}

// sync runs f on the hub goroutine and waits for it (used by tests and HTTP lookups).
func (h *Hub) sync(f func()) {
	ch := make(chan struct{})
	h.do(func() { f(); close(ch) })
	select {
	case <-ch:
	case <-h.done:
	}
}

// Stop shuts the hub down and closes every connection.
func (h *Hub) Stop() {
	h.sync(func() {
		for c := range h.conns {
			c.Sink.Close()
		}
	})
	select {
	case <-h.done:
	default:
		close(h.done)
	}
}

// IdleSince reports when the hub last processed a command.
func (h *Hub) IdleSince() time.Time {
	var t time.Time
	h.sync(func() { t = h.last })
	return t
}

func (h *Hub) Ended() bool {
	var e bool
	h.sync(func() { e = h.rec.phase == PhaseComplete })
	return e
}

// Attach registers a connection and sends it the current snapshot.
func (h *Hub) Attach(c *Conn) {
	h.do(func() {
		h.conns[c] = struct{}{}
		h.broadcast()
	})
}

func (h *Hub) Detach(c *Conn) {
	h.do(func() {
		if _, ok := h.conns[c]; ok {
			delete(h.conns, c)
			h.broadcast()
		}
	})
}

// Handle processes one inbound message from c.
func (h *Hub) Handle(c *Conn, m Msg) {
	h.do(func() {
		if _, ok := h.conns[c]; !ok {
			return
		}
		var err *Error
		switch m.Type {
		case "ping":
			h.send(c, map[string]string{"type": "pong"})
			return
		case "join":
			err = h.join(c, m)
		case "answer":
			err = h.answer(c, m)
		case "start":
			err = h.start(c, m)
		case "finish":
			err = h.finish(c)
		case "next":
			err = h.next(c)
		default:
			err = &Error{"bad_request", "unknown message type"}
		}
		if err != nil {
			h.send(c, map[string]any{"type": "error", "code": err.Code, "message": err.Message})
			return
		}
		h.broadcast()
	})
}

type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Code + ": " + e.Message }

var (
	errForbidden = &Error{"forbidden", "not allowed for this role"}
	errState     = &Error{"bad_state", "not allowed in the current state"}
)

func (h *Hub) join(c *Conn, m Msg) *Error {
	if c.Admin {
		return errForbidden
	}
	name := trimName(m.Name)
	if name == "" {
		return &Error{"bad_name", "display name is required (max 40 characters)"}
	}
	id := m.ClientID
	if _, known := h.rec.clients[id]; !known || id == "" {
		if len(h.rec.clients) >= MaxClients {
			return &Error{"full", "this session is full"}
		}
		id = newID()
	}
	h.rec.clients[id] = name
	// Same clientId in two tabs: last connection wins.
	for o := range h.conns {
		if o != c && o.ClientID == id {
			o.ClientID = ""
			h.send(o, map[string]string{"type": "error", "code": "replaced", "message": "joined from another tab"})
		}
	}
	c.ClientID = id
	h.send(c, map[string]string{"type": "joined", "clientId": id})
	return nil
}

func (h *Hub) answer(c *Conn, m Msg) *Error {
	r := h.rec
	if c.Admin {
		return errForbidden
	}
	if c.ClientID == "" {
		return &Error{"not_joined", "join before answering"}
	}
	if r.phase != PhaseRunning {
		return &Error{"locked", "answers are locked"}
	}
	q := r.set.Questions[r.current]
	if m.QuestionID != q.ID {
		return &Error{"locked", "that question is not active"}
	}
	for _, o := range q.Options {
		if o.Key == m.Key {
			if r.answers[q.ID] == nil {
				r.answers[q.ID] = map[string]string{}
			}
			r.answers[q.ID][c.ClientID] = m.Key // overwrite allowed while running
			return nil
		}
	}
	return &Error{"bad_option", "unknown option"}
}

func (h *Hub) start(c *Conn, m Msg) *Error {
	r := h.rec
	if !c.Admin {
		return errForbidden
	}
	if r.phase != PhaseIdle {
		return errState
	}
	if !quiz.ValidDuration(m.DurationSec) {
		return &Error{"bad_duration", "duration must be 10, 15, 30 or 60"}
	}
	r.epoch++
	epoch := r.epoch
	now := h.clock.Now()
	r.phase = PhaseRunning
	r.statuses[r.current] = Running
	r.timer = Timer{DurationSec: m.DurationSec, StartedAt: ms(now), Status: "running"}
	h.clock.AfterFunc(time.Duration(m.DurationSec)*time.Second, func() {
		h.do(func() {
			// No-op if the question was already finished manually.
			if r.epoch == epoch && r.phase == PhaseRunning {
				h.lock()
				h.broadcast()
			}
		})
	})
	return nil
}

func (h *Hub) finish(c *Conn) *Error {
	if !c.Admin {
		return errForbidden
	}
	if h.rec.phase != PhaseRunning {
		return errState
	}
	h.lock()
	return nil
}

// lock ends the running question: running → finished → results in one step.
func (h *Hub) lock() {
	r := h.rec
	r.statuses[r.current] = Finished
	r.timer.Status = "finished"
	r.phase = PhaseResults
}

func (h *Hub) next(c *Conn) *Error {
	r := h.rec
	if !c.Admin {
		return errForbidden
	}
	if r.phase != PhaseResults {
		return errState
	}
	if r.current == len(r.set.Questions)-1 {
		r.phase = PhaseComplete
		return nil
	}
	r.current++
	r.phase = PhaseIdle
	r.timer = Timer{DurationSec: 30, Status: "idle"}
	if d := r.set.Questions[r.current].DefaultDurationSec; d != nil {
		r.timer.DurationSec = *d
	}
	return nil
}

func (h *Hub) send(c *Conn, v any) {
	b, _ := json.Marshal(v)
	if !c.Sink.Send(b) {
		delete(h.conns, c)
		c.Sink.Close()
	}
}

func (h *Hub) broadcast() {
	for c := range h.conns {
		h.send(c, struct {
			Type string `json:"type"`
			View
		}{"state", h.view(c)})
	}
}

func (h *Hub) connectedClients() int {
	seen := map[string]bool{}
	for c := range h.conns {
		if c.ClientID != "" {
			seen[c.ClientID] = true
		}
	}
	return len(seen)
}

func (h *Hub) view(c *Conn) View {
	r := h.rec
	q := r.set.Questions[r.current]
	v := View{
		SessionID: h.ID, ServerNow: ms(h.clock.Now()), Phase: r.phase,
		Index: r.current, QuestionCnt: len(r.set.Questions),
		IsLast: r.current == len(r.set.Questions)-1,
		Timer:  r.timer, ClientCount: h.connectedClients(),
	}
	// Answers from clients who have since disconnected still count as answered.
	v.Answered = len(r.answers[q.ID])
	v.ClientCount = max(v.ClientCount, v.Answered)
	showQ := c.Admin || r.phase == PhaseRunning || r.phase == PhaseResults
	if r.phase == PhaseComplete && !c.Admin {
		showQ = false
	}
	if showQ {
		v.Question = &QuestionView{ID: q.ID, Text: q.Text, Options: q.Options}
	}
	if r.phase == PhaseResults || (r.phase == PhaseComplete && c.Admin) {
		v.Tally = r.tally(r.current)
	}
	if c.Admin {
		v.Role = "admin"
		v.Questions = make([]ListItem, len(r.set.Questions))
		v.PastTallies = map[string]map[string]int{}
		for i, qq := range r.set.Questions {
			v.Questions[i] = ListItem{ID: qq.ID, Text: qq.Text, Status: r.statuses[i], Options: qq.Options, DefaultDurationSec: qq.DefaultDurationSec}
			if r.statuses[i] == Finished {
				v.PastTallies[qq.ID] = r.tally(i)
			}
		}
	} else {
		v.Role = "client"
		v.Joined = c.ClientID != ""
		if v.Joined {
			v.MyAnswer = r.answers[q.ID][c.ClientID]
		}
	}
	return v
}

func trimName(s string) string {
	rs := []rune(s)
	for len(rs) > 0 && (rs[0] == ' ' || rs[0] == '\t') {
		rs = rs[1:]
	}
	for len(rs) > 0 && (rs[len(rs)-1] == ' ' || rs[len(rs)-1] == '\t') {
		rs = rs[:len(rs)-1]
	}
	if len(rs) > 40 {
		return ""
	}
	return string(rs)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(errors.New("crypto/rand failed"))
	}
	return hex.EncodeToString(b)
}

// NewSecret returns a random 128-bit hex string for host keys and session ids.
func NewSecret() string { return newID() }
