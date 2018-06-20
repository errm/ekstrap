/*
Copyright 2018 Edward Robinson.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	//jitter around the current second
	jitter := 500 - rand.Intn(1000)
	return time.Duration(millis+jitter) * time.Millisecond
}
