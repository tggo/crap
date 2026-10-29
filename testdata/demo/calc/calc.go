package calc

type Acc struct{ n int }

// Sign is fully covered by calc_test.go.
func Sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// Grade has no test at all.
func Grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 50:
		return "C"
	}
	return "F"
}

func (a *Acc) Add(x int) { a.n += x }
