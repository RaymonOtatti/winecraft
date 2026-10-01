package rate

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func TestBucketBurstThenRefill(t *testing.T) {
	var b Bucket
	for i := 0; i < 3; i++ {
		if !b.Take(t0, time.Second, 3, 1) {
			t.Fatalf("take %d inside the burst failed", i)
		}
	}
	if b.Take(t0, time.Second, 3, 1) {
		t.Fatal("burst exhausted, take must fail")
	}
	if !b.Take(t0.Add(time.Second), time.Second, 3, 1) {
		t.Fatal("one token per interval must refill")
	}
}

func TestBucketNeverExceedsBurst(t *testing.T) {
	var b Bucket
	b.Take(t0, time.Second, 2, 0)
	if got := b.Available(t0.Add(time.Hour), time.Second, 2); got != 2 {
		t.Fatalf("Available after an hour = %v, want the burst of 2", got)
	}
}

func TestBucketMultiTokenCost(t *testing.T) {
	var b Bucket
	if !b.Take(t0, time.Second, 3, 2) || b.Take(t0, time.Second, 3, 2) {
		t.Fatal("a cost of 2 must leave 1 token, not enough for another 2")
	}
}

func TestBucketIgnoresClockGoingBackwards(t *testing.T) {
	var b Bucket
	b.Take(t0, time.Second, 1, 1)
	if b.Take(t0.Add(-time.Hour), time.Second, 1, 1) {
		t.Fatal("an earlier timestamp must not refill or misbehave")
	}
}
