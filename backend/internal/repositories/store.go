package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"planningpoker/internal/domain"
	"sync"
)

type FileStore struct {
	mu    sync.Mutex
	path  string
	state *domain.State
}

func NewFile(path string) (*FileStore, error) {
	s := &FileStore{path: path, state: domain.NewState()}
	b, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(b, s.state)
	} else if os.IsNotExist(err) {
		err = nil
	}
	return s, err
}
func clone(s *domain.State) *domain.State {
	b, _ := json.Marshal(s)
	n := domain.NewState()
	_ = json.Unmarshal(b, n)
	return n
}
func (s *FileStore) View(_ context.Context, fn func(*domain.State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(clone(s.state))
}
func (s *FileStore) Update(_ context.Context, fn func(*domain.State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.state)
	result := fn(next)
	if s.path != "" {
		b, err := json.Marshal(next)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(s.path+".tmp", b, 0600); err != nil {
			return err
		}
		if err = os.Rename(s.path+".tmp", s.path); err != nil {
			return err
		}
	}
	s.state = next
	return result
}
func (s *FileStore) Close() {}
func (s *FileStore) Read(ctx context.Context, fn func(Transaction) error) error {
	return s.View(ctx, func(st *domain.State) error { return fn(&memoryTransaction{st}) })
}
func (s *FileStore) Write(ctx context.Context, _ string, fn func(Transaction) error) error {
	return s.Update(ctx, func(st *domain.State) error {
		original := clone(st)
		err := fn(&memoryTransaction{st})
		if err != nil {
			var domainError *domain.Error
			if !errors.As(err, &domainError) {
				*st = *original
			}
		}
		return err
	})
}
