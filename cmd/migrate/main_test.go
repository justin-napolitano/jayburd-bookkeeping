package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
)

// TestOfflineAndInvalidMigration proves rejected commands cannot construct an IAM transport.
func TestOfflineAndInvalidMigration(t *testing.T) {
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	prod := []string{"-action=up", "-environment=production", "-instance=fictional-project:us-central1:jayburd-prod"}
	for _, tc := range []struct {
		args []string
		ok   bool
	}{
		{nil, true}, {[]string{"-action=verify", "-environment=production"}, true},
		{[]string{"-environment=invalid"}, false}, {[]string{"-action=down"}, false},
		{[]string{"-unknown=secret-canary"}, false}, {[]string{"extra"}, false},
		{[]string{"-action=status", "-instance=fictional-project:us-central1:jayburd-prod"}, false},
		{prod, false}, {append(append([]string{}, prod...), "-approval-expires-at=2026-10-28T14:00:00Z"), false},
		{append(append([]string{}, prod...), "-approval-expires-at=2026-11-05T14:00:00Z"), false},
		{append(append([]string{}, prod...), "-approval-expires-at=2026-10-29T14:00:00+00:00"), false},
		{[]string{"-action=up", "-environment=production", "-instance=jayburd-bookkeeping-dev:us-central1:jayburd-prod", "-approval-expires-at=2026-10-29T14:00:00Z"}, false},
		{[]string{"-action=up", "-environment=production", "-instance=fictional-project:us-central1:jayburd-dev", "-approval-expires-at=2026-10-29T14:00:00Z"}, false},
	} {
		calls := 0
		err := runWith(tc.args, now, func(context.Context, clouddb.Config, string) (*pgxpool.Config, func(), error) {
			calls++
			return nil, nil, errors.New("unexpected")
		})
		if calls != 0 || (err == nil) != tc.ok {
			t.Fatalf("command accepted incorrectly: calls=%d ok=%v", calls, err == nil)
		}
	}
}

// TestMigrationProfileAndDeadline observes selected identity context without opening any database.
func TestMigrationProfileAndDeadline(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, action := range []string{"up", "status"} {
		for _, duration := range []time.Duration{time.Minute, 24 * time.Hour} {
			calls := 0
			expiry := now.Add(duration)
			err := runWith([]string{"-action=" + action, "-environment=production", "-instance=fictional-project:us-central1:jayburd-prod", "-approval-expires-at=" + expiry.Format("2006-01-02T15:04:05Z")}, now, func(ctx context.Context, c clouddb.Config, env string) (*pgxpool.Config, func(), error) {
				calls++
				want := expiry
				if duration > 10*time.Minute {
					want = now.Add(10 * time.Minute)
				}
				got, ok := ctx.Deadline()
				if !ok || !got.Equal(want) || env != "production" || c.Component != "migrator" {
					t.Fatal("incorrect profile/deadline")
				}
				return nil, nil, errors.New("synthetic transport stop")
			})
			if calls != 1 || err == nil {
				t.Fatal("missing transport boundary")
			}
		}
	}
}

// TestSandboxMigrationAndURLOverride retains legacy selection and denies inherited connection strings.
func TestSandboxMigrationAndURLOverride(t *testing.T) {
	now := time.Now()
	args := []string{"-action=status", "-instance=fictional-project:us-central1:jayburd-dev"}
	calls := 0
	connect := func(ctx context.Context, c clouddb.Config, env string) (*pgxpool.Config, func(), error) {
		calls++
		if env != "sandbox" || c.Component != "migrator" {
			t.Fatal("legacy profile changed")
		}
		return nil, nil, errors.New("synthetic transport stop")
	}
	if runWith(args, now, connect) == nil || calls != 1 {
		t.Fatal("legacy transport not selected")
	}
	t.Setenv("JAYBURD_DATABASE_URL", "")
	calls = 0
	if runWith(args, now, connect) == nil || calls != 0 {
		t.Fatal("URL override reached transport")
	}
}
