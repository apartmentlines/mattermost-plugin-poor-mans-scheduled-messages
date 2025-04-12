package kvstore

import (
	"github.com/mattermost/mattermost/server/public/pluginapi"
	// "github.com/pkg/errors"
)

// We expose our calls to the KVStore pluginapi methods through this interface for testability and stability.
// This allows us to better control which values are stored with which keys.

type Client struct {
	client *pluginapi.Client
}

func NewKVStore(client *pluginapi.Client) KVStore {
	return Client{
		client: client,
	}
}

func (kv Client) LoadLastIDFromKV() (int64, error) {
	var lastID int64
	err := kv.client.KV.Get("schedule_last_id", &lastID)
	if err != nil {
		return 0, err
	}
	return lastID, nil
}
