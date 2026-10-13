package repositories

import (
	"context"
	"fmt"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/testutil"
	"sync"
	"testing"
	"time"
)

func TestPostgresPersistenceAndConcurrentTransactions(t *testing.T) {
	url := testutil.Postgres(t)
	ctx := context.Background()
	s, err := New(&config.Config{DatabaseURL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := s.Write(ctx, fmt.Sprintf("user:qa-%d", n), func(tx Transaction) error {
				key := fmt.Sprintf("qa-%d", n)
				return tx.SaveUser(domain.User{ID: key, Phone: key, Name: "PG QA", Avatar: "violet", CreatedAt: time.Now()})
			}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	second, err := New(&config.Config{DatabaseURL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err = second.Read(ctx, func(tx Transaction) error {
		for i := 0; i < 20; i++ {
			u, e := tx.User(fmt.Sprintf("qa-%d", i))
			if e != nil {
				return e
			}
			if u.Name != "PG QA" {
				t.Errorf("lost transaction %d", i)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Domain errors still commit OTP attempt counters, by repository contract.
	wanted := domain.Fail(401, "attempt", "attempt")
	_ = s.Write(ctx, "phone:qa", func(tx Transaction) error {
		now := time.Now()
		if err := tx.SaveOTP("qa", domain.OTP{Attempts: 3, SentAt: now, ExpiresAt: now.Add(time.Minute), WindowStart: now}); err != nil {
			return err
		}
		return wanted
	})
	if err = second.Read(ctx, func(tx Transaction) error {
		o, e := tx.OTP("qa")
		if e != nil {
			return e
		}
		if o.Attempts != 3 {
			t.Fatal("OTP attempt was rolled back")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
