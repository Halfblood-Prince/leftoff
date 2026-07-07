package leftoff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanupProtectedBranchIsNotDeletionCandidate(t *testing.T) {
	branch := BranchInfo{
		Name:      "main",
		Date:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Protected: true,
		Stale:     true,
	}
	findings := analyzeBranchForCleanup(branch, time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if len(findings) != 1 {
		t.Fatalf("expected protected branch finding")
	}
	if findings[0].Risk != "info" {
		t.Fatalf("expected info risk, got %s", findings[0].Risk)
	}
	if findings[0].CommandPreview != "" {
		t.Fatalf("protected branch should not have deletion preview")
	}
}

func TestCleanupUnpushedBranchIsHighRisk(t *testing.T) {
	branch := BranchInfo{
		Name:     "feature/local-only",
		Date:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Unpushed: true,
		Stale:    true,
	}
	findings := analyzeBranchForCleanup(branch, time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if len(findings) != 1 {
		t.Fatalf("expected unpushed branch finding")
	}
	assertContains(t, findings[0].Title, "unpushed")
	if findings[0].Risk != "high" {
		t.Fatalf("expected high risk, got %s", findings[0].Risk)
	}
	if findings[0].CommandPreview != "" {
		t.Fatalf("unpushed branch should not have deletion preview")
	}
}

func TestCleanupDirtyWorktreeIsHighRisk(t *testing.T) {
	findings := analyzeWorktreeForCleanup(WorktreeCleanupInfo{
		Path:   filepath.Join("tmp", "worktree"),
		Branch: "feature/dirty",
		Exists: true,
		Dirty:  true,
	})
	if len(findings) != 1 {
		t.Fatalf("expected dirty worktree finding")
	}
	if findings[0].Risk != "high" {
		t.Fatalf("expected high risk, got %s", findings[0].Risk)
	}
	if findings[0].CommandPreview != "" {
		t.Fatalf("dirty worktree should not have cleanup preview")
	}
}

func TestCleanupStaleSyncedBranchHasPreviewOnly(t *testing.T) {
	branch := BranchInfo{
		Name:     "feature/old",
		Date:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Upstream: "origin/feature/old",
		Hash:     "abcdef123456",
		Subject:  "old branch",
		Stale:    true,
	}
	findings := analyzeBranchForCleanup(branch, time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if len(findings) != 1 {
		t.Fatalf("expected stale branch candidate")
	}
	if findings[0].Risk != "medium" {
		t.Fatalf("expected medium risk, got %s", findings[0].Risk)
	}
	assertContains(t, findings[0].CommandPreview, "git branch -d feature/old")
}

func TestCleanUpApplyDedupeActivityBacksUp(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "sample", Slug: "sample", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}
	line := `{"timestamp":"2026-07-06T13:00:00+02:00","kind":"capture","record_id":"TASK-2026-07-06-001","record_type":"task","project":"sample","summary":"x","evidence":"User capture."}`
	if err := os.WriteFile(paths.Activity, []byte(line+"\n"+line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := store.CleanUp(context.Background(), CleanUpRequest{Project: "sample", Apply: true, Confirm: true, Action: "dedupe-activity"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected one applied cleanup, got %#v", result.Applied)
	}
	content := readFile(t, paths.Activity)
	if strings.Count(content, line) != 1 {
		t.Fatalf("expected duplicate activity to be removed")
	}
}
