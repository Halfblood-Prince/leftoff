package leftoff

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Clock = func() time.Time {
		return time.Date(2026, 7, 6, 13, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	}
	return store
}

func TestInitCreatesLayout(t *testing.T) {
	store := fixedStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	required := []string{
		"profile.md",
		"config.yml",
		"inbox.md",
		"projects",
		"patterns/recurring-friction.md",
		"patterns/reusable-recipes.md",
		"weekly",
		"cache/local-scan-metadata.json",
		"workspace",
		"backups",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(store.Root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestCaptureTaskToInbox(t *testing.T) {
	store := fixedStore(t)
	result, err := store.Capture(context.Background(), CaptureRequest{Text: "task: Add Windows installation smoke test"})
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != "TASK-2026-07-06-001" {
		t.Fatalf("unexpected id: %s", result.ID)
	}
	if result.ProjectSlug != "" {
		t.Fatalf("expected inbox capture, got project %q", result.ProjectSlug)
	}

	content := readFile(t, filepath.Join(store.Root, "inbox.md"))
	assertContains(t, content, "## TASK-2026-07-06-001 - Add Windows installation smoke test")
	assertContains(t, content, "- Status: inbox")
	assertContains(t, content, "- Evidence: User capture.")
}

func TestCaptureDecisionToProject(t *testing.T) {
	store := fixedStore(t)
	result, err := store.Capture(context.Background(), CaptureRequest{
		Project: "leftoff",
		Text:    "decision: Use Markdown and JSONL because records must stay editable",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != "DECISION-2026-07-06-001" {
		t.Fatalf("unexpected id: %s", result.ID)
	}
	if result.ProjectSlug != "leftoff" {
		t.Fatalf("unexpected project slug: %s", result.ProjectSlug)
	}

	decisionFile := filepath.Join(store.Root, "projects", "leftoff", "decisions.md")
	content := readFile(t, decisionFile)
	assertContains(t, content, "- Type: decision")
	assertContains(t, content, "- Status: accepted")

	activity := readFile(t, filepath.Join(store.Root, "projects", "leftoff", "activity.jsonl"))
	assertContains(t, activity, `"kind":"capture"`)
	assertContains(t, activity, `"record_type":"decision"`)
}

func TestCaptureRejectsSecret(t *testing.T) {
	store := fixedStore(t)
	_, err := store.Capture(context.Background(), CaptureRequest{Text: "task: rotate token=super-secret-value"})
	if !errors.Is(err, ErrSecretCapture) {
		t.Fatalf("expected ErrSecretCapture, got %v", err)
	}

	content := readFile(t, filepath.Join(store.Root, "inbox.md"))
	if strings.Contains(content, "super-secret-value") {
		t.Fatalf("secret was persisted")
	}
}

func TestCaptureDuplicateInput(t *testing.T) {
	store := fixedStore(t)
	req := CaptureRequest{Project: "sample-app", Text: "task: Add Windows installation smoke test"}
	if _, err := store.Capture(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	_, err := store.Capture(context.Background(), req)
	if !errors.Is(err, ErrDuplicateCapture) {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	content := readFile(t, filepath.Join(store.Root, "projects", "sample-app", "open-loops.md"))
	if count := strings.Count(content, "## TASK-2026-07-06-"); count != 1 {
		t.Fatalf("expected one stored record, got %d", count)
	}
}

func TestValidateRepairBacksUpMalformedMarkdown(t *testing.T) {
	store := fixedStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	inbox := filepath.Join(store.Root, "inbox.md")
	if err := os.WriteFile(inbox, []byte("not a heading\n\n- user text\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	issues, err := store.Validate(ValidateOptions{Repair: true})
	if err != nil {
		t.Fatal(err)
	}

	var repaired ValidationIssue
	for _, issue := range issues {
		if issue.Path == inbox {
			repaired = issue
			break
		}
	}
	if !repaired.Repaired || repaired.BackupPath == "" {
		t.Fatalf("expected repaired issue with backup, got %#v", repaired)
	}

	content := readFile(t, inbox)
	assertContains(t, content, "# leftoff inbox")
	assertContains(t, content, "not a heading")

	if _, err := os.Stat(repaired.BackupPath); err != nil {
		t.Fatalf("expected backup to exist: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
