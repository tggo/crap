package calc

import "testing"

func TestSign(t *testing.T) {
	for x, want := range map[int]int{-5: -1, 0: 0, 7: 1} {
		if got := Sign(x); got != want {
			t.Errorf("Sign(%d) = %d, want %d", x, got, want)
		}
	}
}
