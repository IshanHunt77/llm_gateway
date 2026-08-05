package ratelimit

import "testing"

func TestRateLimit(t *testing.T) {
	b := New(3, 1)
	expected := []bool{true, true, true, false}
	for i, want := range expected {
		got := b.Allow()
		if got != want {
			t.Errorf("call %d: got %v, want %v", i+1, got, want)
		}
	}
}
