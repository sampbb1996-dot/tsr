package tests

import (
        "os"
        "strings"
        "testing"

        "github.com/sampbb1996-dot/tsr/internal/kernel"
        "github.com/sampbb1996-dot/tsr/internal/policy"
        "github.com/sampbb1996-dot/tsr/internal/tsr"
)

// ---------- Policy engine unit tests ----------

func TestPolicyHighRiskIsReadonly(t *testing.T) {
        r := policy.Evaluate(policy.Context{Env: "prod", Risk: "high", Actor: "service:billing"})
        if r.Regime != "READONLY" {
                t.Errorf("expected READONLY for high risk, got %s", r.Regime)
        }
        if r.Strain.MaxCommit != 0 {
                t.Errorf("expected MaxCommit=0 for high risk, got %d", r.Strain.MaxCommit)
        }
        if len(r.Capabilities.FileWrite) != 0 || len(r.Capabilities.Net) != 0 {
                t.Error("expected no capabilities for high risk")
        }
}

func TestPolicyAdminIsCalm(t *testing.T) {
        r := policy.Evaluate(policy.Context{Env: "dev", Risk: "low", Actor: "admin:root"})
        if r.Regime != "CALM" {
                t.Errorf("expected CALM for admin actor, got %s", r.Regime)
        }
        if len(r.Capabilities.FileWrite) == 0 {
                t.Error("expected file_write capabilities for admin")
        }
        if r.Capabilities.FileWrite[0] != "*" {
                t.Errorf("expected '*' file_write capability for admin, got %s", r.Capabilities.FileWrite[0])
        }
}

func TestPolicyProdHasRestrictedCapabilities(t *testing.T) {
        r := policy.Evaluate(policy.Context{Env: "prod", Risk: "low", Actor: "service:api"})
        if r.Regime != "CALM" {
                t.Errorf("expected CALM for prod, got %s", r.Regime)
        }
        if len(r.Capabilities.FileWrite) == 0 || r.Capabilities.FileWrite[0] != "logs/*" {
                t.Errorf("expected 'logs/*' file_write for prod, got %v", r.Capabilities.FileWrite)
        }
        if len(r.Capabilities.Net) == 0 || r.Capabilities.Net[0] != "*.internal" {
                t.Errorf("expected '*.internal' net for prod, got %v", r.Capabilities.Net)
        }
}

func TestPolicyDefaultIsCalm(t *testing.T) {
        r := policy.Evaluate(policy.Context{Env: "dev", Risk: "low", Actor: "service:test"})
        if r.Regime != "CALM" {
                t.Errorf("expected CALM for default, got %s", r.Regime)
        }
        if r.Strain.MaxBranch != 20 {
                t.Errorf("expected MaxBranch=20 for default, got %d", r.Strain.MaxBranch)
        }
}

func TestPolicyPrecedenceHighRiskBeforeAdmin(t *testing.T) {
        // risk=high takes precedence even if actor contains "admin"
        r := policy.Evaluate(policy.Context{Env: "dev", Risk: "high", Actor: "admin:superuser"})
        if r.Regime != "READONLY" {
                t.Errorf("expected READONLY: high risk must outrank admin, got %s", r.Regime)
        }
}

// ---------- Capability enforcement tests ----------

func TestCapabilityFileWriteAllowed(t *testing.T) {
        code := `
regime "CALM";
capability file_write "*";
commit "ok" {
  write_file("cap_test_out.txt", "ok");
}
say("done");
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        _ = removeIfExists("cap_test_out.txt")
        if exitCode != 0 {
                t.Fatalf("expected exit 0, got %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "done") {
                t.Errorf("expected 'done', got: %s", stdout)
        }
}

func TestCapabilityFileWriteDeniedByPattern(t *testing.T) {
        code := `
regime "CALM";
capability file_write "logs/*";
commit "try" {
  write_file("secret.txt", "blocked");
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit: path not matching logs/* should be denied")
        }
        if !strings.Contains(stderr, "file_write capability") {
                t.Errorf("expected capability error, got: %s", stderr)
        }
}

func TestCapabilityFileWritePatternAllows(t *testing.T) {
        code := `
regime "CALM";
capability file_write "cap_logs_*";
commit "ok" {
  write_file("cap_logs_app.txt", "entry");
}
say("logged");
`
        exitCode, stdout, stderr := tsr.RunSource(code, "test.tsr", false)
        _ = removeIfExists("cap_logs_app.txt")
        if exitCode != 0 {
                t.Fatalf("expected exit 0, got %d: %s", exitCode, stderr)
        }
        if !strings.Contains(stdout, "logged") {
                t.Errorf("expected 'logged', got: %s", stdout)
        }
}

func TestCapabilityReadonlyForbidsRegardlessOfCapability(t *testing.T) {
        code := `
regime "READONLY";
capability file_write "*";
commit "try" {
  write_file("cap_test_out.txt", "blocked");
}
`
        exitCode, _, stderr := tsr.RunSource(code, "test.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected non-zero exit: READONLY forbids all writes regardless of capability")
        }
        if !strings.Contains(stderr, "READONLY") {
                t.Errorf("expected READONLY error, got: %s", stderr)
        }
}

// ---------- KERNEL context block tests ----------

func TestKernelContextHighRiskEmitsReadonly(t *testing.T) {
        src := `
context {
  env: "prod"
  risk: "high"
  actor: "service:billing"
}
budget auto
say "hello"
`
        tsrSrc, _, policyTrace, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if !strings.Contains(tsrSrc, `regime "READONLY"`) {
                t.Errorf("expected READONLY regime from high-risk context, got:\n%s", tsrSrc)
        }
        if !strings.Contains(policyTrace, "regime=READONLY") {
                t.Errorf("expected regime=READONLY in policy trace, got:\n%s", policyTrace)
        }
        if !strings.Contains(policyTrace, "risk=high") {
                t.Errorf("expected risk=high in policy trace, got:\n%s", policyTrace)
        }
}

func TestKernelContextProdEmitsRestrictedCapabilities(t *testing.T) {
        src := `
context {
  env: "prod"
  risk: "low"
  actor: "service:api"
}
budget auto
say "prod ready"
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if !strings.Contains(tsrSrc, `capability file_write "logs/*"`) {
                t.Errorf("expected logs/* file_write capability in prod, got:\n%s", tsrSrc)
        }
        if !strings.Contains(tsrSrc, `capability net "*.internal"`) {
                t.Errorf("expected *.internal net capability in prod, got:\n%s", tsrSrc)
        }
}

func TestKernelContextHighRiskFailsAtRuntime(t *testing.T) {
        src := `
context {
  env: "prod"
  risk: "high"
  actor: "service:billing"
}
budget auto
write_file "results.txt" "blocked"
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        exitCode, _, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected runtime failure: READONLY forbids write_file")
        }
        if !strings.Contains(stderr, "READONLY") {
                t.Errorf("expected READONLY error, got: %s", stderr)
        }
}

func TestKernelContextNoCapabilityBlocksWrite(t *testing.T) {
        // prod context: file_write limited to logs/*
        // write to secret.txt should fail capability check
        src := `
context {
  env: "prod"
  risk: "low"
  actor: "service:api"
}
budget auto
write_file "secret.txt" "blocked"
`
        tsrSrc, _, _, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        exitCode, _, stderr := tsr.RunSource(tsrSrc, "compiled.tsr", false)
        if exitCode == 0 {
                t.Fatal("expected runtime failure: secret.txt not in logs/*")
        }
        if !strings.Contains(stderr, "file_write capability") {
                t.Errorf("expected capability error, got: %s", stderr)
        }
}

func TestKernelContextPolicyTrace(t *testing.T) {
        src := `
context {
  env: "dev"
  risk: "low"
  actor: "service:test"
}
budget auto
say "trace test"
`
        _, _, policyTrace, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if policyTrace == "" {
                t.Error("expected non-empty policy trace for context block")
        }
        if !strings.Contains(policyTrace, "[policy]") {
                t.Errorf("expected [policy] header in trace, got:\n%s", policyTrace)
        }
        if !strings.Contains(policyTrace, "env=dev") {
                t.Errorf("expected env=dev in trace, got:\n%s", policyTrace)
        }
}

func TestNoPolicyTraceWithoutContext(t *testing.T) {
        src := `
mode "CALM"
budget auto
say "no context"
`
        _, _, policyTrace, err := kernel.CompileSource(src, "test.krn")
        if err != nil {
                t.Fatalf("compile error: %s", err)
        }
        if policyTrace != "" {
                t.Errorf("expected empty policy trace without context block, got:\n%s", policyTrace)
        }
}

// helper
func removeIfExists(path string) error {
        return os.Remove(path)
}
