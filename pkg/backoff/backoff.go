package backoff

import (
	"math/rand"
	"time"
)

// Backoff represents a back-off strategy
// Seq represents a seqence of durations in seconds to backoff for
type Backoff struct {
	Seq []int
}

// Duration returns a duration to backoff for given the step number n.
// The duration is jittered by +- 500ms
func (b Backoff) Duration(n int) time.Duration {
	if len(b.Seq) == 0 {
		return jittered(0)
	} else if n >= len(b.Seq) {
		return jittered(b.Seq[len(b.Seq)-1])
	}
	return jittered(b.Seq[n-1])
}

func jittered(t int) time.Duration {
	if t == 0 {
		return time.Duration(0)
	}
	millis := t * 1000
	//jitter arround the current second
	jitter := 500 - rand.Intn(1000)
	return time.Duration(millis+jitter) * time.Millisecond
}
