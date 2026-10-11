package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureTask supplies a bounded fictional task with no external dependencies.
func fixtureTask(id string, deps ...string) Task {
	return Task{ID: id, Title: "Synthetic task", Phase: "Proof", DependsOn: deps,
		StoryPoints: 3, EstimateConfidence: "medium", Docs: []string{"docs/context.md"},
		Outcome: "Synthetic outcome", Acceptance: []string{"Prove result"},
		Validation: []string{"Synthetic test"}, Exclusions: []string{"No live data"}}
}

// TestGraphRejectsInvalidDependencies prevents the fail-open readiness pattern observed upstream.
func TestGraphRejectsInvalidDependencies(t *testing.T) {
	cases := map[string][]Task{
		"missing":   {fixtureTask("JB-001", "JB-999")},
		"self":      {fixtureTask("JB-001", "JB-001")},
		"cycle":     {fixtureTask("JB-001", "JB-002"), fixtureTask("JB-002", "JB-001")},
		"duplicate": {fixtureTask("JB-001"), fixtureTask("JB-001")},
		"repeated":  {fixtureTask("JB-001"), fixtureTask("JB-002", "JB-001", "JB-001")},
		"bad id":    {fixtureTask("../../escape")},
		"empty":     {},
	}
	for name, tasks := range cases {
		t.Run(name, func(t *testing.T) {
			if validateGraph(tasks) == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
	if err := validateGraph([]Task{fixtureTask("JB-002", "JB-001"), fixtureTask("JB-001")}); err != nil {
		t.Fatal(err)
	}
}

// TestScopeAndSchemaRejectDrift prevents an unreviewed status mirror or incomplete work definition.
func TestScopeAndSchemaRejectDrift(t *testing.T) {
	task := fixtureTask("JB-001")
	task.Validation = nil
	if validateGraph([]Task{task}) == nil {
		t.Fatal("missing validation accepted")
	}
	for _, data := range []string{`[{"id":"JB-001","status":"done"}]`, `[] {}`} {
		var tasks []Task
		if decode([]byte(data), &tasks) == nil {
			t.Fatal("unknown field or trailing value accepted")
		}
	}
}

// TestContextBoundary rejects traversal, protected paths, oversized documents and symlinks.
func TestContextBoundary(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("# Synthetic\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.md", "docs/../outside.md", ".env", "docs/sample.secret.md", "docs/.env.md", "docs/credentials.md"} {
		if _, err := readDocument(root, path); err == nil {
			t.Fatalf("accepted protected path %s", path)
		}
	}
	if err := os.Symlink(filepath.Join(root, "docs", "context.md"), filepath.Join(root, "docs", "alias.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := readDocument(root, "docs/alias.md"); err == nil {
		t.Fatal("accepted symlink")
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "large.md"), bytes.Repeat([]byte("x"), maxDocumentBytes+1), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readDocument(root, "docs/large.md"); err == nil {
		t.Fatal("accepted oversized document")
	}
}

// TestRegistryBindsRepository refuses identity drift or two seed tasks claiming one issue.
func TestRegistryBindsRepository(t *testing.T) {
	tasks := []Task{fixtureTask("JB-001"), fixtureTask("JB-002")}
	valid := Issue{Number: 1, URL: "https://github.com/" + repository + "/issues/1"}
	if err := validateRegistry(tasks, map[string]Issue{"JB-001": valid}); err != nil {
		t.Fatal(err)
	}
	for _, registry := range []map[string]Issue{
		{"JB-999": valid},
		{"JB-001": valid, "JB-002": valid},
		{"JB-001": {Number: 1, URL: "https://github.com/other/repo/issues/1"}},
	} {
		if validateRegistry(tasks, registry) == nil {
			t.Fatal("invalid registry accepted")
		}
	}
}

// fixtureRoot writes only synthetic project inputs for end-to-end CLI tests.
func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"docs", "planning"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	backlog, err := json.Marshal([]Task{fixtureTask("JB-001")})
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string][]byte{
		"planning/backlog.json": backlog,
		"planning/issues.json":  []byte("{}"),
		"docs/context.md":       []byte("# Synthetic context\n"),
	} {
		if err := os.WriteFile(filepath.Join(root, path), content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestCLIIsOfflineAndDeterministic proves packet/draft behavior without cloud or issue mutations.
func TestCLIIsOfflineAndDeterministic(t *testing.T) {
	root := fixtureRoot(t)
	var first, second bytes.Buffer
	if err := run(root, []string{"check"}, &first); err != nil {
		t.Fatal(err)
	}
	first.Reset()
	for _, output := range []*bytes.Buffer{&first, &second} {
		if err := run(root, []string{"packet", "JB-001"}, output); err != nil {
			t.Fatal(err)
		}
	}
	if first.String() != second.String() || !strings.Contains(first.String(), "SEED CONTEXT ONLY") {
		t.Fatal("packet is nondeterministic or claims readiness")
	}
	if err := run(root, []string{"drafts"}, &first); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".local", "issues", "JB-001.md"))
	if err != nil || !bytes.Contains(data, []byte("<!-- jayburd-seed:JB-001 -->")) {
		t.Fatal("missing stable adoption marker")
	}
	if run(root, []string{"packet", "JB-999"}, &first) == nil {
		t.Fatal("unknown task accepted")
	}
}

// TestBrokenLinkFails closes the gap between task-context existence and navigable docs.
func TestBrokenLinkFails(t *testing.T) {
	root := fixtureRoot(t)
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("[missing](absent.md)"), 0644); err != nil {
		t.Fatal(err)
	}
	if checkLinks(root) == nil {
		t.Fatal("broken link accepted")
	}
}

// TestDraftSymlinkRejected ensures generated issue output cannot redirect outside its directory.
func TestDraftSymlinkRejected(t *testing.T) {
	root := fixtureRoot(t)
	if err := os.Symlink(t.TempDir(), filepath.Join(root, ".local")); err != nil {
		t.Fatal(err)
	}
	if writeDrafts(root, []Task{fixtureTask("JB-001")}, nil) == nil {
		t.Fatal("symlink output directory accepted")
	}
}
