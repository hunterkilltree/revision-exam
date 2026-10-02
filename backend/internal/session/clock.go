package session

import "time"

// Clock abstracts time so the hub can be driven deterministically in tests.
type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Stopper
}

type Stopper interface{ Stop() bool }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) AfterFunc(d time.Duration, f func()) Stopper {
	return time.AfterFunc(d, f)
}

// RealClock is the production clock.
var RealClock Clock = realClock{}
