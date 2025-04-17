package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/channel"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

type stubLinker struct{}

func (stubLinker) GetInfoOrUnknown(string) *channel.Info { return &channel.Info{} }
func (stubLinker) MakeChannelLink(*channel.Info) string  { return "stub link" }

type fakePoster struct {
	createCalled int32
	dmCalled     int32
	createErr    error
	dmErr        error
}

func (p *fakePoster) CreatePost(*model.Post) error {
	p.createCalled++
	return p.createErr
}

func (p *fakePoster) DM(_, _ string, _ *model.Post) error {
	p.dmCalled++
	return p.dmErr
}

type fakeLogger struct{}

func (fakeLogger) Error(string, ...any) {}
func (fakeLogger) Warn(string, ...any)  {}
func (fakeLogger) Info(string, ...any)  {}
func (fakeLogger) Debug(string, ...any) {}

type fakeStore struct {
	ids          map[string]int64
	listErr      error
	messages     map[string]*types.ScheduledMessage
	getErr       error
	deleteErr    error
	deleteCalled int32
	getCalled    int32
}

func (fs *fakeStore) SaveScheduledMessage(string, *types.ScheduledMessage) error { return nil }
func (fs *fakeStore) DeleteScheduledMessage(string, string) error {
	fs.deleteCalled++
	return fs.deleteErr
}
func (fs *fakeStore) CleanupMessageFromUserIndex(string, string) error { return nil }
func (fs *fakeStore) GetScheduledMessage(id string) (*types.ScheduledMessage, error) {
	fs.getCalled++
	if fs.getErr != nil {
		return nil, fs.getErr
	}
	msg, ok := fs.messages[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return msg, nil
}
func (fs *fakeStore) ListAllScheduledIDs() (map[string]int64, error) { return fs.ids, fs.listErr }
func (fs *fakeStore) ListUserMessageIDs(string) ([]string, error)    { return nil, nil }
func (fs *fakeStore) GenerateMessageID() string                      { return "gen" }

func newScheduler(st store.Store, poster *fakePoster, clk clock.Clock) *Scheduler {
	linker := stubLinker{}
	return New(poster, fakeLogger{}, st, linker, "bot", clk)
}

func TestProcessDueMessages_PostSuccess(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID:             "id",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
	}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if poster.createCalled != 1 {
		t.Fatalf("expected createCalled 1 got %d", poster.createCalled)
	}
	if st.deleteCalled != 1 {
		t.Fatalf("expected deleteCalled 1 got %d", st.deleteCalled)
	}
}

func TestProcessDueMessages_PostFailure(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID:             "id",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
	}
	poster := &fakePoster{createErr: errors.New("fail")}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if poster.dmCalled != 1 {
		t.Fatalf("expected dmCalled 1 got %d", poster.dmCalled)
	}
}

func TestProcessDueMessages_NotDueYet(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID:             "id",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
	}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if poster.createCalled != 0 {
		t.Fatalf("expected createCalled 0 got %d", poster.createCalled)
	}
}

func TestProcessDueMessages_ListError(t *testing.T) {
	st := &fakeStore{listErr: errors.New("boom")}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{time.Now().UTC()})

	s.processDueMessages()

	if poster.createCalled != 0 {
		t.Fatalf("should not create post on list error")
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID:             "id",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
	}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{now})
	s.newTicker = func(time.Duration) *time.Ticker { return time.NewTicker(1 * time.Millisecond) }

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.Start()
		wg.Done()
	}()
	time.Sleep(5 * time.Millisecond)
	s.Stop()
	wg.Wait()

	if poster.createCalled == 0 {
		t.Fatalf("expected at least one create post call")
	}
}

func TestProcessDueMessages_LoadMessageError(t *testing.T) {
	now := time.Now().UTC()
	st := &fakeStore{
		ids:     map[string]int64{"id": now.Add(-time.Minute).Unix()},
		getErr:  errors.New("cannot load"),
	}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if poster.createCalled != 0 || poster.dmCalled != 0 || st.deleteCalled != 0 {
		t.Fatalf("unexpected calls: create=%d dm=%d delete=%d",
			poster.createCalled, poster.dmCalled, st.deleteCalled)
	}
}

func TestProcessDueMessages_DeleteScheduleError(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID:        "id", UserID: "u", ChannelID: "c",
		PostAt: now.Add(-time.Minute), MessageContent: "x", Timezone: "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
		deleteErr: errors.New("kv fail"),
	}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if st.deleteCalled != 1 {
		t.Fatalf("delete not attempted")
	}
	if poster.createCalled != 0 || poster.dmCalled != 0 {
		t.Fatalf("post or dm should not be called on delete error")
	}
}

func TestProcessDueMessages_DMError(t *testing.T) {
	now := time.Now().UTC()
	msg := &types.ScheduledMessage{
		ID: "id", UserID: "u", ChannelID: "c",
		PostAt: now.Add(-time.Minute), MessageContent: "x", Timezone: "UTC",
	}
	st := &fakeStore{
		ids:      map[string]int64{"id": msg.PostAt.Unix()},
		messages: map[string]*types.ScheduledMessage{"id": msg},
	}
	poster := &fakePoster{
		createErr: errors.New("post fail"),
		dmErr:     errors.New("dm fail"),
	}
	s := newScheduler(st, poster, fakeClock{now})

	s.processDueMessages()

	if poster.createCalled != 1 || poster.dmCalled != 1 {
		t.Fatalf("expected create=1 dm=1 got create=%d dm=%d",
			poster.createCalled, poster.dmCalled)
	}
}

func TestProcessDueMessages_EmptyIDMap(t *testing.T) {
	st := &fakeStore{ids: map[string]int64{}}
	poster := &fakePoster{}
	s := newScheduler(st, poster, fakeClock{time.Now().UTC()})

	s.processDueMessages()

	if poster.createCalled != 0 || poster.dmCalled != 0 || st.deleteCalled != 0 {
		t.Fatalf("no operations expected, got create=%d dm=%d delete=%d",
			poster.createCalled, poster.dmCalled, st.deleteCalled)
	}
}
