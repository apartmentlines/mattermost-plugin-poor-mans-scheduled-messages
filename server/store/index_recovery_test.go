package store

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/require"
)

type pausingCreationKV struct {
	*testutil.MemoryKV
	paused  atomic.Bool
	writing chan struct{}
	resume  chan struct{}
}

func (db *pausingCreationKV) KVSetWithOptions(key string, value []byte, opts model.PluginKVSetOptions) (bool, *model.AppError) {
	if key == schedKey("A") && db.paused.CompareAndSwap(false, true) {
		close(db.writing)
		<-db.resume
	}
	return db.MemoryKV.KVSetWithOptions(key, value, opts)
}

func TestConcurrentListCleanupCannotHideAcceptedJob(t *testing.T) {
	db := &pausingCreationKV{MemoryKV: testutil.NewMemoryKV(), writing: make(chan struct{}), resume: make(chan struct{})}
	st := NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 1)
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(db.resume) }) })
	done := make(chan error, 1)
	go func() { done <- st.SaveScheduledMessage("user", testMessage("A")) }()
	<-db.writing
	// Perform exactly the missing-record cleanup used by /schedule list.
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	for _, id := range ids {
		_, err := st.GetScheduledMessage(id)
		if errors.Is(err, types.ErrMessageNotFound) {
			require.NoError(t, st.CleanupMessageFromUserIndex("user", id))
		} else {
			require.NoError(t, err)
		}
	}
	require.NoError(t, st.SaveScheduledMessage("user", testMessage("B")))
	release.Do(func() { close(db.resume) })
	aErr := <-done
	_, err = st.ListScheduledMessages()
	require.NoError(t, err)
	ids, err = st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Contains(t, ids, "B")
	if aErr == nil {
		require.Contains(t, ids, "A", "a successfully accepted job must remain visible and cancellable")
	}
	require.Len(t, ids, 1, "concurrent cleanup must not release a live creation reservation")
}

func TestRepairAcceptedJobDoesNotApplyCreationQuota(t *testing.T) {
	db := testutil.NewMemoryKV()
	cli := pluginapi.NewClient(db, nil)
	st := NewKVStore(testutil.FakeLogger{}, &cli.KV, nil, 1)
	// A is an already accepted job whose index was lost in an earlier crash.
	_, err := cli.KV.Set(schedKey("A"), testMessage("A"))
	require.NoError(t, err)
	require.NoError(t, st.SaveScheduledMessage("user", testMessage("B")))
	_, err = st.ListScheduledMessages()
	require.NoError(t, err)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"A", "B"}, ids)
	require.NoError(t, st.DeleteScheduledMessage("user", "A"))
}

func TestStaleCleanupPreservesLiveCanonicalJob(t *testing.T) {
	st, _ := testStore()
	saveJob(t, st, "A")
	// A caller can act on an earlier missing-record read after creation finishes.
	require.NoError(t, st.CleanupMessageFromUserIndex("user", "A"))
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, ids)
}

func TestLiveCreationReservationSurvivesListCleanup(t *testing.T) {
	db := testutil.NewMemoryKV()
	st := NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 1)
	indexed := make(chan struct{})
	resume := make(chan struct{})
	var reads atomic.Int32
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(resume) }) })
	db.AfterGet = func(key string) {
		if key == schedKey("A") && reads.Add(1) == 2 {
			// Reservation is committed; activation has not run yet.
			close(indexed)
			<-resume
		}
	}
	done := make(chan error, 1)
	go func() { done <- st.SaveScheduledMessage("user", testMessage("A")) }()
	<-indexed
	creating, err := st.GetScheduledMessage("A")
	require.NoError(t, err)
	require.Equal(t, types.StateCreating, creating.State)
	require.False(t, creating.Due(testNow))
	_, err = st.ClaimScheduledMessage("user", "A", testNow, true)
	require.ErrorIs(t, err, types.ErrMessageBusy)
	require.NoError(t, st.CleanupMessageFromUserIndex("user", "A"))
	require.ErrorIs(t, st.SaveScheduledMessage("user", testMessage("B")), types.ErrMessageLimit)
	release.Do(func() { close(resume) })
	require.NoError(t, <-done)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, ids)
	require.NoError(t, st.DeleteScheduledMessage("user", "A"))
}

func TestInterruptedCreationRecoversItsIndexBeforeActivation(t *testing.T) {
	db := testutil.NewMemoryKV()
	st := NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 1)
	db.FailWrites(indexKey("user"), 1)
	require.Error(t, st.SaveScheduledMessage("user", testMessage("A")))
	creating, err := st.GetScheduledMessage("A")
	require.NoError(t, err)
	require.Equal(t, types.StateCreating, creating.State)
	require.False(t, creating.Due(testNow))
	jobs, err := st.ListScheduledMessages()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, types.StatePending, jobs[0].State)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, ids)
}

func TestRejectedCreationNeverBecomesPublishable(t *testing.T) {
	db := testutil.NewMemoryKV()
	st := NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 1)
	saveJob(t, st, "A")
	require.ErrorIs(t, st.SaveScheduledMessage("user", testMessage("B")), types.ErrMessageLimit)
	rejected, err := st.GetScheduledMessage("B")
	require.NoError(t, err)
	require.Equal(t, types.StateCancelled, rejected.State)
	require.Empty(t, rejected.MessageContent)
	jobs, err := st.ListScheduledMessages()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, "A", jobs[0].ID)
}

func TestFailedCanonicalCreationDoesNotReserveCapacity(t *testing.T) {
	db := testutil.NewMemoryKV()
	st := NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 1)
	db.FailWrites(schedKey("A"), 1)
	require.Error(t, st.SaveScheduledMessage("user", testMessage("A")))
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Empty(t, ids)
	require.NoError(t, st.SaveScheduledMessage("user", testMessage("B")))
}
