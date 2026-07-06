package leftoff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareSavedStateDetectsChangedBranchAndHead(t *testing.T) {
	saved := SavedState{
		Exists: true,
		Branch: "main",
		Head:   "abc123",
	}
	snapshot := GitSnapshot{
		Available: true,
		IsRepo:    true,
		Branch:    "feature/resume",
		Head:      "def456",
		Worktree:  t.TempDir(),
	}

	changes := strings.Join(CompareSavedState(saved, snapshot), "\n")
	assertContains(t, changes, "Branch changed from main to feature/resume.")
	assertContains(t, changes, "Head changed from abc123 to def456.")
}

func TestCompareSavedStateDetectsDeletedWorktree(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "deleted-worktree")
	saved := SavedState{
		Exists:   true,
		Worktree: missing,
		Branch:   "main",
		Head:     "abc123",
	}
	snapshot := GitSnapshot{
		Available: true,
		IsRepo:    false,
	}

	changes := strings.Join(CompareSavedState(saved, snapshot), "\n")
	assertContains(t, changes, "Saved worktree no longer exists: "+missing)
}

func TestResumeNoGitRepository(t *testing.T) {
	store := fixedStore(t)
	repo := t.TempDir()

	result, err := store.Resume(context.Background(), ResumeRequest{RepoPath: repo})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Output, "not a Git repository") && !strings.Contains(result.Output, "Git context is unavailable") {
		t.Fatalf("expected no-git context in output:\n%s", result.Output)
	}
	assertContains(t, result.Output, "What remains uncertain")
}

func TestResumeOutputHandlesEmptyHistory(t *testing.T) {
	store := fixedStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	output := renderResume(
		"sample",
		"",
		GitSnapshot{Available: true, IsRepo: true, RepoName: "sample", Branch: "main", Head: "abc123", Worktree: t.TempDir()},
		SavedState{},
		nil,
		nil,
		nil,
		nil,
		true,
		"",
	)

	assertContains(t, output, "Recent commits: none recorded")
	assertContains(t, output, "No saved project state was found.")
	assertContains(t, output, "Safe commands to run")
}

func TestSaveGitStateWritesCompactMetadataOnly(t *testing.T) {
	store := fixedStore(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot := GitSnapshot{
		Available:   true,
		IsRepo:      true,
		InspectedAt: store.now(),
		Root:        root,
		RepoName:    "sample",
		Remote:      "https://github.com/example/sample.git",
		Branch:      "main",
		Head:        "abc123",
		Worktree:    root,
		ChangedFiles: []ChangedFile{
			{Status: "M", Path: "main.go"},
		},
		RecentCommits: []Commit{
			{Hash: "abc123", Summary: "initial commit"},
		},
		Commands: []string{"git status --short"},
	}

	path, err := store.SaveGitState(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	content := readFile(t, path)
	assertContains(t, content, "- Branch: main")
	assertContains(t, content, "- M main.go")
	if strings.Contains(content, "package main") {
		t.Fatalf("source contents were persisted")
	}
}

func TestSaveGitStateRedactsSecretInCommitTitle(t *testing.T) {
	store := fixedStore(t)
	secret := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	snapshot := GitSnapshot{
		Available:   true,
		IsRepo:      true,
		InspectedAt: store.now(),
		Root:        t.TempDir(),
		RepoName:    "sample",
		Remote:      "https://github.com/example/sample.git",
		Branch:      "feature/" + secret,
		Head:        "abc123",
		Worktree:    t.TempDir(),
		ChangedFiles: []ChangedFile{
			{Status: "M", Path: "config/" + secret + ".txt"},
		},
		RecentCommits: []Commit{
			{Hash: "abc123", Summary: "fix release with token " + secret},
		},
		Commands: []string{"git status --short " + secret},
	}

	path, err := store.SaveGitState(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	content := readFile(t, path)
	if strings.Contains(content, secret) {
		t.Fatalf("secret-like external metadata was persisted:\n%s", content)
	}
	assertContains(t, content, "[redacted GitHub token]")
}
