package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// NarrativeService stores claims and groups them into narratives.
type NarrativeService struct {
	repo           data.NarrativeRepoer
	embedder       data.Embedder
	generator      data.NarrativeStatementGenerator
	matchThreshold float64
}

// NewNarrativeService creates a new NarrativeService. matchThreshold is the
// minimum cosine similarity for a claim to join an existing narrative.
func NewNarrativeService(
	repo data.NarrativeRepoer,
	embedder data.Embedder,
	generator data.NarrativeStatementGenerator,
	matchThreshold float64,
) *NarrativeService {
	return &NarrativeService{
		repo:           repo,
		embedder:       embedder,
		generator:      generator,
		matchThreshold: matchThreshold,
	}
}

// ActiveTopics returns the closed topic list offered to the claim analyzer.
func (s *NarrativeService) ActiveTopics(ctx context.Context) ([]data.AnalyzerTopic, error) {
	topics, err := s.repo.ListActiveTopics(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]data.AnalyzerTopic, len(topics))
	for i, t := range topics {
		out[i] = data.AnalyzerTopic{ID: t.ID, Name: t.Name, Description: t.Description}
	}
	return out, nil
}

// RecordClaims stores claims and enqueues one cluster job per newly stored
// claim, all in one transaction. A claim that already exists is skipped.
func (s *NarrativeService) RecordClaims(ctx context.Context, claims []model.Claim) error {
	return s.repo.WithTx(ctx, func(r data.NarrativeRepoer) error {
		for i := range claims {
			created, err := r.CreateClaim(ctx, &claims[i])
			if errors.Is(err, data.ErrConflict) {
				continue
			}
			if err != nil {
				return err
			}
			if err := r.EnqueueClusterClaim(ctx, created.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// ClusterClaim groups one stored claim into a narrative of its topic: it
// joins the closest existing narrative when similar enough, otherwise it
// starts a new one. Claims of one topic are handled one at a time so two
// similar claims cannot both start a narrative.
func (s *NarrativeService) ClusterClaim(ctx context.Context, claimID uuid.UUID) error {
	claim, err := s.repo.GetClaim(ctx, claimID)
	if err != nil {
		return fmt.Errorf("failed to load claim %s: %w", claimID, err)
	}

	return s.repo.WithTx(ctx, func(r data.NarrativeRepoer) error {
		if err := r.LockTopic(ctx, claim.TopicID); err != nil {
			return err
		}

		narratives, err := r.ListNarrativesByTopic(ctx, claim.TopicID)
		if err != nil {
			return err
		}

		narrative, found, err := s.findMatch(ctx, claim, narratives)
		if err != nil {
			return err
		}

		if !found {
			statement, err := s.generator.Generate(ctx, claim.Text)
			if err != nil {
				return fmt.Errorf("failed to write narrative statement: %w", err)
			}
			narrative, err = r.CreateNarrative(ctx, &model.Narrative{
				TopicID:   claim.TopicID,
				Statement: statement,
			})
			if err != nil {
				return err
			}
			log.Printf("[NarrativeService] new narrative %s from claim %s", narrative.ID, claim.ID)
		}

		if err := r.LinkClaim(ctx, claim.ID, narrative.ID); err != nil {
			return err
		}
		return s.refreshNarrative(ctx, r, narrative.ID)
	})
}

// DeactivateStale deactivates narratives without a claim inside window.
func (s *NarrativeService) DeactivateStale(
	ctx context.Context,
	window time.Duration,
) (int64, error) {
	return s.repo.DeactivateStaleNarratives(ctx, time.Now().Add(-window))
}

// findMatch embeds the claim and the narrative statements (nothing is stored)
// and returns the narrative closest to the claim, or false when none reaches
// the threshold.
func (s *NarrativeService) findMatch(
	ctx context.Context,
	claim *model.Claim,
	narratives []model.Narrative,
) (*model.Narrative, bool, error) {
	if len(narratives) == 0 {
		return nil, false, nil
	}

	texts := make([]string, 0, len(narratives)+1)
	texts = append(texts, claim.Text)
	for i := range narratives {
		texts = append(texts, narratives[i].Statement)
	}

	vectors, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, false, fmt.Errorf("failed to embed claim and narratives: %w", err)
	}
	if len(vectors) != len(texts) {
		return nil, false, fmt.Errorf(
			"embedder returned %d vectors for %d texts",
			len(vectors),
			len(texts),
		)
	}

	best, bestScore := -1, 0.0
	for i := range narratives {
		score := cosineSimilarity(vectors[0], vectors[i+1])
		if best == -1 || score > bestScore {
			best, bestScore = i, score
		}
	}

	log.Printf(
		"[NarrativeService] claim %s best narrative score=%.3f threshold=%.3f",
		claim.ID, bestScore, s.matchThreshold,
	)
	if bestScore < s.matchThreshold {
		return nil, false, nil
	}
	return &narratives[best], true, nil
}

// List returns one page of narratives, newest first.
func (s *NarrativeService) List(
	ctx context.Context,
	pagination *dto.PaginationDTO,
) (*dto.PaginationResults[model.Narrative], error) {
	return s.repo.PaginateNarratives(ctx, pagination)
}

// GetByID returns the narrative with the given id.
func (s *NarrativeService) GetByID(ctx context.Context, id uuid.UUID) (*model.Narrative, error) {
	return s.repo.GetNarrative(ctx, id)
}

// refreshNarrative recomputes the harm estimate and status of a narrative from
// its newest claim and marks it active. The repo keeps an expert-validated
// harm and an expert_verified status as they are.
func (s *NarrativeService) refreshNarrative(
	ctx context.Context,
	r data.NarrativeRepoer,
	narrativeID uuid.UUID,
) error {
	newest, err := r.GetNewestClaim(ctx, narrativeID)
	if err != nil {
		return err
	}
	return r.UpdateNarrativeEstimate(
		ctx,
		narrativeID,
		estimateHarm(newest),
		narrativeStatusFor(newest),
	)
}

// estimateHarm derives a harm level (1-5) from the claim's severity and
// high-risk flag: routine 1, serious 3, emergency 5, and at least 4 when the
// claim is high-risk. It is only an estimate until an expert validates it.
func estimateHarm(claim *model.Claim) *int {
	harm := model.NarrativeHarmMin
	switch claim.Severity {
	case model.SeveritySerious:
		harm = 3
	case model.SeverityEmergency:
		harm = model.NarrativeHarmMax
	case model.SeverityRoutine:
		// Keeps the minimum.
	}
	if claim.HighRisk && harm < 4 {
		harm = 4
	}
	return &harm
}

// narrativeStatusFor maps the newest claim's verdict to the narrative status:
// no evidence in the knowledge base is not_in_kb, anything else auto_checked.
func narrativeStatusFor(claim *model.Claim) model.NarrativeStatus {
	if claim.Verdict == model.VerdictInsufficientEvidence {
		return model.NarrativeStatusNotInKB
	}
	return model.NarrativeStatusAutoChecked
}

// cosineSimilarity returns the cosine similarity of two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		normA += x * x
		normB += y * y
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
