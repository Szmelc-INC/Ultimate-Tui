package engine

import (
	"math"
	"strconv"
	"strings"
)

type sizeKind int

const (
	flex sizeKind = iota
	fixed
	percent
)

type sizeSpec struct {
	kind  sizeKind
	value float64
}

// parseSize reads a Child.Size string. Anything unparseable degrades to flex
// weight 1 rather than erroring — layout should always render something.
func parseSize(s string) sizeSpec {
	s = strings.TrimSpace(s)
	switch {
	case s == "" || s == "*":
		return sizeSpec{flex, 1}
	case strings.HasSuffix(s, "%"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil || v < 0 {
			return sizeSpec{flex, 1}
		}
		return sizeSpec{percent, v}
	case strings.HasSuffix(s, "*"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "*"), 64)
		if err != nil || v <= 0 {
			v = 1
		}
		return sizeSpec{flex, v}
	default:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 {
			return sizeSpec{flex, 1}
		}
		return sizeSpec{fixed, v}
	}
}

// distribute splits total cells across specs: fixed slots take their cells,
// percent slots take their share of total, flex slots divide the remainder by
// weight. Rounding drift lands on the last flex slot (or last slot if none).
func distribute(specs []sizeSpec, total int) []int {
	out := make([]int, len(specs))
	used := 0
	flexWeight := 0.0

	for i, s := range specs {
		switch s.kind {
		case fixed:
			out[i] = int(s.value)
			used += out[i]
		case percent:
			out[i] = int(math.Round(s.value / 100 * float64(total)))
			used += out[i]
		case flex:
			flexWeight += s.value
		}
	}

	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}

	acc, lastFlex := 0, -1
	if flexWeight > 0 {
		for i, s := range specs {
			if s.kind == flex {
				out[i] = int(float64(remaining) * s.value / flexWeight)
				acc += out[i]
				lastFlex = i
			}
		}
	}

	drift := remaining - acc
	if flexWeight > 0 && lastFlex >= 0 {
		out[lastFlex] += drift
	} else if len(out) > 0 && drift != 0 {
		out[len(out)-1] += drift
	}

	for i := range out {
		if out[i] < 0 {
			out[i] = 0
		}
	}
	return out
}
