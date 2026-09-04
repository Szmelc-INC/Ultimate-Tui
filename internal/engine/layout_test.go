package engine

import "testing"

func specs(ss ...string) []sizeSpec {
	out := make([]sizeSpec, len(ss))
	for i, s := range ss {
		out[i] = parseSize(s)
	}
	return out
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func TestDistributeFillsExactly(t *testing.T) {
	cases := []struct {
		name  string
		specs []sizeSpec
		total int
	}{
		{"two flex", specs("*", "*"), 101},
		{"pct plus flex", specs("30%", "*"), 100},
		{"fixed plus flex", specs("20", "*", "*"), 83},
		{"weighted flex", specs("1*", "3*"), 100},
		{"all percent rounding", specs("33%", "33%", "34%"), 100},
	}
	for _, tc := range cases {
		got := distribute(tc.specs, tc.total)
		if s := sum(got); s != tc.total {
			t.Errorf("%s: sum(%v) = %d, want %d", tc.name, got, s, tc.total)
		}
	}
}

func TestDistributeWeightedFlex(t *testing.T) {
	got := distribute(specs("1*", "3*"), 100)
	if got[0] != 25 || got[1] != 75 {
		t.Fatalf("weighted flex = %v, want [25 75]", got)
	}
}

func TestDistributeNoNegatives(t *testing.T) {
	// Fixed sizes overflow the container; flex must clamp at 0, not go negative.
	got := distribute(specs("50", "50", "*"), 40)
	for i, v := range got {
		if v < 0 {
			t.Fatalf("slot %d negative: %v", i, got)
		}
	}
}
