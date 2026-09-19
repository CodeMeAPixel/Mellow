package dashboard

import "testing"

func TestSuppressHidesSmallCounts(t *testing.T) {
	if got := suppress(0); got == nil || *got != 0 {
		t.Errorf("zero is safe to show, got %v", got)
	}
	for _, n := range []int64{1, 2, 4} {
		if got := suppress(n); got != nil {
			t.Errorf("%d could identify someone and must be hidden", n)
		}
	}
	for _, n := range []int64{5, 6, 250} {
		if got := suppress(n); got == nil || *got != n {
			t.Errorf("%d should be shown as is", n)
		}
	}
}
