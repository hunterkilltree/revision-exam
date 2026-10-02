// Package session holds the live-quiz session record and its single-writer hub.
package session

import (
	"time"

	"quizlive/internal/quiz"
)

type Phase string

const (
	PhaseIdle     Phase = "idle"
	PhaseRunning  Phase = "running"
	PhaseResults  Phase = "results"
	PhaseComplete Phase = "complete"
)

type QStatus string

const (
	NotStarted QStatus = "not_started"
	Running    QStatus = "running"
	Finished   QStatus = "finished"
)

type Timer struct {
	DurationSec int    `json:"durationSec"`
	StartedAt   int64  `json:"startedAt"` // unix ms, 0 while idle
	Status      string `json:"status"`    // idle | running | finished
}

// record is the mutable session state. Only the hub goroutine touches it.
type record struct {
	set      *quiz.Set
	current  int
	phase    Phase
	timer    Timer
	statuses []QStatus
	clients  map[string]string            // clientId -> display name
	answers  map[string]map[string]string // questionId -> clientId -> option key
	epoch    int
}

// tally is derived from answers on every read; it is never stored.
func (r *record) tally(qi int) map[string]int {
	q := r.set.Questions[qi]
	t := make(map[string]int, len(q.Options))
	for _, o := range q.Options {
		t[o.Key] = 0
	}
	for _, key := range r.answers[q.ID] {
		if _, ok := t[key]; ok {
			t[key]++
		}
	}
	return t
}

func ms(t time.Time) int64 { return t.UnixMilli() }
