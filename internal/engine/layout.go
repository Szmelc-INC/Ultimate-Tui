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

// zoomStep is how many cells one zoom step is worth on a split's axis.
const zoomStep = 3

// zoomCells sums the zoom bias, in cells, of every leaf in a subtree.
func zoomCells(n node) int {
	if n.leaf != nil {
		return n.leaf.zoom * zoomStep
	}
	total := 0
	for _, c := range n.children {
		total += zoomCells(c.node)
	}
	return total
}

// shiftZoom redistributes zoom bias across a split's freshly distributed sizes
// while holding the total constant: a biased child grows (or shrinks) by its
// bias and the opposite amount is taken from (or given to) its siblings in
// proportion to their size, every slot floored at zoomFloor. Any rounding drift
// is parked on the largest slot so sum(out) == sum(sizes) exactly — the
// invariant that keeps bordered boxes from shearing.
func shiftZoom(sizes, bias []int) []int {
	n := len(sizes)
	out := append([]int(nil), sizes...)
	if n < 2 {
		return out
	}
	const zoomFloor = 3

	orig := 0
	for _, s := range sizes {
		orig += s
	}

	for i := 0; i < n; i++ {
		b := bias[i]
		if b == 0 {
			continue
		}
		donorTotal := 0
		for j := 0; j < n; j++ {
			if j != i {
				donorTotal += out[j]
			}
		}
		if donorTotal <= 0 {
			continue
		}
		switch {
		case b > 0:
			if room := donorTotal - (n-1)*zoomFloor; b > room {
				b = room
			}
		case out[i]+b < zoomFloor:
			b = zoomFloor - out[i]
		}
		if b == 0 {
			continue
		}
		snap := append([]int(nil), out...)
		out[i] += b
		removed, last := 0, -1
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			take := b * snap[j] / donorTotal
			if snap[j]-take < zoomFloor {
				take = snap[j] - zoomFloor
			}
			if take < 0 {
				take = 0
			}
			out[j] -= take
			removed += take
			last = j
		}
		if d := b - removed; d != 0 && last >= 0 {
			out[last] -= d
		}
	}

	// Guarantee sum(out) == orig even if a clamp swallowed some drift.
	cur := 0
	for _, s := range out {
		cur += s
	}
	if d := orig - cur; d != 0 {
		big := 0
		for j := 1; j < n; j++ {
			if out[j] > out[big] {
				big = j
			}
		}
		out[big] += d
	}
	for j := range out {
		if out[j] < 0 {
			out[j] = 0
		}
	}
	return out
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
