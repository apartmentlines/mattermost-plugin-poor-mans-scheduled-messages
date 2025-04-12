package kvstore

type KVStore interface {
	LoadLastIDFromKV() (int64, error)
}
