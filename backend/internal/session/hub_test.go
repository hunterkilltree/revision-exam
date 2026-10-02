package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"quizlive/internal/quiz"
)

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}
type fakeTimer struct {
	at   time.Time
	f    func()
	done bool
}

func (t *fakeTimer) Stop() bool { t.done = true; return true }
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}
func (c *fakeClock) AfterFunc(d time.Duration, f func()) Stopper {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	return t
}
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []*fakeTimer
	for _, t := range c.timers {
		if !t.done && !t.at.After(c.now) {
			t.done = true
			due = append(due, t)
		}
	}
	c.mu.Unlock()
	for _, t := range due {
		t.f()
	}
}

type sink struct {
	mu     sync.Mutex
	frames [][]byte
}

func (s *sink) Send(b []byte) bool {
	s.mu.Lock()
	s.frames = append(s.frames, b)
	s.mu.Unlock()
	return true
}
func (s *sink) Close() {}
func (s *sink) lastState(t *testing.T) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.frames) - 1; i >= 0; i-- {
		var m map[string]any
		json.Unmarshal(s.frames[i], &m)
		if m["type"] == "state" {
			return m
		}
	}
	t.Fatal("no state frame")
	return nil
}
func (s *sink) lastError() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.frames) - 1; i >= 0; i-- {
		var m map[string]any
		json.Unmarshal(s.frames[i], &m)
		if m["type"] == "error" {
			return m["code"].(string)
		}
	}
	return ""
}

func testSet(n int) *quiz.Set {
	s := &quiz.Set{}
	for i := 0; i < n; i++ {
		s.Questions = append(s.Questions, quiz.Question{
			ID: fmt.Sprintf("q%d", i+1), Text: "T",
			Options: []quiz.Option{{Key: "A", Text: "a"}, {Key: "B", Text: "b"}, {Key: "C", Text: "c"}},
		})
	}
	return s
}

type rig struct {
	h   *Hub
	clk *fakeClock
	t   *testing.T
}

func newRig(t *testing.T, n int) *rig {
	clk := &fakeClock{now: time.UnixMilli(1_000_000)}
	h := NewHub("s", "k", testSet(n), clk)
	t.Cleanup(h.Stop)
	return &rig{h, clk, t}
}

func (r *rig) conn(admin bool) (*Conn, *sink) {
	s := &sink{}
	c := &Conn{Admin: admin, Sink: s}
	r.h.Attach(c)
	r.h.sync(func() {})
	return c, s
}
func (r *rig) send(c *Conn, m Msg) { r.h.Handle(c, m); r.h.sync(func() {}) }
func (r *rig) client(name string) (*Conn, *sink) {
	c, s := r.conn(false)
	r.send(c, Msg{Type: "join", Name: name})
	return c, s
}

func TestFullFlow(t *testing.T) {
	r := newRig(t, 2)
	admin, as := r.conn(true)
	c1, s1 := r.client("a")
	c2, _ := r.client("b")
	_, _ = c2, s1

	if st := s1.lastState(t); st["phase"] != "idle" || st["question"] != nil {
		t.Fatalf("client should not see question while idle: %v", st)
	}
	r.send(admin, Msg{Type: "start", DurationSec: 15})
	if as.lastState(t)["phase"] != "running" {
		t.Fatal("expected running")
	}
	r.send(c1, Msg{Type: "answer", QuestionID: "q1", Key: "A"})
	r.send(c1, Msg{Type: "answer", QuestionID: "q1", Key: "B"}) // overwrite
	r.send(c2, Msg{Type: "answer", QuestionID: "q1", Key: "B"})
	st := as.lastState(t)
	if st["answeredCount"].(float64) != 2 || st["clientCount"].(float64) != 2 {
		t.Fatalf("counts: %v", st)
	}
	if st["tally"] != nil || s1.lastState(t)["tally"] != nil {
		t.Fatal("tally must be hidden while running")
	}
	if s1.lastState(t)["myAnswer"] != "B" {
		t.Fatal("myAnswer should be B")
	}
	r.clk.Advance(15 * time.Second)
	r.h.sync(func() {})
	st = s1.lastState(t)
	tally := st["tally"].(map[string]any)
	if st["phase"] != "results" || tally["A"].(float64) != 0 || tally["B"].(float64) != 2 {
		t.Fatalf("results: %v", st)
	}
	r.send(c1, Msg{Type: "answer", QuestionID: "q1", Key: "A"})
	if s1.lastError() != "locked" {
		t.Fatal("answer after lock must be rejected")
	}
	r.send(admin, Msg{Type: "next"})
	if as.lastState(t)["currentQuestionIndex"].(float64) != 1 {
		t.Fatal("expected q2")
	}
	r.send(admin, Msg{Type: "start", DurationSec: 10})
	r.send(admin, Msg{Type: "finish"})
	r.send(admin, Msg{Type: "next"})
	if as.lastState(t)["phase"] != "complete" {
		t.Fatal("expected complete")
	}
	st = as.lastState(t)
	if st["pastTallies"].(map[string]any)["q2"].(map[string]any)["A"].(float64) != 0 {
		t.Fatal("zero-answer question should have all-zero tally")
	}
}

func TestExpireAfterFinishIsNoOp(t *testing.T) {
	r := newRig(t, 2)
	admin, as := r.conn(true)
	r.send(admin, Msg{Type: "start", DurationSec: 10})
	r.send(admin, Msg{Type: "finish"})
	r.send(admin, Msg{Type: "next"})
	r.send(admin, Msg{Type: "start", DurationSec: 30})
	r.clk.Advance(10 * time.Second) // stale timer from q1 fires
	r.h.sync(func() {})
	if as.lastState(t)["phase"] != "running" {
		t.Fatal("stale expiry must not end the new question")
	}
}

func TestAuthAndIllegalTransitions(t *testing.T) {
	r := newRig(t, 1)
	admin, as := r.conn(true)
	c, cs := r.client("x")
	r.send(c, Msg{Type: "start", DurationSec: 15})
	if cs.lastError() != "forbidden" {
		t.Fatal("client must not start")
	}
	r.send(admin, Msg{Type: "finish"})
	if as.lastError() != "bad_state" {
		t.Fatal("finish while idle")
	}
	r.send(admin, Msg{Type: "start", DurationSec: 20})
	if as.lastError() != "bad_duration" {
		t.Fatal("bad duration")
	}
	r.send(c, Msg{Type: "answer", QuestionID: "q1", Key: "A"})
	if cs.lastError() != "locked" {
		t.Fatal("answer while idle")
	}
	r.send(admin, Msg{Type: "start", DurationSec: 10})
	r.send(c, Msg{Type: "answer", QuestionID: "q1", Key: "Z"})
	if cs.lastError() != "bad_option" {
		t.Fatal("bad option")
	}
	r.send(admin, Msg{Type: "next"})
	if as.lastError() != "bad_state" {
		t.Fatal("next while running")
	}
}

func TestReconnectKeepsAnswer(t *testing.T) {
	r := newRig(t, 1)
	admin, _ := r.conn(true)
	c, cs := r.client("x")
	id := ""
	r.h.sync(func() { id = c.ClientID })
	r.send(admin, Msg{Type: "start", DurationSec: 30})
	r.send(c, Msg{Type: "answer", QuestionID: "q1", Key: "C"})
	r.h.Detach(c)
	c2, s2 := r.conn(false)
	r.send(c2, Msg{Type: "join", Name: "x", ClientID: id})
	_ = cs
	st := s2.lastState(t)
	if st["myAnswer"] != "C" || st["phase"] != "running" {
		t.Fatalf("resume failed: %v", st)
	}
}

func TestConcurrentAnswers(t *testing.T) {
	r := newRig(t, 1)
	admin, as := r.conn(true)
	r.send(admin, Msg{Type: "start", DurationSec: 60})
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _ := r.client(fmt.Sprintf("c%d", i))
			r.send(c, Msg{Type: "answer", QuestionID: "q1", Key: []string{"A", "B", "C"}[i%3]})
		}(i)
	}
	wg.Wait()
	r.send(admin, Msg{Type: "finish"})
	st := as.lastState(t)
	tl := st["tally"].(map[string]any)
	if tl["A"].(float64)+tl["B"].(float64)+tl["C"].(float64) != 200 {
		t.Fatalf("tally: %v", tl)
	}
}
