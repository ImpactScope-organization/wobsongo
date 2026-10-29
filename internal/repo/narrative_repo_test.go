package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// noopClusterClaimWorker satisfies River's client-side check that a job's
// kind is registered in the Workers bundle before it can be inserted. These
// tests never start the client, so Work is never actually invoked.
type noopClusterClaimWorker struct {
	river.WorkerDefaults[queue.ClusterClaimJob]
}

func (*noopClusterClaimWorker) Work(context.Context, *river.Job[queue.ClusterClaimJob]) error {
	return nil
}

// narrativeTestEnv is one transaction that is always rolled back when the
// test ends, so each test sees a clean view of the database. The raw tx is
// exposed for the few setups the repo has no method for (e.g. marking a
// narrative as expert verified).
type narrativeTestEnv struct {
	ctx  context.Context
	tx   pgx.Tx
	q    *db.Queries
	repo data.NarrativeRepoer
}

func newNarrativeTestEnv(t *testing.T, pool *pgxpool.Pool) *narrativeTestEnv {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })

	q := db.New(tx)
	return &narrativeTestEnv{
		ctx:  ctx,
		tx:   tx,
		q:    q,
		repo: repo.NewNarrativeRepo(q, pool, nil),
	}
}

func (e *narrativeTestEnv) newTopic(t *testing.T, active bool) db.Topic {
	t.Helper()

	topic, err := e.q.UpsertTopicByName(e.ctx, db.UpsertTopicByNameParams{
		Name:        "test-topic-" + uuid.NewString(),
		Description: "A test topic",
		IsActive:    active,
	})
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	return topic
}

func (e *narrativeTestEnv) newNarrative(t *testing.T, topicID uuid.UUID) *model.Narrative {
	t.Helper()

	narrative, err := e.repo.CreateNarrative(e.ctx, &model.Narrative{
		TopicID:   topicID,
		Statement: "narrative " + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("failed to create narrative: %v", err)
	}
	return narrative
}

func (e *narrativeTestEnv) newClaim(
	t *testing.T,
	topicID uuid.UUID,
	occurredAt time.Time,
) *model.Claim {
	t.Helper()

	claim, err := e.repo.CreateClaim(
		e.ctx,
		newNarrativeTestClaim(topicID, "claim "+uuid.NewString(), occurredAt),
	)
	if err != nil {
		t.Fatalf("failed to create claim: %v", err)
	}
	return claim
}

// newNarrativeTestClaim builds a claim value that is not stored yet.
// Timestamps are truncated to microseconds, the precision Postgres keeps.
func newNarrativeTestClaim(topicID uuid.UUID, text string, occurredAt time.Time) *model.Claim {
	at := occurredAt.UTC().Truncate(time.Microsecond)
	return &model.Claim{
		Text:       text,
		TopicID:    topicID,
		Language:   model.LanguageFrench,
		Platform:   model.ClaimPlatformWhatsApp,
		Format:     model.ClaimFormatText,
		OccurredAt: at,
		Verdict:    model.VerdictContradicted,
		Severity:   model.SeveritySerious,
		JudgedAt:   at,
	}
}

func TestNarrativeRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ListActiveTopics_ExcludesInactive", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		active := env.newTopic(t, true)
		inactive := env.newTopic(t, false)

		topics, err := env.repo.ListActiveTopics(env.ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var foundActive, foundInactive bool
		for i := range topics {
			switch topics[i].ID {
			case active.ID:
				foundActive = true
			case inactive.ID:
				foundInactive = true
			}
		}
		if !foundActive {
			t.Errorf("expected active topic %s in the list", active.ID)
		}
		if foundInactive {
			t.Errorf("expected inactive topic %s to be left out", inactive.ID)
		}
	})

	t.Run("CreateClaim_RoundTripsFields", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)

		claim := newNarrativeTestClaim(topic.ID, "vitamin C cures colds", time.Now())
		claim.HighRisk = true
		claim.HighRiskReason = "self-medication"
		claim.Reasoning = "the cited chunk contradicts it [0]"
		claim.BriefReasoning = "contradicted"
		claim.Citations = []model.ClaimCitation{{
			Index:      0,
			Source:     "chunk",
			DocumentID: uuid.New(),
			ChunkID:    uuid.New(),
			Text:       "evidence text",
		}}

		created, err := env.repo.CreateClaim(env.ctx, claim)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID == uuid.Nil {
			t.Error("expected a generated claim ID")
		}

		got, err := env.repo.GetClaim(env.ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching created claim: %v", err)
		}
		if got.Text != claim.Text || got.TopicID != claim.TopicID {
			t.Errorf("fetched claim does not match created one: %+v vs %+v", got, claim)
		}
		if got.Verdict != claim.Verdict || got.Severity != claim.Severity {
			t.Errorf(
				"expected verdict %v and severity %v, got %v and %v",
				claim.Verdict, claim.Severity, got.Verdict, got.Severity,
			)
		}
		if !got.HighRisk || got.HighRiskReason != claim.HighRiskReason {
			t.Errorf("expected the high-risk flag and reason to round-trip, got %+v", got)
		}
		if !got.OccurredAt.Equal(claim.OccurredAt) {
			t.Errorf("expected occurred_at %v, got %v", claim.OccurredAt, got.OccurredAt)
		}
		if len(got.Citations) != 1 || got.Citations[0] != claim.Citations[0] {
			t.Errorf("expected citations %+v, got %+v", claim.Citations, got.Citations)
		}
	})

	t.Run("CreateClaim_NilCitationsStoredAsEmpty", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)

		created, err := env.repo.CreateClaim(
			env.ctx,
			newNarrativeTestClaim(topic.ID, "no citations", time.Now()),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := env.repo.GetClaim(env.ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching created claim: %v", err)
		}
		if len(got.Citations) != 0 {
			t.Errorf("expected no citations, got %+v", got.Citations)
		}
	})

	t.Run("CreateClaim_SameMessageAndText_ReturnsConflict", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		message, err := env.q.InsertConversationMessage(env.ctx, db.InsertConversationMessageParams{
			Jid:     "test-" + uuid.NewString(),
			Role:    "user",
			Content: "hello",
		})
		if err != nil {
			t.Fatalf("failed to create message: %v", err)
		}

		claim := newNarrativeTestClaim(topic.ID, "duplicated claim", time.Now())
		claim.MessageID = &message.ID
		_, err = env.repo.CreateClaim(env.ctx, claim)
		if err != nil {
			t.Fatalf("unexpected error on first create: %v", err)
		}

		_, err = env.repo.CreateClaim(env.ctx, claim)
		if !errors.Is(err, data.ErrConflict) {
			t.Errorf("expected data.ErrConflict, got %v", err)
		}
	})

	t.Run("GetClaim_NotFound", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)

		_, err := env.repo.GetClaim(env.ctx, uuid.New())
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})

	t.Run("CreateNarrative_StartsActiveAndNeedsCheck", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)

		created := env.newNarrative(t, topic.ID)

		if created.ID == uuid.Nil {
			t.Error("expected a generated narrative ID")
		}
		if created.Status != model.NarrativeStatusNeedsCheck {
			t.Errorf("expected status %q, got %q", model.NarrativeStatusNeedsCheck, created.Status)
		}
		if !created.IsActive {
			t.Error("expected a new narrative to be active")
		}
		if created.HarmValidated || created.Harm != nil {
			t.Errorf("expected no harm estimate yet, got %+v", created)
		}

		got, err := env.repo.GetNarrative(env.ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching created narrative: %v", err)
		}
		if got.ID != created.ID || got.Statement != created.Statement || got.TopicID != topic.ID {
			t.Errorf("fetched narrative does not match created one: %+v vs %+v", got, created)
		}
	})

	t.Run("GetNarrative_NotFound", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)

		_, err := env.repo.GetNarrative(env.ctx, uuid.New())
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})

	t.Run("ListNarrativesByTopic_OnlyReturnsOwnTopic", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topicA := env.newTopic(t, true)
		topicB := env.newTopic(t, true)
		empty := env.newTopic(t, true)
		a1 := env.newNarrative(t, topicA.ID)
		a2 := env.newNarrative(t, topicA.ID)
		env.newNarrative(t, topicB.ID)

		got, err := env.repo.ListNarrativesByTopic(env.ctx, topicA.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ids := make(map[uuid.UUID]bool, len(got))
		for i := range got {
			ids[got[i].ID] = true
		}
		if len(got) != 2 || !ids[a1.ID] || !ids[a2.ID] {
			t.Errorf("expected narratives %s and %s, got %+v", a1.ID, a2.ID, got)
		}

		none, err := env.repo.ListNarrativesByTopic(env.ctx, empty.ID)
		if err != nil {
			t.Fatalf("unexpected error for a topic without narratives: %v", err)
		}
		if len(none) != 0 {
			t.Errorf("expected no narratives, got %+v", none)
		}
	})

	t.Run("LinkClaim_IsIdempotent", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)
		claim := env.newClaim(t, topic.ID, time.Now())

		err := env.repo.LinkClaim(env.ctx, claim.ID, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error on first link: %v", err)
		}
		err = env.repo.LinkClaim(env.ctx, claim.ID, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error linking the same pair twice: %v", err)
		}

		var links int
		err = env.tx.QueryRow(
			env.ctx,
			"select count(*) from narrative_claims where claim_id = $1 and narrative_id = $2",
			claim.ID, narrative.ID,
		).Scan(&links)
		if err != nil {
			t.Fatalf("failed to count links: %v", err)
		}
		if links != 1 {
			t.Errorf("expected exactly 1 link, got %d", links)
		}
	})

	t.Run("GetNewestClaim_ReturnsGreatestOccurredAt", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)
		now := time.Now()
		older := env.newClaim(t, topic.ID, now.Add(-2*time.Hour))
		newer := env.newClaim(t, topic.ID, now.Add(-1*time.Hour))
		for _, claim := range []*model.Claim{newer, older} {
			err := env.repo.LinkClaim(env.ctx, claim.ID, narrative.ID)
			if err != nil {
				t.Fatalf("unexpected error linking claim: %v", err)
			}
		}

		got, err := env.repo.GetNewestClaim(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.ID != newer.ID {
			t.Errorf("expected the newest claim %s, got %s", newer.ID, got.ID)
		}
	})

	t.Run("GetNewestClaim_NoClaims_ReturnsNotFound", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)

		_, err := env.repo.GetNewestClaim(env.ctx, narrative.ID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})

	t.Run("UpdateNarrativeEstimate_SetsHarmStatusAndReactivates", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)
		_, err := env.tx.Exec(
			env.ctx,
			"update narratives set is_active = false where id = $1",
			narrative.ID,
		)
		if err != nil {
			t.Fatalf("failed to deactivate narrative: %v", err)
		}

		harm := 2
		err = env.repo.UpdateNarrativeEstimate(
			env.ctx, narrative.ID, &harm, model.NarrativeStatusAutoChecked,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := env.repo.GetNarrative(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching narrative: %v", err)
		}
		if got.Harm == nil || *got.Harm != 2 {
			t.Errorf("expected harm 2, got %v", got.Harm)
		}
		if got.Status != model.NarrativeStatusAutoChecked {
			t.Errorf("expected status %q, got %q", model.NarrativeStatusAutoChecked, got.Status)
		}
		if !got.IsActive {
			t.Error("expected the narrative to be active again")
		}
	})

	t.Run("UpdateNarrativeEstimate_NilHarmClearsEstimate", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)
		harm := 4
		err := env.repo.UpdateNarrativeEstimate(
			env.ctx, narrative.ID, &harm, model.NarrativeStatusAutoChecked,
		)
		if err != nil {
			t.Fatalf("unexpected error setting harm: %v", err)
		}

		err = env.repo.UpdateNarrativeEstimate(
			env.ctx, narrative.ID, nil, model.NarrativeStatusNotInKB,
		)
		if err != nil {
			t.Fatalf("unexpected error clearing harm: %v", err)
		}

		got, err := env.repo.GetNarrative(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching narrative: %v", err)
		}
		if got.Harm != nil {
			t.Errorf("expected no harm estimate, got %d", *got.Harm)
		}
		if got.Status != model.NarrativeStatusNotInKB {
			t.Errorf("expected status %q, got %q", model.NarrativeStatusNotInKB, got.Status)
		}
	})

	t.Run("UpdateNarrativeEstimate_KeepsExpertValues", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)
		_, err := env.tx.Exec(
			env.ctx,
			"update narratives set harm = 5, harm_validated = true, status = 'expert_verified' where id = $1",
			narrative.ID,
		)
		if err != nil {
			t.Fatalf("failed to mark narrative as expert verified: %v", err)
		}

		harm := 1
		err = env.repo.UpdateNarrativeEstimate(
			env.ctx, narrative.ID, &harm, model.NarrativeStatusAutoChecked,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := env.repo.GetNarrative(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching narrative: %v", err)
		}
		if got.Harm == nil || *got.Harm != 5 || !got.HarmValidated {
			t.Errorf("expected the validated harm 5 to be kept, got %+v", got)
		}
		if got.Status != model.NarrativeStatusExpertVerified {
			t.Errorf(
				"expected status %q to be kept, got %q",
				model.NarrativeStatusExpertVerified,
				got.Status,
			)
		}
	})

	t.Run("DeactivateStaleNarratives_OnlyWithoutRecentClaims", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		now := time.Now()
		stale := env.newNarrative(t, topic.ID)
		fresh := env.newNarrative(t, topic.ID)
		withoutClaims := env.newNarrative(t, topic.ID)
		staleClaim := env.newClaim(t, topic.ID, now.Add(-10*24*time.Hour))
		freshClaim := env.newClaim(t, topic.ID, now.Add(-1*time.Hour))
		err := env.repo.LinkClaim(env.ctx, staleClaim.ID, stale.ID)
		if err != nil {
			t.Fatalf("unexpected error linking stale claim: %v", err)
		}
		err = env.repo.LinkClaim(env.ctx, freshClaim.ID, fresh.ID)
		if err != nil {
			t.Fatalf("unexpected error linking fresh claim: %v", err)
		}

		count, err := env.repo.DeactivateStaleNarratives(env.ctx, now.Add(-7*24*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count < 2 {
			t.Errorf("expected at least 2 narratives deactivated, got %d", count)
		}

		wantActive := map[uuid.UUID]bool{
			stale.ID:         false,
			fresh.ID:         true,
			withoutClaims.ID: false,
		}
		for id, want := range wantActive {
			got, err := env.repo.GetNarrative(env.ctx, id)
			if err != nil {
				t.Fatalf("unexpected error fetching narrative %s: %v", id, err)
			}
			if got.IsActive != want {
				t.Errorf("narrative %s: expected is_active %t, got %t", id, want, got.IsActive)
			}
		}
	})

	t.Run("PaginateNarratives_NewestFirst", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		// created_at defaults to the transaction start, so narratives made in
		// one test tx would tie. Far-future timestamps give a known order that
		// sorts before any other row in the table.
		base := time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)
		narratives := []*model.Narrative{
			env.newNarrative(t, topic.ID),
			env.newNarrative(t, topic.ID),
			env.newNarrative(t, topic.ID),
		}
		for i, n := range narratives {
			_, err := env.tx.Exec(
				env.ctx,
				"update narratives set created_at = $2 where id = $1",
				n.ID, base.Add(time.Duration(i)*time.Hour),
			)
			if err != nil {
				t.Fatalf("failed to set created_at: %v", err)
			}
		}
		oldest, middle, newest := narratives[0], narratives[1], narratives[2]

		first, err := env.repo.PaginateNarratives(
			env.ctx,
			&dto.PaginationDTO{Page: 1, PerPage: 2},
			data.NarrativeFilter{},
		)
		if err != nil {
			t.Fatalf("unexpected error on page 1: %v", err)
		}
		if len(first.Items) != 2 ||
			first.Items[0].ID != newest.ID ||
			first.Items[1].ID != middle.ID {
			t.Errorf("expected page 1 to be [%s %s], got %+v", newest.ID, middle.ID, first.Items)
		}
		if first.Page != 1 || first.PerPage != 2 {
			t.Errorf("expected page 1 of 2 per page, got page %d of %d", first.Page, first.PerPage)
		}
		if first.TotalItems < 3 {
			t.Errorf("expected at least 3 narratives in total, got %d", first.TotalItems)
		}

		second, err := env.repo.PaginateNarratives(
			env.ctx,
			&dto.PaginationDTO{Page: 2, PerPage: 2},
			data.NarrativeFilter{},
		)
		if err != nil {
			t.Fatalf("unexpected error on page 2: %v", err)
		}
		if len(second.Items) == 0 || second.Items[0].ID != oldest.ID {
			t.Errorf("expected page 2 to start with %s, got %+v", oldest.ID, second.Items)
		}
		if second.Page != 2 {
			t.Errorf("expected page 2, got %d", second.Page)
		}
	})

	t.Run("PaginateNarratives_FiltersByIsActiveAndStatus", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		active := env.newNarrative(t, topic.ID) // active, needs_check
		inactive := env.newNarrative(t, topic.ID)
		verified := env.newNarrative(t, topic.ID)
		_, err := env.tx.Exec(env.ctx,
			"update narratives set is_active = false where id = $1", inactive.ID)
		if err != nil {
			t.Fatalf("failed to deactivate narrative: %v", err)
		}
		_, err = env.tx.Exec(env.ctx,
			"update narratives set status = 'expert_verified' where id = $1", verified.ID)
		if err != nil {
			t.Fatalf("failed to set status: %v", err)
		}

		ids := func(res *dto.PaginationResults[model.Narrative]) map[uuid.UUID]bool {
			m := make(map[uuid.UUID]bool, len(res.Items))
			for i := range res.Items {
				m[res.Items[i].ID] = true
			}
			return m
		}
		all := &dto.PaginationDTO{Page: 1, PerPage: 100}

		off := false
		got, err := env.repo.PaginateNarratives(env.ctx, all, data.NarrativeFilter{IsActive: &off})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m := ids(got); !m[inactive.ID] || m[active.ID] || m[verified.ID] {
			t.Errorf(
				"is_active=false should return only the inactive narrative, got %+v",
				got.Items,
			)
		}

		got, err = env.repo.PaginateNarratives(
			env.ctx, all, data.NarrativeFilter{Status: string(model.NarrativeStatusExpertVerified)},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m := ids(got); !m[verified.ID] || m[active.ID] || m[inactive.ID] {
			t.Errorf("status filter should return only the verified narrative, got %+v", got.Items)
		}
		if got.TotalItems != len(got.Items) {
			t.Errorf(
				"total_items %d should count only matching rows (%d)",
				got.TotalItems,
				len(got.Items),
			)
		}
	})

	t.Run("Narrative_TopicNameAndNewestClaimVerdict", func(t *testing.T) {
		env := newNarrativeTestEnv(t, pool)
		topic := env.newTopic(t, true)
		narrative := env.newNarrative(t, topic.ID)

		// No claim yet: topic name is there, verdict is not.
		got, err := env.repo.GetNarrative(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.TopicName != topic.Name {
			t.Errorf("expected topic_name %q, got %q", topic.Name, got.TopicName)
		}
		if got.Verdict != nil {
			t.Errorf("expected no verdict without claims, got %q", *got.Verdict)
		}

		// Two claims: the verdict must come from the newest one.
		now := time.Now()
		older := newNarrativeTestClaim(topic.ID, "older "+uuid.NewString(), now.Add(-2*time.Hour))
		older.Verdict = model.VerdictSupported
		newer := newNarrativeTestClaim(topic.ID, "newer "+uuid.NewString(), now.Add(-1*time.Hour))
		newer.Verdict = model.VerdictContradicted
		for _, c := range []*model.Claim{older, newer} {
			created, err := env.repo.CreateClaim(env.ctx, c)
			if err != nil {
				t.Fatalf("failed to create claim: %v", err)
			}
			if err := env.repo.LinkClaim(env.ctx, created.ID, narrative.ID); err != nil {
				t.Fatalf("failed to link claim: %v", err)
			}
		}

		got, err = env.repo.GetNarrative(env.ctx, narrative.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Verdict == nil || *got.Verdict != model.VerdictContradicted.String() {
			t.Errorf(
				"expected verdict %q from the newest claim, got %v",
				"contradicted",
				got.Verdict,
			)
		}

		// The paginated list must carry the same fields.
		list, err := env.repo.PaginateNarratives(
			env.ctx, &dto.PaginationDTO{Page: 1, PerPage: 100}, data.NarrativeFilter{},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i := range list.Items {
			if list.Items[i].ID != narrative.ID {
				continue
			}
			if list.Items[i].TopicName != topic.Name || list.Items[i].Verdict == nil ||
				*list.Items[i].Verdict != "contradicted" {
				t.Errorf("list item missing topic_name/verdict: %+v", list.Items[i])
			}
			return
		}
		t.Error("narrative not found in the paginated list")
	})
}

// TestNarrativeRepo_WithTx_Enqueue commits real rows (a rolled-back test tx
// cannot show that the job and the claim land together), so it cleans up
// after itself.
func TestNarrativeRepo_WithTx_Enqueue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	defer pool.Close()

	ctx := t.Context()

	// River validates at Insert-time that the job kind is registered in the
	// client's Workers bundle, even though this test never starts the client
	// to process jobs - it only inserts and inspects river_job directly.
	workers := river.NewWorkers()
	river.AddWorker(workers, &noopClusterClaimWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("failed to construct river client: %v", err)
	}

	narrativeRepo := repo.NewNarrativeRepo(q, pool, func() *river.Client[pgx.Tx] {
		return riverClient
	})

	topic, err := q.UpsertTopicByName(ctx, db.UpsertTopicByNameParams{
		Name:        "test-topic-" + uuid.NewString(),
		Description: "A test topic",
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	// Registered first, so it runs last: claims reference the topic.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from topics where id = $1", topic.ID)
	})

	countJobs := func(t *testing.T, claimID uuid.UUID) int {
		t.Helper()

		var jobs int
		err := pool.QueryRow(
			ctx,
			"select count(*) from river_job where kind = $1 and args->>'claim_id' = $2",
			queue.JobTypeClusterClaim, claimID.String(),
		).Scan(&jobs)
		if err != nil {
			t.Fatalf("failed to count river jobs: %v", err)
		}
		return jobs
	}

	t.Run("Success_CommitsClaimAndEnqueuesJob", func(t *testing.T) {
		claim := newNarrativeTestClaim(topic.ID, "committed claim", time.Now())
		var claimID uuid.UUID

		err := narrativeRepo.WithTx(ctx, func(tx data.NarrativeRepoer) error {
			created, err := tx.CreateClaim(ctx, claim)
			if err != nil {
				return err
			}
			claimID = created.ID
			return tx.EnqueueClusterClaim(ctx, created.ID)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), "delete from claims where id = $1", claimID)
		})

		_, err = narrativeRepo.GetClaim(ctx, claimID)
		if err != nil {
			t.Errorf("expected the claim to be committed, got error: %v", err)
		}
		if jobs := countJobs(t, claimID); jobs != 1 {
			t.Errorf("expected 1 enqueued cluster job, got %d", jobs)
		}
	})

	t.Run("Failure_RollsBackClaimAndJob", func(t *testing.T) {
		claim := newNarrativeTestClaim(topic.ID, "rolled back claim", time.Now())
		wantErr := errors.New("boom")
		var claimID uuid.UUID

		err := narrativeRepo.WithTx(ctx, func(tx data.NarrativeRepoer) error {
			created, err := tx.CreateClaim(ctx, claim)
			if err != nil {
				return err
			}
			claimID = created.ID
			err = tx.EnqueueClusterClaim(ctx, created.ID)
			if err != nil {
				return err
			}
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v from WithTx, got %v", wantErr, err)
		}

		_, err = narrativeRepo.GetClaim(ctx, claimID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected the claim to be rolled back, got: %v", err)
		}
		if jobs := countJobs(t, claimID); jobs != 0 {
			t.Errorf("expected the job to be rolled back, got %d", jobs)
		}
	})
}
