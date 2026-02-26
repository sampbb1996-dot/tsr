package tsr

import (
	"fmt"
	"net/url"
	"path"
)

const (
	RegimeCalm     = "CALM"
	RegimeReadonly = "READONLY"
)

type StrainBudget struct {
	MaxBranch    int
	MaxLoop      int
	MaxCallDepth int
	MaxCommit    int
	MaxNest      int
}

func defaultBudget() StrainBudget {
	return StrainBudget{
		MaxBranch:    20,
		MaxLoop:      20,
		MaxCallDepth: 50,
		MaxCommit:    50,
		MaxNest:      20,
	}
}

type GovernanceState struct {
	Regime       string
	Budget       StrainBudget
	BudgetSet    bool
	Capabilities map[string][]string
	CommitDepth  int
	NestDepth    int
	BranchCount  int
	LoopCount    int
	CallDepth    int
	CommitCount  int
}

func newGovernanceState() *GovernanceState {
	return &GovernanceState{
		Regime:       RegimeCalm,
		Budget:       defaultBudget(),
		Capabilities: make(map[string][]string),
	}
}

func (g *GovernanceState) addCapability(kind, pattern string) {
	g.Capabilities[kind] = append(g.Capabilities[kind], pattern)
}

// hasCapability checks whether the given target matches any declared pattern
// for the specified capability kind. If no patterns are declared for the kind,
// the capability check is not applied (additive restriction model).
func (g *GovernanceState) hasCapability(kind, target string) bool {
	patterns, ok := g.Capabilities[kind]
	if !ok || len(patterns) == 0 {
		return true
	}
	for _, pat := range patterns {
		if matchCapPattern(pat, target) {
			return true
		}
	}
	return false
}

// matchCapPattern returns true if target matches pattern.
// '*' alone matches any non-empty string.
// All other patterns use path.Match rules (glob-style, '/' is separator).
func matchCapPattern(pattern, target string) bool {
	if pattern == "*" {
		return true
	}
	matched, err := path.Match(pattern, target)
	return err == nil && matched
}

// extractHost pulls the hostname (and port if present) from a URL.
// Falls back to the raw URL on parse error.
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Host
}

// checkAction validates that the given irreversible action is permitted under
// the current regime and capability set.
//
// target is the path (for file ops) or URL (for HTTP ops).
func (g *GovernanceState) checkAction(action, target, file string, line, col int) error {
	switch action {
	case "write_file", "append_file":
		if g.Regime == RegimeReadonly {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q forbidden in regime=%s (commit_depth=%d)",
				file, line, col, action, g.Regime, g.CommitDepth)
		}
		if g.Regime == RegimeCalm && g.CommitDepth < 1 {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q requires commit block in regime=%s (commit_depth=%d)",
				file, line, col, action, g.Regime, g.CommitDepth)
		}
		if !g.hasCapability("file_write", target) {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q denied: path %q does not match any file_write capability",
				file, line, col, action, target)
		}

	case "http_get", "http_post":
		if g.Regime == RegimeReadonly {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q forbidden in regime=%s (commit_depth=%d)",
				file, line, col, action, g.Regime, g.CommitDepth)
		}
		if g.Regime == RegimeCalm && g.CommitDepth < 1 {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q requires commit block in regime=%s (commit_depth=%d)",
				file, line, col, action, g.Regime, g.CommitDepth)
		}
		host := extractHost(target)
		if !g.hasCapability("net", host) {
			return fmt.Errorf("%s:%d:%d: governance error: action=%q denied: host %q does not match any net capability",
				file, line, col, action, host)
		}
	}
	return nil
}

func (g *GovernanceState) enterCommit(file string, line, col int) error {
	g.CommitCount++
	if g.CommitCount > g.Budget.MaxCommit {
		return fmt.Errorf("%s:%d:%d: strain budget exceeded: max_commit=%d actual=%d",
			file, line, col, g.Budget.MaxCommit, g.CommitCount)
	}
	g.CommitDepth++
	return nil
}

func (g *GovernanceState) exitCommit() {
	g.CommitDepth--
}

func (g *GovernanceState) enterNest(file string, line, col int) error {
	g.NestDepth++
	if g.NestDepth > g.Budget.MaxNest {
		return fmt.Errorf("%s:%d:%d: strain budget exceeded: max_nest=%d actual=%d",
			file, line, col, g.Budget.MaxNest, g.NestDepth)
	}
	return nil
}

func (g *GovernanceState) exitNest() {
	g.NestDepth--
}

func (g *GovernanceState) incrementBranch(file string, line, col int) error {
	g.BranchCount++
	if g.BranchCount > g.Budget.MaxBranch {
		return fmt.Errorf("%s:%d:%d: strain budget exceeded: max_branch=%d actual=%d",
			file, line, col, g.Budget.MaxBranch, g.BranchCount)
	}
	return nil
}

func (g *GovernanceState) incrementLoop(file string, line, col int) error {
	g.LoopCount++
	if g.LoopCount > g.Budget.MaxLoop {
		return fmt.Errorf("%s:%d:%d: strain budget exceeded: max_loop=%d actual=%d",
			file, line, col, g.Budget.MaxLoop, g.LoopCount)
	}
	return nil
}

func (g *GovernanceState) enterCall(file string, line, col int) error {
	g.CallDepth++
	if g.CallDepth > g.Budget.MaxCallDepth {
		return fmt.Errorf("%s:%d:%d: strain budget exceeded: max_call_depth=%d actual=%d",
			file, line, col, g.Budget.MaxCallDepth, g.CallDepth)
	}
	return nil
}

func (g *GovernanceState) exitCall() {
	g.CallDepth--
}
