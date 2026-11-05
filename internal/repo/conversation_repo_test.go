package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type conversationTestEnv struct {
	ctx  context.Context
	tx   pgx.Tx
	repo data.ConversationRepoer
}

func newConversationTestEnv(t *testing.T, pool *pgxpool.Pool) *conversationTestEnv {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })

	return &conversationTestEnv{
		ctx:  ctx,
		tx:   tx,
		repo: repo.NewConversationRepo(db.New(tx), pool, nil),
	}
}

// newJID returns a jid no other message in the database uses, so the test can
// count it as one new user.
func newJID() string {
	return "test-" + uuid.NewString()
}

// addMessage stores a message for jid and moves it to age before now.
func (e *conversationTestEnv) addMessage(
	t *testing.T,
	jid string,
	role model.ConversationRole,
	age time.Duration,
) {
	t.Helper()

	id, err := e.repo.AppendMessageWithID(e.ctx, jid, role, "hello")
	if err != nil {
		t.Fatalf("failed to append message: %v", err)
	}
	_, err = e.tx.Exec(
		e.ctx,
		`UPDATE conversation_messages SET created_at = NOW() - make_interval(secs => $1) WHERE id = $2`,
		age.Seconds(),
		id,
	)
	if err != nil {
		t.Fatalf("failed to backdate message: %v", err)
	}
}

func (e *conversationTestEnv) activity(
	t *testing.T,
	filter data.UserActivityFilter,
) *model.UserActivity {
	t.Helper()

	activity, err := e.repo.UserActivity(e.ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return activity
}

func lastDays(n int32) data.UserActivityFilter {
	return data.UserActivityFilter{Days: &n}
}

// activityWindows are the filters the dashboard offers, each with what the
// messages of CountsEachUserOncePerWindow add to it.
var activityWindows = []struct {
	name   string
	filter data.UserActivityFilter
	want   model.UserActivity
}{
	{"AllTime", data.UserActivityFilter{}, model.UserActivity{Users: 3, Messages: 5}},
	{"Last7Days", lastDays(7), model.UserActivity{Users: 1, Messages: 2}},
	{"Last15Days", lastDays(15), model.UserActivity{Users: 1, Messages: 3}},
	{"Last30Days", lastDays(30), model.UserActivity{Users: 2, Messages: 4}},
}

func TestConversationRepo_UserActivity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	const day = 24 * time.Hour

	t.Run("CountsEachUserOncePerWindow", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)

		// The database may hold other messages, so each window is judged by
		// what the test added to it.
		before := make([]*model.UserActivity, len(activityWindows))
		for i, w := range activityWindows {
			before[i] = env.activity(t, w.filter)
		}

		// a writes often, across windows; b and c wrote only long ago.
		a, b, c := newJID(), newJID(), newJID()
		env.addMessage(t, a, model.ConversationRoleUser, 1*day)
		env.addMessage(t, a, model.ConversationRoleUser, 2*day)
		env.addMessage(t, a, model.ConversationRoleUser, 10*day)
		env.addMessage(t, b, model.ConversationRoleUser, 20*day)
		env.addMessage(t, c, model.ConversationRoleUser, 40*day)

		for i, w := range activityWindows {
			after := env.activity(t, w.filter)
			got := model.UserActivity{
				Users:    after.Users - before[i].Users,
				Messages: after.Messages - before[i].Messages,
			}
			if got != w.want {
				t.Errorf("%s: expected %+v more, got %+v", w.name, w.want, got)
			}
		}
	})

	t.Run("IgnoresMessagesNotFromUsers", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)

		before := make([]*model.UserActivity, len(activityWindows))
		for i, w := range activityWindows {
			before[i] = env.activity(t, w.filter)
		}

		jid := newJID()
		env.addMessage(t, jid, model.ConversationRoleAssistant, 1*day)
		env.addMessage(t, jid, model.ConversationRoleSystem, 1*day)

		for i, w := range activityWindows {
			if after := env.activity(t, w.filter); *after != *before[i] {
				t.Errorf("%s: expected no change, got %+v then %+v", w.name, before[i], after)
			}
		}
	})

	t.Run("NewMessageCountsInEveryWindow", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)

		before := make([]*model.UserActivity, len(activityWindows))
		for i, w := range activityWindows {
			before[i] = env.activity(t, w.filter)
		}

		env.addMessage(t, newJID(), model.ConversationRoleUser, 0)

		want := model.UserActivity{Users: 1, Messages: 1}
		for i, w := range activityWindows {
			after := env.activity(t, w.filter)
			got := model.UserActivity{
				Users:    after.Users - before[i].Users,
				Messages: after.Messages - before[i].Messages,
			}
			if got != want {
				t.Errorf("%s: expected %+v more, got %+v", w.name, want, got)
			}
		}
	})
}
