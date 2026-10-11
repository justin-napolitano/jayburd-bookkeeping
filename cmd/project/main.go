// Project validates repository planning inputs and renders bounded, offline context.
// It never calls GitHub, reads credentials, schedules work, or changes issue state.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const repository = "justin-napolitano/jayburd-bookkeeping"
const maxDocumentBytes = 32768

var taskID = regexp.MustCompile(`^JB-[0-9]{3}$`)
var localLink = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

// Task is a publication proposal, deliberately without a mutable execution status.
type Task struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Phase              string   `json:"phase"`
	DependsOn          []string `json:"depends_on"`
	StoryPoints        int      `json:"story_points"`
	EstimateConfidence string   `json:"estimate_confidence"`
	Docs               []string `json:"docs"`
	Outcome            string   `json:"outcome"`
	Acceptance         []string `json:"acceptance"`
	Validation         []string `json:"validation"`
	Exclusions         []string `json:"exclusions"`
}

// Issue stores identity only; current scope, state and dependencies live on GitHub.
type Issue struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// decode rejects unknown fields and trailing JSON so schema mistakes fail closed.
func decode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

// validateGraph rejects incomplete, ambiguous or cyclic task dependencies.
func validateGraph(tasks []Task) error {
	if len(tasks) == 0 {
		return errors.New("empty backlog")
	}
	byID := map[string]Task{}
	for _, t := range tasks {
		if !taskID.MatchString(t.ID) || byID[t.ID].ID != "" {
			return fmt.Errorf("invalid or duplicate task identity %q", t.ID)
		}
		if strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Outcome) == "" || t.Phase == "" ||
			len(t.Docs) == 0 || len(t.Acceptance) == 0 || len(t.Validation) == 0 || len(t.Exclusions) == 0 {
			return fmt.Errorf("%s lacks scope, context, acceptance, validation or exclusions", t.ID)
		}
		for _, values := range [][]string{t.Docs, t.Acceptance, t.Validation, t.Exclusions} {
			for _, value := range values {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("%s contains an empty required item", t.ID)
				}
			}
		}
		if t.StoryPoints != 1 && t.StoryPoints != 2 && t.StoryPoints != 3 && t.StoryPoints != 5 && t.StoryPoints != 8 {
			return fmt.Errorf("%s needs a bounded Fibonacci estimate (1..8)", t.ID)
		}
		if t.EstimateConfidence != "low" && t.EstimateConfidence != "medium" && t.EstimateConfidence != "high" {
			return fmt.Errorf("%s has invalid estimate confidence", t.ID)
		}
		byID[t.ID] = t
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		t, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown dependency %s", id)
		}
		if visiting[id] {
			return fmt.Errorf("dependency cycle at %s", id)
		}
		if done[id] {
			return nil
		}
		visiting[id] = true
		seen := map[string]bool{}
		for _, dep := range t.DependsOn {
			if seen[dep] {
				return fmt.Errorf("%s repeats dependency %s", id, dep)
			}
			seen[dep] = true
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id], done[id] = false, true
		return nil
	}
	for _, t := range tasks {
		if err := visit(t.ID); err != nil {
			return err
		}
	}
	return nil
}

// safeFile accepts only regular files beneath the root with no symlink components.
func safeFile(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes repository")
	}
	current := root
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("missing path %s", rel)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("symlink paths are not accepted")
		}
	}
	info, err := os.Stat(current)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("expected a regular file")
	}
	return current, nil
}

// readDocument reads only bounded Markdown context, excluding secret-shaped paths.
func readDocument(root, rel string) ([]byte, error) {
	if !strings.HasPrefix(rel, "docs/") || !strings.HasSuffix(rel, ".md") || strings.Contains(rel, "\\") {
		return nil, errors.New("context must be Markdown beneath docs/")
	}
	for _, part := range strings.Split(rel, "/") {
		lower := strings.ToLower(part)
		if strings.HasPrefix(part, ".") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
			return nil, errors.New("protected context path")
		}
	}
	path, err := safeFile(root, rel)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxDocumentBytes+1))
	if len(data) > maxDocumentBytes {
		return nil, errors.New("context document exceeds byte budget")
	}
	return data, err
}

// validateRegistry refuses mismatched repositories, unknown tasks and reused numbers.
func validateRegistry(tasks []Task, registry map[string]Issue) error {
	known, seen := map[string]bool{}, map[int]bool{}
	for _, t := range tasks {
		known[t.ID] = true
	}
	for id, issue := range registry {
		expected := fmt.Sprintf("https://github.com/%s/issues/%d", repository, issue.Number)
		if !known[id] || issue.Number < 1 || issue.URL != expected || seen[issue.Number] {
			return fmt.Errorf("invalid issue identity for %s", id)
		}
		seen[issue.Number] = true
	}
	return nil
}

// load validates the graph and its allowlisted context before any command runs.
func load(root string) ([]Task, map[string]Issue, error) {
	var tasks []Task
	registry := map[string]Issue{}
	for rel, target := range map[string]any{"planning/backlog.json": &tasks, "planning/issues.json": &registry} {
		path, err := safeFile(root, rel)
		if err != nil {
			return nil, nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		if err = decode(data, target); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", rel, err)
		}
	}
	if err := validateGraph(tasks); err != nil {
		return nil, nil, err
	}
	if err := validateRegistry(tasks, registry); err != nil {
		return nil, nil, err
	}
	for _, t := range tasks {
		for _, doc := range t.Docs {
			if _, err := readDocument(root, doc); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", t.ID, err)
			}
		}
	}
	return tasks, registry, nil
}

// checkLinks checks local Markdown file targets, not remote URLs or heading anchors.
func checkLinks(root string) error {
	return filepath.WalkDir(filepath.Join(root, "docs"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("docs must not contain symlinks")
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, err := readDocument(root, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		for _, match := range localLink.FindAllSubmatch(data, -1) {
			target := strings.SplitN(string(match[1]), "#", 2)[0]
			if target == "" || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "http://") {
				continue
			}
			local := filepath.Join(filepath.Dir(rel), target)
			if _, err := safeFile(root, local); err != nil {
				return fmt.Errorf("broken local link in %s: %s", rel, target)
			}
		}
		return nil
	})
}

// render produces a one-time issue proposal with a stable adoption marker.
func render(t Task, registry map[string]Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- jayburd-seed:%s -->\n\n## Outcome\n\n%s\n\n", t.ID, t.Outcome)
	fmt.Fprintf(&b, "Phase: %s. Initial estimate: %d points (%s confidence); not a time promise.\n\n", t.Phase, t.StoryPoints, t.EstimateConfidence)
	b.WriteString("## Acceptance criteria\n\n")
	for _, item := range t.Acceptance {
		fmt.Fprintf(&b, "- [ ] %s\n", item)
	}
	b.WriteString("\n## Validation\n\n")
	for _, item := range t.Validation {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Dependencies\n\n")
	if len(t.DependsOn) == 0 {
		b.WriteString("No initial predecessors. Confirm live issue state before implementation.\n")
	}
	for _, dep := range t.DependsOn {
		if issue, ok := registry[dep]; ok {
			fmt.Fprintf(&b, "- %s: %s\n", dep, issue.URL)
		} else {
			fmt.Fprintf(&b, "- %s (resolve to native blocked-by relationship on publication)\n", dep)
		}
	}
	b.WriteString("\n## Context\n\n")
	for _, doc := range t.Docs {
		fmt.Fprintf(&b, "- [%s](https://github.com/%s/blob/main/%s)\n", doc, repository, doc)
	}
	b.WriteString("\n## Exclusions\n\n")
	for _, item := range t.Exclusions {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Completion and handoff\n\nRecord tests/review evidence, runbook and PR/commit. Close as completed only\nafter acceptance and merge/review. Canceled predecessors do not unblock work.\n")
	return b.String()
}

// writeDrafts creates generated proposals locally without overwriting symlink targets.
func writeDrafts(root string, tasks []Task, registry map[string]Issue) error {
	dir := root
	for _, part := range []string{".local", "issues"} {
		dir = filepath.Join(dir, part)
		if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("draft directory must be a real local directory")
		}
	}
	for _, t := range tasks {
		path := filepath.Join(dir, t.ID+".md")
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			return errors.New("draft target must be a regular file")
		}
		if err := os.WriteFile(path, []byte(render(t, registry)), 0644); err != nil {
			return err
		}
	}
	return nil
}

// run keeps all commands offline; packets never assert current execution readiness.
func run(root string, args []string, out io.Writer) error {
	if len(args) < 1 {
		return errors.New("usage: project check|backlog|packet ID|drafts (run at repo root)")
	}
	tasks, registry, err := load(root)
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		if err := checkLinks(root); err != nil {
			return err
		}
		fmt.Fprintf(out, "PASS: %d issue seeds, acyclic dependencies, context paths and local doc links; %d published identities\n", len(tasks), len(registry))
	case "backlog":
		fmt.Fprintln(out, "Initial proposals only; GitHub owns live state. No readiness inferred.")
		for _, t := range tasks {
			fmt.Fprintf(out, "%s  %s  predecessors=%v\n", t.ID, t.Title, t.DependsOn)
		}
	case "drafts":
		if err := writeDrafts(root, tasks, registry); err != nil {
			return err
		}
		fmt.Fprintf(out, "Rendered %d proposals in .local/issues; nothing published.\n", len(tasks))
	case "packet":
		if len(args) != 2 {
			return errors.New("packet requires exactly one seed task ID")
		}
		for _, t := range tasks {
			if t.ID != args[1] {
				continue
			}
			var packet strings.Builder
			fmt.Fprintf(&packet, "# %s: %s\n\nSEED CONTEXT ONLY. Verify current GitHub issue, blockers, scope and evidence.\n", t.ID, t.Title)
			if issue, ok := registry[t.ID]; ok {
				fmt.Fprintf(&packet, "Current issue: %s\n", issue.URL)
			}
			packet.WriteString("\n" + render(t, registry))
			docs := append([]string(nil), t.Docs...)
			sort.Strings(docs)
			for _, doc := range docs {
				data, err := readDocument(root, doc)
				if err != nil {
					return err
				}
				fmt.Fprintf(&packet, "\n---\nSource: %s\n\n%s", doc, data)
			}
			if packet.Len() > maxDocumentBytes {
				return errors.New("packet exceeds byte budget; narrow its context")
			}
			_, err := io.WriteString(out, packet.String())
			return err
		}
		return errors.New("unknown task ID")
	default:
		return errors.New("unknown command")
	}
	return nil
}

// main exposes only redacted configuration-free project operations.
func main() {
	if err := run(".", os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED:", err)
		os.Exit(1)
	}
}
