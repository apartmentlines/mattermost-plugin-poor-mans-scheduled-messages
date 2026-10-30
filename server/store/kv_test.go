package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/require"
)

var testNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func testStore() (ports.Store, *testutil.MemoryKV) {
	db := testutil.NewMemoryKV()
	return storeOn(db), db
}
func storeOn(db *testutil.MemoryKV) ports.Store {
	return NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 100)
}
func testMessage(id string) *types.ScheduledMessage {
	return &types.ScheduledMessage{ID: id, UserID: "user", ChannelID: "channel", RootID: "thread", PostAt: testNow, MessageContent: "original", Timezone: "UTC"}
}
func saveJob(t *testing.T, st ports.Store, id string) *types.ScheduledMessage {
	t.Helper()
	msg := testMessage(id)
	require.NoError(t, st.SaveScheduledMessage(msg.UserID, msg))
	return msg
}

func TestSaveLoadAndDuplicateProtection(t *testing.T) {
	st, _ := testStore()
	msg := saveJob(t, st, "job")
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StatePending, got.State)
	require.Equal(t, msg.MessageContent, got.MessageContent)
	require.Equal(t, msg.RootID, got.RootID)
	msg.MessageContent = "overwrite"
	require.ErrorIs(t, st.SaveScheduledMessage(msg.UserID, msg), types.ErrConflict)
	got, err = st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, "original", got.MessageContent)
	_, err = st.GetScheduledMessage("absent")
	require.ErrorIs(t, err, types.ErrMessageNotFound)
	require.NotEqual(t, st.GenerateMessageID(), st.GenerateMessageID())
}

func TestConcurrentClaimsAcrossStoreInstances(t *testing.T) {
	st, db := testStore()
	saveJob(t, st, "job")
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := storeOn(db).ClaimScheduledMessage("user", "job", testNow, true)
			if err == nil {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, winners.Load())
}

func TestClaimHonorsScheduleAndBackoff(t *testing.T) {
	st, _ := testStore()
	saveJob(t, st, "job")
	_, err := st.ClaimScheduledMessage("user", "job", testNow.Add(-time.Second), false)
	require.ErrorIs(t, err, types.ErrNotDue)
	_, err = st.ClaimScheduledMessage("other", "job", testNow, true)
	require.ErrorIs(t, err, types.ErrNotOwner)
	claim, err := st.ClaimScheduledMessage("user", "job", testNow, true)
	require.NoError(t, err)
	require.NoError(t, st.StartDelivery(claim, testNow))
	require.NoError(t, st.FinishDelivery(claim, types.DeliveryResult{State: types.StateRetry, Reason: "rate_limited"}, testNow))
	_, err = st.ClaimScheduledMessage("user", "job", testNow, false)
	require.ErrorIs(t, err, types.ErrNotDue)
	_, err = st.ClaimScheduledMessage("user", "job", testNow.Add(time.Minute), false)
	require.NoError(t, err)
}

func TestLeaseRecoveryFencesPreviousWorker(t *testing.T) {
	for _, started := range []bool{false, true} {
		t.Run(fmt.Sprint(started), func(t *testing.T) {
			st, db := testStore()
			saveJob(t, st, "job")
			first, err := st.ClaimScheduledMessage("user", "job", testNow, true)
			require.NoError(t, err)
			if started {
				require.NoError(t, st.StartDelivery(first, testNow))
			}
			second, err := storeOn(db).ClaimScheduledMessage("user", "job", testNow.Add(types.DeliveryLease), false)
			require.NoError(t, err)
			require.NotEqual(t, first.ClaimToken, second.ClaimToken)
			want := types.StateProcessing
			if started {
				want = types.StateReconciling
			}
			require.Equal(t, want, second.State)
			require.ErrorIs(t, st.StartDelivery(first, testNow), types.ErrLeaseLost)
			require.ErrorIs(t, st.FinishDelivery(first, types.DeliveryResult{State: types.StateSent, PostID: "post"}, testNow), types.ErrLeaseLost)
		})
	}
}

func TestCancelVersusStartDelivery(t *testing.T) {
	for iteration := 0; iteration < 25; iteration++ {
		st, db := testStore()
		saveJob(t, st, "job")
		claim, err := st.ClaimScheduledMessage("user", "job", testNow, true)
		require.NoError(t, err)
		gate := make(chan struct{})
		var reads atomic.Int32
		db.AfterGet = func(key string) {
			if key == schedKey("job") {
				n := reads.Add(1)
				if n == 1 {
					<-gate
				}
				if n == 2 {
					close(gate)
				}
			}
		}
		cancelled := make(chan error, 1)
		started := make(chan error, 1)
		go func() { cancelled <- st.DeleteScheduledMessage("user", "job") }()
		go func() { started <- storeOn(db).StartDelivery(claim, testNow) }()
		cancelErr, startErr := <-cancelled, <-started
		if cancelErr == nil {
			require.ErrorIs(t, startErr, types.ErrLeaseLost)
		} else {
			require.NoError(t, startErr)
			require.ErrorIs(t, cancelErr, types.ErrMessageBusy)
		}
	}
}

func TestConcurrentIndexAddAndRemove(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			st, db := testStore()
			if remove {
				saveJob(t, st, "old")
			}
			gate := make(chan struct{})
			var reads atomic.Int32
			db.AfterGet = func(key string) {
				if key == indexKey("user") {
					n := reads.Add(1)
					if n == 1 {
						<-gate
					}
					if n == 2 {
						close(gate)
					}
				}
			}
			errs := make(chan error, 2)
			go func() { errs <- st.SaveScheduledMessage("user", testMessage("new")) }()
			go func() {
				if remove {
					errs <- storeOn(db).DeleteScheduledMessage("user", "old")
				} else {
					errs <- storeOn(db).SaveScheduledMessage("user", testMessage("second"))
				}
			}()
			require.NoError(t, <-errs)
			require.NoError(t, <-errs)
			ids, err := st.ListUserMessageIDs("user")
			require.NoError(t, err)
			if remove {
				require.Equal(t, []string{"new"}, ids)
			} else {
				require.ElementsMatch(t, []string{"new", "second"}, ids)
			}
		})
	}
}

func TestReceiptSurvivesIndexCleanupFailure(t *testing.T) {
	st, db := testStore()
	saveJob(t, st, "job")
	claim, err := st.ClaimScheduledMessage("user", "job", testNow, true)
	require.NoError(t, err)
	require.NoError(t, st.StartDelivery(claim, testNow))
	db.FailWrites(indexKey("user"), 1)
	require.NoError(t, st.FinishDelivery(claim, types.DeliveryResult{State: types.StateSent, PostID: "post"}, testNow))
	_, err = st.ClaimScheduledMessage("user", "job", testNow, true)
	require.ErrorIs(t, err, types.ErrMessageFinal)
	msg, err := st.GetScheduledMessage("job")
	require.NoError(t, err)
	require.Empty(t, msg.MessageContent)
	require.Equal(t, "post", msg.PostID)
	jobs, err := st.ListScheduledMessages()
	require.NoError(t, err)
	require.Empty(t, jobs)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestJobIndexRepairAndLegacyJSON(t *testing.T) {
	st, db := testStore()
	cli := pluginapi.NewClient(db, nil)
	// Extra fields, whitespace, and absent state must not break byte-level CAS.
	raw := []byte(`{ "id":"legacy", "user_id":"user", "channel_id":"channel", "message_content":"original", "post_at":"2020-01-01T00:00:00Z", "future_field": true }`)
	_, err := cli.KV.Set(schedKey("legacy"), raw)
	require.NoError(t, err)
	jobs, err := st.ListScheduledMessages()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Equal(t, []string{"legacy"}, ids)
	claim, err := st.ClaimScheduledMessage("user", "legacy", testNow, false)
	require.NoError(t, err)
	require.Equal(t, types.StateProcessing, claim.State)
}

func TestRetryLimitAndExplicitRetry(t *testing.T) {
	st, _ := testStore()
	saveJob(t, st, "job")
	now := testNow
	for i := 1; i <= types.MaxDeliveryAttempts; i++ {
		claim, err := st.ClaimScheduledMessage("user", "job", now, false)
		require.NoError(t, err)
		require.NoError(t, st.StartDelivery(claim, now))
		require.NoError(t, st.FinishDelivery(claim, types.DeliveryResult{State: types.StateRetry}, now))
		require.Equal(t, i, claim.Attempts)
		if i < types.MaxDeliveryAttempts {
			require.Equal(t, types.StateRetry, claim.State)
			now = claim.NextAttemptAt
		} else {
			require.Equal(t, types.StateFailed, claim.State)
		}
	}
	_, err := st.ClaimScheduledMessage("user", "job", now.Add(time.Hour), false)
	require.ErrorIs(t, err, types.ErrNotDue)
	claim, err := st.ClaimScheduledMessage("user", "job", now, true)
	require.NoError(t, err)
	require.Zero(t, claim.Attempts)
}

func TestUncertainJobCannotBeCancelledOrRepublished(t *testing.T) {
	st, _ := testStore()
	saveJob(t, st, "job")
	claim, err := st.ClaimScheduledMessage("user", "job", testNow, true)
	require.NoError(t, err)
	require.NoError(t, st.StartDelivery(claim, testNow))
	require.NoError(t, st.FinishDelivery(claim, types.DeliveryResult{State: types.StateUncertain}, testNow))
	require.ErrorIs(t, st.DeleteScheduledMessage("user", "job"), types.ErrMessageBusy)
	claim, err = st.ClaimScheduledMessage("user", "job", testNow, true)
	require.NoError(t, err)
	require.Equal(t, types.StateReconciling, claim.State)
	require.ErrorIs(t, st.StartDelivery(claim, testNow), types.ErrLeaseLost)
	require.Error(t, st.FinishDelivery(claim, types.DeliveryResult{State: types.StateRetry}, testNow))
}

func TestCancellationAndOwnership(t *testing.T) {
	st, _ := testStore()
	saveJob(t, st, "job")
	require.ErrorIs(t, st.DeleteScheduledMessage("other", "job"), types.ErrNotOwner)
	require.NoError(t, st.DeleteScheduledMessage("user", "job"))
	require.NoError(t, st.DeleteScheduledMessage("user", "job"))
	_, err := st.ClaimScheduledMessage("user", "job", testNow, true)
	require.ErrorIs(t, err, types.ErrMessageFinal)
	ids, err := st.ListUserMessageIDs("user")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestScanPaginatesRawKeys(t *testing.T) {
	st, db := testStore()
	cli := pluginapi.NewClient(db, nil)
	for i := 0; i < 10000; i++ {
		_, err := cli.KV.Set(fmt.Sprintf("aaa:%05d", i), i)
		require.NoError(t, err)
	}
	saveJob(t, st, "job")
	jobs, err := st.ListScheduledMessages()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
}
