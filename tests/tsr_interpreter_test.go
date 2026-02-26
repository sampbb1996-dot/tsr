package tests

import (
        "strings"
        "testing"

        "github.com/sampbb1996-dot/tsr/internal/tsr"
)

func TestHello(t *testing.T) {
        src := `say("Hello, World!");`
        exitCode, stdout, stderr := tsr.RunSource(src, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("expected exit 0, got %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "Hello, World!") {
                t.Errorf("expected 'Hello, World!' in output, got: %s", stdout)
        }
}

func TestArithmetic(t *testing.T) {
        code := `
let a = 10;
let b = 3;
say(to_string(a + b));
say(to_string(a - b));
say(to_string(a * b));
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("expected exit 0, got %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "13") {
                t.Errorf("expected 13, got: %s", stdout)
        }
        if !strings.Contains(stdout, "7") {
                t.Errorf("expected 7, got: %s", stdout)
        }
        if !strings.Contains(stdout, "30") {
                t.Errorf("expected 30, got: %s", stdout)
        }
}

func TestIfElse(t *testing.T) {
        code := `
let x = 5;
if (x > 3) {
  say("big");
} else {
  say("small");
}
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "big") {
                t.Errorf("expected 'big', got: %s", stdout)
        }
}

func TestWhileLoop(t *testing.T) {
        code := `
let i = 0;
let sum = 0;
while (i < 5) {
  sum = sum + i;
  i = i + 1;
}
say(to_string(sum));
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "10") {
                t.Errorf("expected '10', got: %s", stdout)
        }
}

func TestFunctions(t *testing.T) {
        code := `
fn double(n) {
  return n * 2;
}
say(to_string(double(21)));
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "42") {
                t.Errorf("expected '42', got: %s", stdout)
        }
}

func TestListsAndDicts(t *testing.T) {
        code := `
let items = [1, 2, 3];
say(to_string(items[0]));
say(to_string(len(items)));

let d = {"key": "value"};
say(d["key"]);
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "1") {
                t.Errorf("expected '1', got: %s", stdout)
        }
        if !strings.Contains(stdout, "3") {
                t.Errorf("expected '3', got: %s", stdout)
        }
        if !strings.Contains(stdout, "value") {
                t.Errorf("expected 'value', got: %s", stdout)
        }
}

func TestJsonRoundtrip(t *testing.T) {
        code := `
let obj = {"name": "TSR", "version": "1"};
let s = json_stringify(obj);
let parsed = json_parse(s);
say(parsed["name"]);
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "TSR") {
                t.Errorf("expected 'TSR', got: %s", stdout)
        }
}

func TestRecursion(t *testing.T) {
        code := `
fn fib(n) {
  if (n <= 1) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}
say(to_string(fib(5)));
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode != 0 {
                t.Fatalf("exit %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "5") {
                t.Errorf("expected '5' (fib(5)), got: %s", stdout)
        }
}

// wrapInFile is a no-op helper kept for potential future use
func wrapInFile(s string) string { return s }
