package policy

import "strings"

// Context represents the explicit environmental state under which a program runs.
// No context → default policy applied.
type Context struct {
	Env   string
	Risk  string
	Actor string
}

// StrainConfig mirrors tsr.StrainBudget for use in the policy layer.
type StrainConfig struct {
	MaxBranch    int
	MaxLoop      int
	MaxCallDepth int
	MaxCommit    int
	MaxNest      int
}

// CapabilityConfig lists the allowed target patterns per capability kind.
// Empty slice → no capability declared (no additional restriction beyond regime).
type CapabilityConfig struct {
	FileWrite []string
	Net       []string
}

// PolicyResult is the fully resolved governance configuration for a given context.
type PolicyResult struct {
	Regime       string
	Strain       StrainConfig
	Capabilities CapabilityConfig
}

// Evaluate applies governance policy rules in strict, fixed precedence order.
// The rules are deterministic: same context always produces the same result.
//
// Precedence:
//  1. risk == "high"          → READONLY, tight strain, no capabilities
//  2. actor contains "admin"  → CALM, wide strain, full capabilities
//  3. env == "prod"           → CALM, moderate strain, restricted capabilities
//  4. default                 → CALM, standard strain, full capabilities
func Evaluate(ctx Context) PolicyResult {
	if ctx.Risk == "high" {
		return PolicyResult{
			Regime: "READONLY",
			Strain: StrainConfig{
				MaxBranch:    5,
				MaxLoop:      5,
				MaxCallDepth: 10,
				MaxCommit:    0,
				MaxNest:      3,
			},
			Capabilities: CapabilityConfig{},
		}
	}

	if strings.Contains(ctx.Actor, "admin") {
		return PolicyResult{
			Regime: "CALM",
			Strain: StrainConfig{
				MaxBranch:    50,
				MaxLoop:      50,
				MaxCallDepth: 100,
				MaxCommit:    50,
				MaxNest:      20,
			},
			Capabilities: CapabilityConfig{
				FileWrite: []string{"*"},
				Net:       []string{"*"},
			},
		}
	}

	if ctx.Env == "prod" {
		return PolicyResult{
			Regime: "CALM",
			Strain: StrainConfig{
				MaxBranch:    10,
				MaxLoop:      10,
				MaxCallDepth: 25,
				MaxCommit:    5,
				MaxNest:      5,
			},
			Capabilities: CapabilityConfig{
				FileWrite: []string{"logs/*"},
				Net:       []string{"*.internal"},
			},
		}
	}

	return PolicyResult{
		Regime: "CALM",
		Strain: StrainConfig{
			MaxBranch:    20,
			MaxLoop:      20,
			MaxCallDepth: 50,
			MaxCommit:    50,
			MaxNest:      20,
		},
		Capabilities: CapabilityConfig{
			FileWrite: []string{"*"},
			Net:       []string{"*"},
		},
	}
}
