package backoff

import (
	"math/rand"
	"time"
)

type Backoff struct {
	Seq []int
}

func (b Backoff) Pretty(n int) int {
	if len(b.Seq) == 0 {
		return 0
	} else if n >= len(b.Seq) {
		return b.Seq[len(b.Seq)-1]
	}
	return b.Seq[n-1]
}

func (b Backoff) Duration(n int) time.Duration {
	return jittered(b.Pretty(n))
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
