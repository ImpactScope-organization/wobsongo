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

func (e *conversationTestEnv) volume(
	t *testing.T,
	filter data.MessageVolumeFilter,
) *model.MessageVolume {
	t.Helper()

	volume, err := e.repo.MessageVolume(e.ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return volume
}

// volumeDelta is what a test's messages added to a volume. The database may
// hold other messages, so each test judges only what it added.
func volumeDelta(t *testing.T, before, after *model.MessageVolume) model.MessageVolume {
	t.Helper()

	if len(before.Trend) != len(after.Trend) {
		t.Fatalf("trend length changed from %d to %d", len(before.Trend), len(after.Trend))
	}
	delta := model.MessageVolume{
		Current:  after.Current - before.Current,
		Previous: after.Previous - before.Previous,
		Trend:    make([]int64, len(after.Trend)),
	}
	for i := range after.Trend {
		delta.Trend[i] = after.Trend[i] - before.Trend[i]
	}
	return delta
}

// wantTrend builds a trend of n buckets from bucket index to count.
func wantTrend(n int, counts map[int]int64) []int64 {
	trend := make([]int64, n)
	for i, count := range counts {
		trend[i] = count
	}
	return trend
}

func assertVolume(t *testing.T, got model.MessageVolume, current, previous int64, trend []int64) {
	t.Helper()

	if got.Current != current || got.Previous != previous {
		t.Errorf("expected %d current and %d previous, got %d and %d",
			current, previous, got.Current, got.Previous)
	}
	if len(got.Trend) != len(trend) {
		t.Fatalf("expected %d buckets, got %d", len(trend), len(got.Trend))
	}
	for i := range trend {
		if got.Trend[i] != trend[i] {
			t.Errorf(
				"bucket %d: expected %d, got %d (trend %v)",
				i,
				trend[i],
				got.Trend[i],
				got.Trend,
			)
		}
	}
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

func TestConversationRepo_MessageVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	const (
		hour = time.Hour
		day  = 24 * time.Hour
	)
	hourly := data.MessageVolumeFilter{Window: day, Bucket: hour}
	daily := data.MessageVolumeFilter{Window: 7 * day, Bucket: day}

	t.Run("CountsPerHourOverTheLastDay", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)
		before := env.volume(t, hourly)

		a, b := newJID(), newJID()
		env.addMessage(t, a, model.ConversationRoleUser, 30*time.Minute) // last bucket
		env.addMessage(t, a, model.ConversationRoleUser, 30*time.Minute)
		env.addMessage(t, b, model.ConversationRoleUser, 90*time.Minute) // the bucket before
		env.addMessage(t, a, model.ConversationRoleUser, 25*hour)        // previous window
		env.addMessage(t, a, model.ConversationRoleUser, 47*hour)        // previous window
		env.addMessage(t, a, model.ConversationRoleUser, 49*hour)        // before both windows

		got := volumeDelta(t, before, env.volume(t, hourly))
		assertVolume(t, got, 3, 2, wantTrend(24, map[int]int64{23: 2, 22: 1}))
	})

	t.Run("CountsPerDayOverTheLastWeek", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)
		before := env.volume(t, daily)

		jid := newJID()
		env.addMessage(t, jid, model.ConversationRoleUser, 12*hour) // last bucket
		env.addMessage(t, jid, model.ConversationRoleUser, 36*hour) // the bucket before
		env.addMessage(t, jid, model.ConversationRoleUser, 8*day)   // previous window
		env.addMessage(t, jid, model.ConversationRoleUser, 15*day)  // before both windows

		got := volumeDelta(t, before, env.volume(t, daily))
		assertVolume(t, got, 2, 1, wantTrend(7, map[int]int64{6: 1, 5: 1}))
	})

	t.Run("WindowStartBelongsToCurrentWindow", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)
		before := env.volume(t, hourly)

		// Exactly one window old: the first bucket of the current window.
		env.addMessage(t, newJID(), model.ConversationRoleUser, day)

		got := volumeDelta(t, before, env.volume(t, hourly))
		assertVolume(t, got, 1, 0, wantTrend(24, map[int]int64{0: 1}))
	})

	t.Run("IgnoresMessagesNotFromUsers", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)
		before := env.volume(t, hourly)

		jid := newJID()
		env.addMessage(t, jid, model.ConversationRoleAssistant, 30*time.Minute)
		env.addMessage(t, jid, model.ConversationRoleSystem, 30*time.Minute)

		got := volumeDelta(t, before, env.volume(t, hourly))
		assertVolume(t, got, 0, 0, wantTrend(24, nil))
	})

	// The dashboard shows this count as "messages received" next to the
	// chatbot's "questions": over the same window they must be the same number.
	t.Run("CurrentMatchesUserActivityOfTheSameWindow", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)
		volumeBefore := env.volume(t, daily)
		activityBefore := env.activity(t, lastDays(7))

		jid := newJID()
		env.addMessage(t, jid, model.ConversationRoleUser, 12*hour)
		env.addMessage(t, jid, model.ConversationRoleUser, 3*day)
		env.addMessage(t, newJID(), model.ConversationRoleUser, 6*day)
		env.addMessage(t, jid, model.ConversationRoleUser, 8*day) // outside both

		volume := volumeDelta(t, volumeBefore, env.volume(t, daily))
		activity := env.activity(t, lastDays(7)).Messages - activityBefore.Messages

		if volume.Current != 3 || activity != 3 {
			t.Errorf("expected both to count 3 messages, volume %d, activity %d",
				volume.Current, activity)
		}
	})

	t.Run("RejectsInvalidFilter", func(t *testing.T) {
		env := newConversationTestEnv(t, pool)

		filters := map[string]data.MessageVolumeFilter{
			"ZeroBucket":              {Window: day},
			"NegativeBucket":          {Window: day, Bucket: -hour},
			"WindowShorterThanBucket": {Window: hour, Bucket: 2 * hour},
			"WindowNotMultiple":       {Window: day, Bucket: 7 * hour},
		}
		for name, filter := range filters {
			if _, err := env.repo.MessageVolume(env.ctx, filter); err == nil {
				t.Errorf("%s: expected an error, got nil", name)
			}
		}
	})
}
