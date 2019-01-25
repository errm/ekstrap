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

package backoff_test

import (
	"testing"
	"time"

	"github.com/errm/ekstrap/pkg/backoff"
)

func TestEmptyBackoff(t *testing.T) {
	empty := backoff.Backoff{}

	if empty.Duration(2) != time.Duration(0) {
		t.Error("Empty backoff should always return 0")
	}

	if empty.Duration(7) != time.Duration(0) {
		t.Error("Empty backoff should always return 0")
	}
}

func TestJitteredBackoff(t *testing.T) {
	seq := backoff.Backoff{Seq: []int{1, 2, 4, 8}}

	//nolint:staticcheck
	if seq.Duration(1) == seq.Duration(1) {
		t.Error("Jitter should ensure calls are not equal")
	}

	//nolint:staticcheck
	if seq.Duration(4) == seq.Duration(4) {
		t.Error("Jitter should ensure calls are not equal")
	}

	between(t, seq.Duration(1), 500*time.Millisecond, 1500*time.Millisecond)
	between(t, seq.Duration(2), 1500*time.Millisecond, 2500*time.Millisecond)
	between(t, seq.Duration(3), 3500*time.Millisecond, 4500*time.Millisecond)
	between(t, seq.Duration(4), 7500*time.Millisecond, 8500*time.Millisecond)
	between(t, seq.Duration(5), 7500*time.Millisecond, 8500*time.Millisecond)

}

func between(t *testing.T, actual, low, high time.Duration) {
	if actual < low {
		t.Fatalf("Got %s, Expecting >= %s", actual, low)
	}
	if actual > high {
		t.Fatalf("Got %s, Expecting <= %s", actual, high)
	}
}
