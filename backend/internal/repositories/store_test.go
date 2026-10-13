package repositories

import (
	"context"
	"path/filepath"
	"planningpoker/internal/domain"
	"testing"
)

func TestFileStorePersistsAndIsolatesSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Update(context.Background(), func(st *domain.State) error { st.Users["one"] = domain.User{ID: "one", Name: "Persistent"}; return nil }); err != nil {
		t.Fatal(err)
	}
	other, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = other.View(context.Background(), func(st *domain.State) error {
		if st.Users["one"].Name != "Persistent" {
			t.Fatal("persistence failed")
		}
		delete(st.Users, "one")
		return nil
	})
	_ = other.View(context.Background(), func(st *domain.State) error {
		if len(st.Users) != 1 {
			t.Fatal("View mutated stored state")
		}
		return nil
	})
}
