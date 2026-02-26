package tests

import (
        "os"
        "strings"
        "testing"

        "github.com/sampbb1996-dot/tsr/internal/tsr"
)

func TestCommitRequired(t *testing.T) {
        code := `
regime "CALM";
write_file("examples/nope.txt", "bad");
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit code for write_file without commit in CALM")
        }
        if !strings.Contains(stderr, "commit") {
                t.Errorf("expected 'commit' in error message, got: %s", stderr)
        }
        if !strings.Contains(stderr, "write_file") {
                t.Errorf("expected 'write_file' in error message, got: %s", stderr)
        }
}

func TestReadonlyForbidsWrite(t *testing.T) {
        code := `
regime "READONLY";
commit "try" {
  write_file("examples/nope.txt", "bad");
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit code for write_file in READONLY")
        }
        if !strings.Contains(stderr, "READONLY") {
                t.Errorf("expected 'READONLY' in error message, got: %s", stderr)
        }
}

func TestReadonlyForbidsHttp(t *testing.T) {
        code := `
regime "READONLY";
commit "try" {
  let page = http_get("https://example.com");
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit code for http_get in READONLY")
        }
        if !strings.Contains(stderr, "READONLY") {
                t.Errorf("expected 'READONLY' in error, got: %s", stderr)
        }
}

func TestStrainBranchBudget(t *testing.T) {
        code := `
strain { max_branch: 1, max_loop: 20, max_call_depth: 50, max_commit: 50, max_nest: 20 };
if (true) { say("a"); }
if (true) { say("b"); }
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit for max_branch exceeded")
        }
        if !strings.Contains(stderr, "max_branch") {
                t.Errorf("expected 'max_branch' in error, got: %s", stderr)
        }
}

func TestStrainLoopBudget(t *testing.T) {
        code := `
strain { max_branch: 20, max_loop: 2, max_call_depth: 50, max_commit: 50, max_nest: 20 };
let i = 0;
while (i < 10) {
  i = i + 1;
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit for max_loop exceeded")
        }
        if !strings.Contains(stderr, "max_loop") {
                t.Errorf("expected 'max_loop' in error, got: %s", stderr)
        }
}

func TestStrainCommitBudget(t *testing.T) {
        code := `
regime "CALM";
strain { max_branch: 20, max_loop: 20, max_call_depth: 50, max_commit: 1, max_nest: 20 };
commit "a" { say("first"); }
commit "b" { say("second"); }
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit for max_commit exceeded")
        }
        if !strings.Contains(stderr, "max_commit") {
                t.Errorf("expected 'max_commit' in error, got: %s", stderr)
        }
}

func TestStrainNestBudget(t *testing.T) {
        code := `
strain { max_branch: 20, max_loop: 20, max_call_depth: 50, max_commit: 50, max_nest: 1 };
if (true) {
  if (true) {
    say("deep");
  }
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit for max_nest exceeded")
        }
        if !strings.Contains(stderr, "max_nest") {
                t.Errorf("expected 'max_nest' in error, got: %s", stderr)
        }
}

func TestCalm_CommitAllowsWrite(t *testing.T) {
        code := `
regime "CALM";
commit "ok" {
  write_file("tsr_gov_test_out.txt", "ok");
}
say("done");
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        os.Remove("tsr_gov_test_out.txt")
        if exitCode != 0 {
                t.Fatalf("expected exit 0, got %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "done") {
                t.Errorf("expected 'done' in stdout, got: %s", stdout)
        }
}

func TestDefaultRegimeIsCalm(t *testing.T) {
        // No regime statement: should default to CALM
        // write_file without commit should fail
        code := `write_file("examples/nope.txt", "bad");`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit: default CALM requires commit for write_file")
        }
        if !strings.Contains(stderr, "commit") {
                t.Errorf("expected 'commit' in error, got: %s", stderr)
        }
}
