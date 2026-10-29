// Command crap scores every Go function by CRAP: CC² × (1 − coverage)³ + CC.
// It runs go test with a coverage profile, so a complex function without
// tests ranks above a complex function that is tested.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"golang.org/x/tools/cover"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(run(os.Args[1:], wd, os.Stdout, os.Stderr))
}

// run returns 0 on success, 1 when a function is over -max, 2 on any error.
func run(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("crap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	maxCRAP := fs.Float64("max", 0, "exit 1 when any function's CRAP is above this (0 = off)")
	top := fs.Int("top", 0, "print only the N worst functions (0 = all)")
	asJSON := fs.Bool("json", false, "print JSON instead of a table")
	profile := fs.String("profile", "", "read this coverage profile instead of running go test")
	tags := fs.String("tags", "", "build tags, passed to go test and go list")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: crap [flags] [packages]   (default ./...)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	funcs, err := collect(dir, patterns, *profile, *tags, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "crap:", err)
		return 2
	}

	if err := report(stdout, worst(funcs, *top), *asJSON); err != nil {
		fmt.Fprintln(stderr, "crap:", err)
		return 2
	}

	if over := countOver(funcs, *maxCRAP); over > 0 {
		fmt.Fprintf(stderr, "crap: %d function(s) over CRAP %g\n", over, *maxCRAP)
		return 1
	}
	return 0
}

// countOver is 0 when max is off (0 or less).
func countOver(funcs []Func, max float64) int {
	if max <= 0 {
		return 0
	}
	over := 0
	for _, f := range funcs {
		if f.CRAP > max {
			over++
		}
	}
	return over
}

// worst is the first n of funcs, or all of them when n is 0 or too large.
func worst(funcs []Func, n int) []Func {
	if n <= 0 {
		return funcs
	}
	return funcs[:min(n, len(funcs))]
}

func collect(dir string, patterns []string, profile, tags string, stderr io.Writer) ([]Func, error) {
	var tagArgs []string
	if tags != "" {
		tagArgs = []string{"-tags", tags}
	}
	if profile == "" {
		tmp, err := os.CreateTemp("", "crap-*.out")
		if err != nil {
			return nil, err
		}
		tmp.Close()
		defer os.Remove(tmp.Name())
		profile = tmp.Name()
		args := append(append([]string{"test", "-covermode=set", "-coverprofile=" + profile}, tagArgs...), patterns...)
		if out, err := goCmd(dir, args...); err != nil {
			stderr.Write(out)
			return nil, errors.New("go test failed")
		}
	} else if !filepath.IsAbs(profile) {
		profile = filepath.Join(dir, profile)
	}
	profiles, err := cover.ParseProfiles(profile)
	if err != nil {
		return nil, err
	}
	pkgs, err := list(dir, patterns, tagArgs)
	if err != nil {
		return nil, err
	}
	return analyze(dir, pkgs, profiles)
}

func list(dir string, patterns, tagArgs []string) ([]pkg, error) {
	args := append(append([]string{"list", "-json=Dir,ImportPath,GoFiles,CgoFiles"}, tagArgs...), patterns...)
	out, err := goCmd(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("go list: %s", bytes.TrimSpace(out))
	}
	var pkgs []pkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func goCmd(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.Bytes(), err
}

func report(w io.Writer, funcs []Func, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if funcs == nil {
			funcs = []Func{}
		}
		return enc.Encode(funcs)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "CRAP\tCC\tCov%\t \tFunction\tLocation\t")
	for _, f := range funcs {
		fmt.Fprintf(tw, "%.1f\t%d\t%.1f\t \t%s\t%s:%d\t\n", f.CRAP, f.Complexity, f.Coverage*100, f.Name, f.File, f.Line)
	}
	return tw.Flush()
}
