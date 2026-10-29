package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The demo module is run the way a user runs the tool: from its root, with go test.
func TestRunOnDemoModule(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-json", "./..."}, "testdata/demo", &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got []Func
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("bad json: %v\n%s", err, out.String())
	}
	byName := map[string]Func{}
	for _, f := range got {
		byName[f.Name] = f
	}
	if len(byName) != 3 {
		t.Fatalf("want 3 functions, got %+v", got)
	}
	sign, grade, add := byName["Sign"], byName["Grade"], byName["(*Acc).Add"]
	if sign.Complexity != 3 || sign.Coverage != 1 || sign.CRAP != 3 {
		t.Errorf("Sign: %+v", sign)
	}
	if grade.Complexity != 4 || grade.Coverage != 0 || grade.CRAP != 20 {
		t.Errorf("Grade: %+v", grade)
	}
	if add.Package != "example.com/demo/calc" || add.Coverage != 0 || add.CRAP != 2 {
		t.Errorf("Add: %+v", add)
	}
	if got[0].Name != "Grade" {
		t.Errorf("worst first, got %s", got[0].Name)
	}
	if !strings.HasSuffix(grade.File, "calc/calc.go") || grade.Line != 17 {
		t.Errorf("Grade position: %s:%d", grade.File, grade.Line)
	}
}

func TestMaxFailsTheRun(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-max", "10", "./..."}, "testdata/demo", &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1 (Grade is 20)", code)
	}
	if !strings.Contains(errOut.String(), "1 function") {
		t.Errorf("stderr: %q", errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"-max", "30", "./..."}, "testdata/demo", &out, &errOut); code != 0 {
		t.Fatalf("exit %d with -max 30: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Grade") {
		t.Errorf("table missing Grade:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "100.0") {
		t.Errorf("table should show Sign at 100.0%% covered:\n%s", out.String())
	}

	// Grade is exactly 20: at the limit is not over it.
	if code := run([]string{"-max", "20", "-top", "5", "./..."}, "testdata/demo", &out, &errOut); code != 0 {
		t.Fatalf("exit %d with -max 20: %s", code, errOut.String())
	}
}

// Files behind a build tag are scored only when the tag is passed.
func TestTags(t *testing.T) {
	count := func(args ...string) int {
		var out, errOut bytes.Buffer
		if code := run(append([]string{"-json"}, args...), "testdata/demo", &out, &errOut); code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		var got []Func
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return len(got)
	}
	if n := count("./..."); n != 3 {
		t.Errorf("without tag: %d functions, want 3", n)
	}
	if n := count("-tags", "extra", "./..."); n != 4 {
		t.Errorf("with -tags extra: %d functions, want 4", n)
	}
}

func TestTopAndReuseProfile(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-top", "1", "./..."}, "testdata/demo", &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "Grade") {
		t.Fatalf("want header + Grade, got:\n%s", out.String())
	}

	// A profile where nothing ran: Sign drops to 0% and CRAP 12.
	profile := t.TempDir() + "/p.out"
	body := "mode: set\nexample.com/demo/calc/calc.go:6.22,7.11 1 0\n"
	if err := os.WriteFile(profile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"-json", "-profile", profile, "./..."}, "testdata/demo", &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got []Func
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		if f.Name == "Sign" && (f.Coverage != 0 || f.CRAP != 12) {
			t.Errorf("Sign from the given profile: %+v", f)
		}
	}
}

func TestFailingTestsStopTheRun(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"./nope/..."}, "testdata/demo", &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}
