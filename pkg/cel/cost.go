package cel

import (
	"fmt"

	"github.com/google/cel-go/checker"
)

// DefaultCostBudget is the default cost budget for a single expression
// evaluation. Aligned with Kubernetes defaults: enough for typical identity
// operations but prevents runaway expressions.
const DefaultCostBudget uint64 = 10_000_000

// MaxExpressionLength is the maximum length of a CEL expression string.
const MaxExpressionLength = 10_240

// DefaultStringMaxLength is the estimated max length of string values
// (emails, usernames, group names, etc.) used for compile-time cost estimation.
const DefaultStringMaxLength = 256

// DefaultListMaxLength is the estimated max length of list values
// (groups, scopes) used for compile-time cost estimation.
const DefaultListMaxLength = 100

// CostEstimate holds the estimated cost range for a compiled expression.
type CostEstimate struct {
	Min uint64
	Max uint64
}

// EstimateCost returns the estimated cost range for a compiled expression.
// This is computed statically at compile time without evaluating the expression.
func (c *Compiler) EstimateCost(result *CompilationResult) (CostEstimate, error) {
	costEst, err := c.env.EstimateCost(result.ast, &defaultCostEstimator{})
	if err != nil {
		return CostEstimate{}, fmt.Errorf("CEL cost estimation failed: %w", err)
	}

	return CostEstimate{Min: costEst.Min, Max: costEst.Max}, nil
}

// defaultCostEstimator provides size hints for compile-time cost estimation.
// Without these hints, the CEL cost estimator assumes unbounded sizes for
// variables, leading to wildly overestimated max costs.
type defaultCostEstimator struct{}

func (defaultCostEstimator) EstimateSize(element checker.AstNode) *checker.SizeEstimate {
	// Provide size hints for map(string, dyn) variables: identity, request, claims.
	// Without these, the estimator assumes lists/strings can be infinitely large.
	if element.Path() == nil {
		return nil
	}

	path := element.Path()
	if len(path) == 0 {
		return nil
	}

	root := path[0]

	switch root {
	case "identity", "request", "claims":
		// Nested field access (e.g. identity.email, identity.groups)
		if len(path) >= 2 {
			field := path[1]
			switch field {
			case "groups", "scopes":
				// list(string) fields
				return &checker.SizeEstimate{Min: 0, Max: DefaultListMaxLength}
			case "email_verified":
				// bool field — size is always 1
				return &checker.SizeEstimate{Min: 1, Max: 1}
			default:
				// string fields (email, username, user_id, client_id, etc.)
				return &checker.SizeEstimate{Min: 0, Max: DefaultStringMaxLength}
			}
		}
		// The map itself: number of keys
		return &checker.SizeEstimate{Min: 0, Max: 20}
	}

	return nil
}

func (defaultCostEstimator) EstimateCallCost(function, overloadID string, target *checker.AstNode, args []checker.AstNode) *checker.CallEstimate {
	switch function {
	case "dex.emailDomain", "dex.emailLocalPart":
		// Simple string split — O(n) where n is string length, bounded.
		return &checker.CallEstimate{
			CostEstimate: checker.CostEstimate{Min: 1, Max: 2},
		}
	case "dex.groupMatches":
		// Iterates over groups list and matches each against a pattern.
		return &checker.CallEstimate{
			CostEstimate: checker.CostEstimate{Min: 1, Max: DefaultListMaxLength},
		}
	case "dex.groupFilter":
		// Builds a set from allowed list, then iterates groups.
		return &checker.CallEstimate{
			CostEstimate: checker.CostEstimate{Min: 1, Max: 2 * DefaultListMaxLength},
		}
	}

	return nil
}
