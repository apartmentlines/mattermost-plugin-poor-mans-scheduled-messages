package store

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type fakeListMatching struct{ prefixCalled string }

func (f *fakeListMatching) WithPrefix(p string) pluginapi.ListKeysOption {
	f.prefixCalled = p
	return pluginapi.WithPrefix(p)
}

func sampleMessage(id, user string, t time.Time) *types.ScheduledMessage {
	return &types.ScheduledMessage{
		ID:             id,
		UserID:         user,
		ChannelID:      "chan",
		PostAt:         t,
		MessageContent: "hello",
		Timezone:       "UTC",
	}
}

func TestSaveScheduledMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	logger := testutil.FakeLogger{}

	store := NewKVStore(logger, kvMock, listFake, constants.MaxUserMessages)

	userID := "user"
	msgID := uuid.NewString()
	msg := sampleMessage(msgID, userID, time.Now())

	indexKey := testutil.IndexKey(userID)
	schedKey := testutil.SchedKey(msgID)

	gomock.InOrder(
		kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(nil),
		kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(true, nil),
		kvMock.EXPECT().Set(schedKey, msg).Return(true, nil),
	)

	err := store.SaveScheduledMessage(userID, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveScheduledMessage_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	logger := testutil.FakeLogger{}
	store := NewKVStore(logger, kvMock, listFake, constants.MaxUserMessages)

	userID := "user"
	msgID := uuid.NewString()
	msg := sampleMessage(msgID, userID, time.Now())

	indexKey := testutil.IndexKey(userID)
	schedKey := testutil.SchedKey(msgID)

	gomock.InOrder(
		kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(nil),
		kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(true, nil),
		kvMock.EXPECT().Set(schedKey, msg).Return(false, fmt.Errorf("save failed")),
	)

	err := store.SaveScheduledMessage(userID, msg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteScheduledMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	logger := testutil.FakeLogger{}
	store := NewKVStore(logger, kvMock, listFake, constants.MaxUserMessages)

	userID := "user"
	msgID := uuid.NewString()

	indexKey := testutil.IndexKey(userID)
	schedKey := testutil.SchedKey(msgID)

	gomock.InOrder(
		kvMock.EXPECT().Delete(schedKey).Return(nil),
		kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
			func(_ string, ids any) error {
				ptr := ids.(*[]string)
				*ptr = []string{msgID}
				return nil
			},
		),
		kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(true, nil),
	)

	err := store.DeleteScheduledMessage(userID, msgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetScheduledMessage_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	logger := testutil.FakeLogger{}
	store := NewKVStore(logger, kvMock, listFake, constants.MaxUserMessages)

	msgID := uuid.NewString()
	schedKey := testutil.SchedKey(msgID)

	kvMock.EXPECT().Get(schedKey, gomock.Any()).Return(fmt.Errorf("not found"))

	_, err := store.GetScheduledMessage(msgID)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestListAllScheduledIDs_SkipBadKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	logger := testutil.FakeLogger{}
	store := NewKVStore(logger, kvMock, listFake, constants.MaxUserMessages)

	msgID1 := uuid.NewString()
	msgID2 := uuid.NewString()

	key1 := testutil.SchedKey(msgID1)
	key2 := testutil.SchedKey(msgID2)

	prefixOpt := pluginapi.WithPrefix(constants.SchedPrefix)
	listFake.prefixCalled = ""

	kvMock.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.AssignableToTypeOf(prefixOpt)).Return([]string{key1, key2}, nil)

	kvMock.EXPECT().Get(key1, gomock.Any()).Return(fmt.Errorf("corrupt"))
	kvMock.EXPECT().Get(key2, gomock.Any()).DoAndReturn(
		func(_ string, v any) error {
			ptr := v.(*types.ScheduledMessage)
			*ptr = *sampleMessage(msgID2, "u", time.Unix(123, 0))
			return nil
		},
	)

	got, err := store.ListAllScheduledIDs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]int64{msgID2: 123}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: expected %v got %v", want, got)
	}

	if listFake.prefixCalled != constants.SchedPrefix {
		t.Fatalf("prefix mismatch: %s", listFake.prefixCalled)
	}
}

func TestSaveScheduledMessage_GetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	msg := sampleMessage(msgID, userID, time.Now())
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(fmt.Errorf("get failed"))
	err := store.SaveScheduledMessage(userID, msg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSaveScheduledMessage_IndexSetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	msg := sampleMessage(msgID, userID, time.Now())
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(nil)
	kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(false, fmt.Errorf("set index failed"))
	err := store.SaveScheduledMessage(userID, msg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteScheduledMessage_DeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	schedKey := testutil.SchedKey(msgID)
	kvMock.EXPECT().Delete(schedKey).Return(fmt.Errorf("delete failed"))
	err := store.DeleteScheduledMessage(userID, msgID)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteScheduledMessage_GetIndexError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	schedKey := testutil.SchedKey(msgID)
	indexKey := testutil.IndexKey(userID)
	gomock.InOrder(
		kvMock.EXPECT().Delete(schedKey).Return(nil),
		kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(fmt.Errorf("idx get fail")),
	)
	err := store.DeleteScheduledMessage(userID, msgID)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteScheduledMessage_SetIndexError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	schedKey := testutil.SchedKey(msgID)
	indexKey := testutil.IndexKey(userID)
	gomock.InOrder(
		kvMock.EXPECT().Delete(schedKey).Return(nil),
		kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
			func(_ string, ids any) error {
				ptr := ids.(*[]string)
				*ptr = []string{msgID}
				return nil
			},
		),
		kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(false, fmt.Errorf("idx set fail")),
	)
	err := store.DeleteScheduledMessage(userID, msgID)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCleanupMessageFromUserIndex_Happy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	indexKey := testutil.IndexKey(userID)
	gomock.InOrder(
		kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
			func(_ string, ids any) error {
				ptr := ids.(*[]string)
				*ptr = []string{msgID}
				return nil
			},
		),
		kvMock.EXPECT().Set(indexKey, gomock.Any()).Return(true, nil),
	)
	if err := store.CleanupMessageFromUserIndex(userID, msgID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupMessageFromUserIndex_MissingID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
		func(_ string, ids any) error {
			ptr := ids.(*[]string)
			*ptr = []string{"other"}
			return nil
		},
	)
	if err := store.CleanupMessageFromUserIndex(userID, msgID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupMessageFromUserIndex_GetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	msgID := uuid.NewString()
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(fmt.Errorf("idx get fail"))
	if err := store.CleanupMessageFromUserIndex(userID, msgID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetScheduledMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	msgID := uuid.NewString()
	schedKey := testutil.SchedKey(msgID)
	want := sampleMessage(msgID, "u", time.Unix(55, 0))
	kvMock.EXPECT().Get(schedKey, gomock.Any()).DoAndReturn(
		func(_ string, v any) error {
			ptr := v.(*types.ScheduledMessage)
			*ptr = *want
			return nil
		},
	)
	got, err := store.GetScheduledMessage(msgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch got %v want %v", got, want)
	}
}

func TestListAllScheduledIDs_Happy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	msgID := uuid.NewString()
	key := testutil.SchedKey(msgID)
	prefixOpt := pluginapi.WithPrefix(constants.SchedPrefix)
	kvMock.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.AssignableToTypeOf(prefixOpt)).Return([]string{key}, nil)
	kvMock.EXPECT().Get(key, gomock.Any()).DoAndReturn(
		func(_ string, v any) error {
			ptr := v.(*types.ScheduledMessage)
			*ptr = *sampleMessage(msgID, "u", time.Unix(777, 0))
			return nil
		},
	)
	got, err := store.ListAllScheduledIDs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]int64{msgID: 777}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch")
	}
}

func TestListAllScheduledIDs_ListKeysError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	prefixOpt := pluginapi.WithPrefix(constants.SchedPrefix)
	kvMock.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.AssignableToTypeOf(prefixOpt)).Return(nil, fmt.Errorf("list error"))
	_, err := store.ListAllScheduledIDs()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestListUserMessageIDs_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
		func(_ string, ids any) error {
			ptr := ids.(*[]string)
			*ptr = []string{"a", "b"}
			return nil
		},
	)
	got, err := store.ListUserMessageIDs(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch")
	}
}

func TestListUserMessageIDs_GetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)
	userID := "user"
	indexKey := testutil.IndexKey(userID)
	kvMock.EXPECT().Get(indexKey, gomock.Any()).Return(fmt.Errorf("get err"))
	_, err := store.ListUserMessageIDs(userID)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenerateMessageID_Unique(t *testing.T) {
	store := &kvStore{}
	id1 := store.GenerateMessageID()
	id2 := store.GenerateMessageID()
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("ids not unique")
	}
}
func TestSaveScheduledMessage_DuplicateIDNoIndexWrite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kvMock := mock.NewMockKVService(ctrl)
	listFake := &fakeListMatching{}
	store := NewKVStore(testutil.FakeLogger{}, kvMock, listFake, constants.MaxUserMessages)

	userID := "user"
	msgID := uuid.NewString()
	msg := sampleMessage(msgID, userID, time.Now())

	indexKey := testutil.IndexKey(userID)
	schedKey := testutil.SchedKey(msgID)

	kvMock.EXPECT().Get(indexKey, gomock.Any()).DoAndReturn(
		func(_ string, ids any) error {
			ptr := ids.(*[]string)
			*ptr = []string{msgID}
			return nil
		},
	)

	kvMock.EXPECT().Set(schedKey, msg).Return(true, nil)

	if err := store.SaveScheduledMessage(userID, msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
