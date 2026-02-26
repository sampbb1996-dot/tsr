package tests

import (
        "os"
        "strings"
        "testing"

        "github.com/sampbb1996-dot/tsr/internal/kernel"
        "github.com/sampbb1996-dot/tsr/internal/tsr"
)

func TestKernelCompileBasic(t *testing.T) {
        src := `
mode "CALM"
budget auto
set greeting = "hello from KERNEL"
say greeting
`
        tsrSrc, warnings, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        _ = warnings

        if !strings.Contains(tsrSrc, `regime "CALM"`) {
                t.Errorf("expected regime line, got:\n%s", tsrSrc)
        }
        if !strings.Contains(tsrSrc, "strain {") {
                t.Errorf("expected strain line, got:\n%s", tsrSrc)
        }
}

func TestKernelCompileReadonly(t *testing.T) {
        src := `
mode "READONLY"
budget auto
http_get "https://example.com" -> page
say page
`
        tsrSrc, warnings, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if !strings.Contains(tsrSrc, `regime "READONLY"`) {
                t.Errorf("expected READONLY regime, got:\n%s", tsrSrc)
        }
        if len(warnings) == 0 {
                t.Error("expected warning for READONLY mode with irreversible ops")
        }
}

func TestKernelAutoWrapsIrreversibleInCommit(t *testing.T) {
        src := `
mode "CALM"
budget auto
http_get "https://example.com" -> page
say page
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if !strings.Contains(tsrSrc, `commit "KERNEL:auto`) {
                t.Errorf("expected commit block wrapping http_get, got:\n%s", tsrSrc)
        }
}

func TestKernelSetAndSay(t *testing.T) {
        src := `
mode "CALM"
budget auto
set x = 42
say x
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        // Run the compiled TSR
        exitCode, stdout, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode != 0 {
                t.Fatalf("runtime error: %s", stderr)
        }
        if !strings.Contains(stdout, "42") {
                t.Errorf("expected '42' in output, got: %s", stdout)
        }
}

func TestKernelBudgetExplicit(t *testing.T) {
        src := `
mode "CALM"
budget { max_branch: 0, max_loop: 20, max_call_depth: 50, max_commit: 50, max_nest: 20 }
if true {
  say "inside if"
}
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        // Should compile fine but fail at runtime due to max_branch: 0
        exitCode, _, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected runtime error for max_branch: 0")
        }
        if !strings.Contains(stderr, "max_branch") {
                t.Errorf("expected 'max_branch' in error, got: %s", stderr)
        }
}

func TestKernelReadonlyFailsAtRuntime(t *testing.T) {
        src := `
mode "READONLY"
budget auto
http_get "https://example.com" -> page
say page
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        // READONLY mode: http_get without commit should fail at runtime
        exitCode, _, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected runtime failure for http_get in READONLY")
        }
        if !strings.Contains(stderr, "READONLY") {
                t.Errorf("expected 'READONLY' in error, got: %s", stderr)
        }
}

func TestKernelWriteFileWrapped(t *testing.T) {
        src := `
mode "CALM"
budget auto
write_file "kernel_test_out.txt" "test output"
say "done"
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if !strings.Contains(tsrSrc, `commit "KERNEL:auto`) {
                t.Errorf("expected commit block for write_file, got:\n%s", tsrSrc)
        }
        exitCode, stdout, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        os.Remove("kernel_test_out.txt")
        if exitCode != 0 {
                t.Fatalf("runtime error: %s", stderr)
        }
        if !strings.Contains(stdout, "done") {
                t.Errorf("expected 'done', got: %s", stdout)
        }
}

func TestKernelIfStatement(t *testing.T) {
        src := `
mode "CALM"
budget auto
set x = 10
if x {
  say "x is truthy"
}
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        exitCode, stdout, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode != 0 {
                t.Fatalf("runtime error: %s", stderr)
        }
        if !strings.Contains(stdout, "x is truthy") {
                t.Errorf("expected 'x is truthy', got: %s", stdout)
        }
}

func TestKernelAutoBudgetCounts(t *testing.T) {
        // 2 ifs, 1 while, 2 http_gets -> auto budget should reflect these
        src := `
mode "CALM"
budget auto
if true {
  say "a"
}
if true {
  say "b"
}
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        // max_branch should be 2+2=4
        if !strings.Contains(tsrSrc, "max_branch: 4") {
                t.Errorf("expected max_branch: 4 in budget, got:\n%s", tsrSrc)
        }
}
