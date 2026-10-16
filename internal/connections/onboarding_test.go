package connections

import (
	"context"
	"testing"
)

// TestCreatorOnboardingRouting proves runtime logins never come from Begin JSON.
func TestCreatorOnboardingRouting(t *testing.T) {
	s, repo, provider, _ := fixture()
	s.creatorLogins = &WorkerLogins{Sync: "synthetic_sync", Publisher: "synthetic_publisher"}
	if _, err := s.List(context.Background(), scope); err != nil {
		t.Fatal(err)
	}
	if len(provider.calls) != 0 || repo.calls[0] != "creator_list_connections" {
		t.Fatal("discovery contacted provider or used unguarded SQL")
	}
	begin(t, s)
	if repo.calls[1] != "creator_begin_connection" || count(provider.calls, "link") != 1 {
		t.Fatal("creator preparation did not precede provider handoff")
	}
}

// TestCreatorOnboardingRejectsSQLFailure prevents provider calls after denied/unknown preparation.
func TestCreatorOnboardingRejectsSQLFailure(t *testing.T) {
	s, repo, provider, _ := fixture()
	s.creatorLogins = &WorkerLogins{Sync: "synthetic_sync", Publisher: "synthetic_publisher"}
	repo.failOp = "creator_begin_connection"
	_, err := s.Begin(context.Background(), scope, "new", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	code(t, err, "commit_unknown")
	if len(provider.calls) != 0 {
		t.Fatal("provider contacted after uncertain authorization")
	}
}

// TestCreatorWorkerConfiguration rejects ambiguous runtime login names before any I/O.
func TestCreatorWorkerConfiguration(t *testing.T) {
	for _, logins := range []WorkerLogins{{}, {Sync: "same", Publisher: "same"}, {Sync: "invalid/name", Publisher: "publisher"}} {
		if _, err := NewForBookCreators(nil, nil, nil, logins); err == nil {
			t.Fatal("unsafe worker configuration accepted")
		}
	}
}
