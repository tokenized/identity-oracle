package tests

import (
	"context"

	"github.com/tokenized/pkg/storage"
)

// ============================================================
// Storage

type mockStorage struct {
	data map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: map[string][]byte{},
	}
}

func (m mockStorage) Write(ctx context.Context, key string, body []byte, options *storage.Options) error {
	m.data[key] = body
	return nil
}

func (m mockStorage) Read(ctx context.Context, key string) ([]byte, error) {
	body, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return body, nil
}

func (m mockStorage) Remove(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// Search is not implemented TODO
func (m mockStorage) Search(ctx context.Context, query map[string]string) ([][]byte, error) {
	objects := [][]byte{}
	return objects, nil
}

// List is not implemented TODO
func (m mockStorage) List(ctx context.Context, path string) ([]string, error) {
	objects := []string{}
	return objects, nil
}

func (m mockStorage) Clear(ctx context.Context, query map[string]string) error {
	for key := range m.data {
		delete(m.data, key)
	}
	return nil
}
