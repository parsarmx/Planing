package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"planningpoker/internal/domain"
	"planningpoker/pkg"
	"time"
)

type ImportResult struct {
	Users, Rooms, Sessions, OTPs int
	AlreadyImported              bool
}

// Import is atomic and refuses to overwrite a database with existing application
// data. The source file is never edited; a source hash makes retries idempotent.
func (s *PostgresStore) Import(ctx context.Context, b []byte) (ImportResult, error) {
	result := ImportResult{}
	st := domain.NewState()
	if err := json.Unmarshal(b, st); err != nil {
		return result, fmt.Errorf("invalid legacy JSON: %w", err)
	}
	err := s.Write(ctx, "legacy-import", func(tx Transaction) error {
		t := tx.(*postgresTransaction)
		hash := pkg.Hash(string(b))
		done, err := t.q.HasLegacyImport(ctx, hash)
		if err != nil {
			return err
		}
		if done {
			result.AlreadyImported = true
			return nil
		}
		empty, err := t.q.DatabaseEmpty(ctx)
		if err != nil {
			return err
		}
		if !empty.Bool {
			return fmt.Errorf("import requires an empty application database; existing data was preserved")
		}
		for key, u := range st.Users {
			if key != u.ID || u.ID == "" || u.Phone == "" {
				return fmt.Errorf("invalid user in legacy data")
			}
			if u.CreatedAt.IsZero() {
				u.CreatedAt = time.Now()
			}
			if u.AvatarConfig != nil && !u.AvatarConfig.Valid() {
				return fmt.Errorf("invalid legacy avatar")
			}
			if err = tx.SaveUser(u); err != nil {
				return err
			}
			result.Users++
		}
		for key, r := range st.Rooms {
			if key != r.ID || r.ID == "" {
				return fmt.Errorf("invalid room in legacy data")
			}
			r.Visibility = domain.RoomVisibility(r.Visibility)
			r.Theme = domain.RoomTheme(r.Theme)
			if r.Revision < 1 {
				r.Revision = 1
			}
			if r.CreatedAt.IsZero() {
				r.CreatedAt = time.Now()
			}
			r.LastActivityAt = domain.RoomActivity(r)
			if err = tx.SaveRoom(r); err != nil {
				return err
			}
			result.Rooms++
		}
		for hash, session := range st.Sessions {
			if err = tx.SaveSession(hash, session); err != nil {
				return err
			}
			result.Sessions++
		}
		for phone, o := range st.OTPs {
			if err = tx.SaveOTP(phone, o); err != nil {
				return err
			}
			result.OTPs++
		}
		return t.q.RecordLegacyImport(ctx, hash)
	})
	return result, err
}

func (s *PostgresStore) LegacyJSON(ctx context.Context) ([]byte, error) {
	var b []byte
	// This retired table is intentionally outside the current sqlc schema.
	err := s.pool.QueryRow(ctx, "SELECT data FROM poker_state WHERE id=1").Scan(&b)
	return b, err
}
