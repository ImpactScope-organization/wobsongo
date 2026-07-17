package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"golang.org/x/sync/errgroup"
)

// claimSearchLimit bounds how many fused hybrid-search results feed the
// judge per sub-claim — mirrors cmd/rag.go's ragDefaultLimit.
const claimSearchLimit = 10

// Citation is one piece of evidence a judge verdict cited. Index is the same
// 0-based position the evidence was shown at in the judge's prompt (see
// buildJudgePrompt in external/judge_client.go) — the judge is instructed to
// reference that same index inline in its Reasoning text as "[N]", so a
// caller can link a prose citation directly back to this entry.
type Citation struct {
	Index      int
	Source     string
	Text       string
	ChunkText  string
	TruthTier  string
	DocumentID uuid.UUID
	// Language is the source chunk's/fact's own language, mirroring
	// RAGResult.Language.
	Language model.Language
}

// SubClaimResult is one sub-claim's retrieval + judgment outcome.
type SubClaimResult struct {
	Claim                   string
	Verdict                 model.Verdict
	Severity                model.Severity
	RecommendMedicalConsult bool
	Reasoning               string
	// Citations is the evidence the judge actually cited — empty whenever
	// Verdict is InsufficientEvidence.
	Citations []Citation
}

// ClaimCheckResult is the overall outcome of checking one input message,
// which may decompose into multiple sub-claims.
type ClaimCheckResult struct {
	InScope        bool
	RefusalReason  string
	OverallSummary string
	SubClaims      []SubClaimResult
}

// ClaimService orchestrates claim-checking as a fixed, bounded pipeline —
// not an open-ended agentic loop: scope+decompose the raw input once, then
// for every resulting sub-claim, concurrently retrieve evidence and judge
// it, and aggregate the results.
type ClaimService struct {
	analyzer data.ClaimAnalyzer
	judge    data.ClaimJudge
	rag      *RAGService
}

// NewClaimService is a constructor for ClaimService.
func NewClaimService(
	analyzer data.ClaimAnalyzer,
	judge data.ClaimJudge,
	rag *RAGService,
) *ClaimService {
	return &ClaimService{analyzer: analyzer, judge: judge, rag: rag}
}

// CheckClaim scopes/decomposes req.Text and, if in scope, judges every
// resulting sub-claim concurrently against knowledge-base evidence.
func (s *ClaimService) CheckClaim(
	ctx context.Context,
	req *dto.CheckClaimDTO,
) (*ClaimCheckResult, error) {
	message := req.Text
	analysis, err := s.analyzer.Analyze(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze claim: %w", err)
	}
	if !analysis.InScope {
		return &ClaimCheckResult{InScope: false, RefusalReason: analysis.RefusalReason}, nil
	}

	subClaims := analysis.SubClaims
	if len(subClaims) == 0 {
		// The analyzer said in-scope but produced no decomposition — check
		// the original message directly rather than returning nothing.
		subClaims = []string{message}
	}

	results := make([]SubClaimResult, len(subClaims))
	// errgroup.WithContext, not a plain group: this is a synchronous,
	// user-facing request — a partial result (some sub-claims judged, one
	// errored) is worse than failing the whole request cleanly so the
	// caller can retry, unlike the background document-ingestion workers
	// where one chunk's failure shouldn't cancel its siblings.
	g, gctx := errgroup.WithContext(ctx)
	for i, claim := range subClaims {
		g.Go(func() error {
			result, err := s.checkSubClaim(gctx, claim)
			if err != nil {
				return err
			}
			results[i] = *result
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to check sub-claims: %w", err)
	}

	return &ClaimCheckResult{
		InScope:        true,
		OverallSummary: summarizeVerdicts(results),
		SubClaims:      results,
	}, nil
}

// checkSubClaim retrieves evidence for one sub-claim and judges it. A
// sub-claim with no retrieved evidence never reaches the judge — there's
// nothing to cite, so InsufficientEvidence is returned directly rather than
// spending an LLM call to reach the same conclusion.
func (s *ClaimService) checkSubClaim(ctx context.Context, claim string) (*SubClaimResult, error) {
	hits, err := s.rag.Search(ctx, claim, claimSearchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to search evidence for sub-claim %q: %w", claim, err)
	}

	if len(hits) == 0 {
		return &SubClaimResult{
			Claim:   claim,
			Verdict: model.VerdictInsufficientEvidence,
		}, nil
	}

	evidence := make([]data.JudgeEvidence, len(hits))
	for i, h := range hits {
		evidence[i] = data.JudgeEvidence{
			Source:     h.Source,
			Text:       h.Text,
			ChunkText:  h.ChunkText,
			TruthTier:  h.TruthTier,
			DocumentID: h.DocumentID,
			Language:   h.Language,
		}
	}

	verdict, err := s.judge.Judge(ctx, &data.JudgeRequest{Claim: claim, Evidence: evidence})
	if err != nil {
		return nil, fmt.Errorf("failed to judge sub-claim %q: %w", claim, err)
	}

	citations := make([]Citation, 0, len(verdict.CitedEvidence))
	for _, idx := range verdict.CitedEvidence {
		if idx >= 0 && idx < len(evidence) {
			e := evidence[idx]
			citations = append(citations, Citation{
				Index:      idx,
				Source:     e.Source,
				Text:       e.Text,
				ChunkText:  e.ChunkText,
				TruthTier:  e.TruthTier,
				DocumentID: e.DocumentID,
				Language:   e.Language,
			})
		}
	}

	return &SubClaimResult{
		Claim:                   claim,
		Verdict:                 verdict.Verdict,
		Severity:                verdict.Severity,
		RecommendMedicalConsult: verdict.RecommendMedicalConsult,
		Reasoning:               verdict.Reasoning,
		Citations:               citations,
	}, nil
}

// summarizeVerdicts rolls up per-sub-claim verdicts into one overall
// description of the original message.
func summarizeVerdicts(results []SubClaimResult) string {
	anyContradicted := false
	anyInsufficient := false
	allSupported := true

	for _, r := range results {
		switch r.Verdict {
		case model.VerdictContradicted, model.VerdictMixed:
			anyContradicted = true
			allSupported = false
		case model.VerdictInsufficientEvidence:
			anyInsufficient = true
			allSupported = false
		case model.VerdictPartiallySupported:
			allSupported = false
		}
	}

	switch {
	case anyContradicted:
		return "contains inaccuracies"
	case allSupported:
		return "supported"
	case anyInsufficient:
		return "partially verified — some aspects could not be checked against the knowledge base"
	default:
		return "partially supported"
	}
}
