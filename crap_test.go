package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"testing"
)

func funcs(t *testing.T, src string) map[string]*ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			out[fd.Name.Name] = fd
		}
	}
	return out
}

func TestComplexity(t *testing.T) {
	fs := funcs(t, `package x
func straight() int { return 1 }
func branches(a, b bool, xs []int) int {
	n := 0
	if a && b { n++ } else if a || b { n-- }
	for i := 0; i < 3; i++ { n++ }
	for range xs { n++ }
	switch n {
	case 1, 2:
		n = 0
	case 3:
	default:
	}
	return n
}
func selects(c, d chan int) {
	select {
	case <-c:
	case <-d:
	default:
	}
}
func literal() func() int {
	return func() int { if true { return 1 }; return 0 }
}
`)
	cases := map[string]int{
		// 1 + if + && + else-if + || + for + range + case + case
		"branches": 9,
		"straight": 1,
		"selects":  3,
		"literal":  2,
	}
	for name, want := range cases {
		if got := complexity(fs[name]); got != want {
			t.Errorf("%s: complexity %d, want %d", name, got, want)
		}
	}
}

func TestFuncName(t *testing.T) {
	fs := funcs(t, `package x
type T struct{}
type G[K comparable, V any] struct{}
type H[E any] struct{}
func plain() {}
func (T) value() {}
func (*T) pointer() {}
func (*G[K, V]) twoParams() {}
func (H[E]) oneParam() {}
`)
	want := map[string]string{
		"plain":     "plain",
		"value":     "(T).value",
		"pointer":   "(*T).pointer",
		"twoParams": "(*G).twoParams",
		"oneParam":  "(H).oneParam",
	}
	for name, w := range want {
		if got := funcName(fs[name]); got != w {
			t.Errorf("%s: %q, want %q", name, got, w)
		}
	}
}

func TestStatementsWithoutProfile(t *testing.T) {
	fs := funcs(t, `package x
func f(x int) int {
	y := x
	if y > 0 {
		y++
	}
	switch y {
	case 1:
		return 1
	}
	return y
}
func empty() {}
`)
	// y :=, if, y++, switch, return 1, return y
	if got := statements(fs["f"]); got != 6 {
		t.Errorf("f: %d statements, want 6", got)
	}
	if got := statements(fs["empty"]); got != 0 {
		t.Errorf("empty: %d statements, want 0", got)
	}
}

func TestSortWorstFirstBreaksTiesByName(t *testing.T) {
	fs := []Func{{Name: "b", CRAP: 2}, {Name: "a", CRAP: 2}, {Name: "c", CRAP: 5}}
	sortWorstFirst(fs)
	if fs[0].Name != "c" || fs[1].Name != "a" || fs[2].Name != "b" {
		t.Errorf("order: %s %s %s", fs[0].Name, fs[1].Name, fs[2].Name)
	}
}

func TestScore(t *testing.T) {
	cases := []struct {
		cc   int
		cov  float64
		want float64
	}{
		{1, 0, 2},
		{1, 1, 1},
		{4, 0, 20},
		{10, 1, 10},
		{5, 0.5, 25*0.125 + 5},
	}
	for _, c := range cases {
		if got := score(c.cc, c.cov); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("score(%d, %v) = %v, want %v", c.cc, c.cov, got, c.want)
		}
	}
}
