package storage

type StorageEngine interface {
	Save(data []byte) error
	Load(id string) ([]byte, error)
}
