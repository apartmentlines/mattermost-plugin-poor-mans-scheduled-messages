package testutil

import (
	"bytes"
	"sort"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
)

// MemoryKV implements the server's byte-level compare-and-set contract for tests.
// Wrapping it in pluginapi.NewClient exercises real JSON encoding and KV options.
type MemoryKV struct {
	plugintest.API
	mu         sync.Mutex
	data       map[string][]byte
	failWrites map[string]int
	// AfterGet, when set before starting workers, can coordinate concurrent reads.
	AfterGet func(string)
}

// NewMemoryKV creates a shared persistent test database.
func NewMemoryKV() *MemoryKV {
	return &MemoryKV{data: make(map[string][]byte), failWrites: make(map[string]int)}
}

// KVGet returns an isolated snapshot of a stored value.
func (m *MemoryKV) KVGet(key string) ([]byte, *model.AppError) {
	m.mu.Lock()
	value := bytes.Clone(m.data[key])
	m.mu.Unlock()
	if m.AfterGet != nil {
		m.AfterGet(key)
	}
	return value, nil
}

// KVSetWithOptions compares the exact serialized old value atomically.
func (m *MemoryKV) KVSetWithOptions(key string, value []byte, opts model.PluginKVSetOptions) (bool, *model.AppError) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrites[key] > 0 {
		m.failWrites[key]--
		return false, &model.AppError{Id: "test.storage_failure", StatusCode: 500}
	}
	if opts.Atomic && !bytes.Equal(m.data[key], opts.OldValue) {
		return false, nil
	}
	if value == nil {
		delete(m.data, key)
	} else {
		m.data[key] = bytes.Clone(value)
	}
	return true, nil
}

// KVList provides server-style pagination over all plugin keys.
func (m *MemoryKV) KVList(page, perPage int) ([]string, *model.AppError) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start := page * perPage
	if start >= len(keys) {
		return nil, nil
	}
	end := min(start+perPage, len(keys))
	return keys[start:end], nil
}

// FailWrites injects storage errors into subsequent writes to a key.
func (m *MemoryKV) FailWrites(key string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failWrites[key] = count
}
