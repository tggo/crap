package main

import (
	"cmp"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"path/filepath"
	"slices"

	"golang.org/x/tools/cover"
)

// Func is one function's CRAP result.
type Func struct {
	Name       string  `json:"name"`
	Package    string  `json:"package"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Complexity int     `json:"complexity"`
	Coverage   float64 `json:"coverage"`
	CRAP       float64 `json:"crap"`
}

// score is CC² × (1 − coverage)³ + CC, with coverage in [0, 1].
func score(cc int, coverage float64) float64 {
	c := float64(cc)
	return c*c*math.Pow(1-coverage, 3) + c
}

// complexity is gocyclo's count: 1, plus one per if, for, range, non-default
// case, non-default select case, && and ||. Function literals count toward
// the function that holds them.
func complexity(fn *ast.FuncDecl) int {
	n := 1
	ast.Inspect(fn, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			n++
		case *ast.CaseClause:
			if x.List != nil {
				n++
			}
		case *ast.CommClause:
			if x.Comm != nil {
				n++
			}
		case *ast.BinaryExpr:
			if x.Op == token.LAND || x.Op == token.LOR {
				n++
			}
		}
		return true
	})
	return n
}

// funcName spells methods the way go tool cover does: (*T).M or (T).M.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return "(" + recvType(fn.Recv.List[0].Type) + ")." + fn.Name.Name
}

func recvType(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return "*" + recvType(x.X)
	case *ast.IndexExpr:
		return recvType(x.X)
	case *ast.IndexListExpr:
		return recvType(x.X)
	case *ast.Ident:
		return x.Name
	}
	return "?"
}

// coverage is the share of the function's statements that ran. A function
// with no statements is fully covered; a file absent from the profile is not.
func coverage(blocks []cover.ProfileBlock, start, end token.Position) float64 {
	total, hit := 0, 0
	for _, b := range blocks {
		if b.StartLine < start.Line || b.EndLine > end.Line {
			continue
		}
		total += b.NumStmt
		if b.Count > 0 {
			hit += b.NumStmt
		}
	}
	if total == 0 {
		return 1
	}
	return float64(hit) / float64(total)
}

// pkg is the part of go list's output the analysis needs.
type pkg struct {
	Dir        string
	ImportPath string
	GoFiles    []string
	CgoFiles   []string
}

// analyze scores every function with a body in the packages. Profile keys are
// import paths, so a file is looked up as ImportPath/name.go — never by
// absolute path.
func analyze(root string, pkgs []pkg, profiles []*cover.Profile) ([]Func, error) {
	blocks := map[string][]cover.ProfileBlock{}
	for _, p := range profiles {
		blocks[p.FileName] = p.Blocks
	}
	var out []Func
	fset := token.NewFileSet()
	for _, p := range pkgs {
		for _, name := range append(p.GoFiles, p.CgoFiles...) {
			fileBlocks, profiled := blocks[p.ImportPath+"/"+name]
			found, err := analyzeFile(fset, root, p, name, fileBlocks, profiled)
			if err != nil {
				return nil, err
			}
			out = append(out, found...)
		}
	}
	sortWorstFirst(out)
	return out, nil
}

func analyzeFile(fset *token.FileSet, root string, p pkg, name string, blocks []cover.ProfileBlock, profiled bool) ([]Func, error) {
	path := filepath.Join(p.Dir, name)
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	var out []Func
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		cov := 0.0
		if profiled {
			cov = coverage(blocks, fset.Position(fn.Pos()), fset.Position(fn.End()))
		}
		cc := complexity(fn)
		out = append(out, Func{
			Name:       funcName(fn),
			Package:    p.ImportPath,
			File:       filepath.ToSlash(rel),
			Line:       fset.Position(fn.Pos()).Line,
			Complexity: cc,
			Coverage:   cov,
			CRAP:       score(cc, cov),
		})
	}
	return out, nil
}

func sortWorstFirst(out []Func) {
	slices.SortStableFunc(out, func(a, b Func) int {
		return cmp.Or(
			cmp.Compare(b.CRAP, a.CRAP),
			cmp.Compare(a.Package+a.Name, b.Package+b.Name),
		)
	})
}
