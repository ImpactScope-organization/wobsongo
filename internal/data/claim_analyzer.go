package data

import "context"

// ClaimAnalysis is the result of scoping and decomposing a raw input message,
// before any retrieval happens.
type ClaimAnalysis struct {
	// InScope is false if the input isn't a health-related inquiry at all —
	// the system explicitly doesn't take inquiries unrelated to health.
	InScope bool

	// RefusalReason explains why InScope is false. Empty when InScope is true.
	RefusalReason string

	// SubClaims are the normalized, individually-checkable propositions
	// extracted from the raw input — a claim decomposed from potentially
	// messy source phrasing (e.g. a video transcript sentence) into one or
	// more crisp assertions a retrieval query can match well against. Empty
	// when InScope is false.
	SubClaims []string
}

// ClaimAnalyzer scopes and decomposes a raw input message before retrieval:
// rejects non-health-related input, and splits a compound or loosely-phrased
// claim into normalized sub-claims. Provider-agnostic, same pattern as
// KnowledgeExtractor; see external.ClaimAnalyzerClient for the concrete
// implementation.
type ClaimAnalyzer interface {
	// Analyze returns the scoping/decomposition result for message.
	Analyze(ctx context.Context, message string) (*ClaimAnalysis, error)
}
