package leftoff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceAddAndScanSavesSafeGitMetadata(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available on PATH")
	}

	store := fixedStore(t)
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "leftoff@example.invalid")
	runGit(t, repo, "config", "user.name", "leftoff test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "Initial workspace commit")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# sample\n\nchanged\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	added, err := store.AddWorkspaceRepo(context.Background(), WorkspaceAddRequest{RepoPath: repo})
	if err != nil {
		t.Fatal(err)
	}
	if added.Repository.Path == "" || added.Repository.ProjectSlug == "" {
		t.Fatalf("expected repository metadata, got %#v", added.Repository)
	}
	if _, err := os.Stat(added.RegistryPath); err != nil {
		t.Fatalf("expected workspace registry: %v", err)
	}

	scan, err := store.ScanWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Repositories) != 1 {
		t.Fatalf("expected one workspace scan item, got %#v", scan.Repositories)
	}
	item := scan.Repositories[0]
	if item.Snapshot.WorktreeStatus != "dirty" {
		t.Fatalf("expected dirty worktree, got %#v", item.Snapshot)
	}
	if item.StatePath == "" {
		t.Fatalf("expected saved state path")
	}
	content := readFile(t, item.StatePath)
	assertContains(t, content, "Initial workspace commit")
	assertContains(t, content, "Repository position")
	assertContains(t, content, "Unpushed commits")
	if strings.Contains(content, "# sample") {
		t.Fatalf("source contents were persisted:\n%s", content)
	}
}

func TestValidateRepairRedactsWorkspaceMetadata(t *testing.T) {
	store := fixedStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	secret := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	path := store.workspaceScanCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"repository":"leaked `+secret+`"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	issues, err := store.Validate(ValidateOptions{Repair: true})
	if err != nil {
		t.Fatal(err)
	}
	var repaired ValidationIssue
	for _, issue := range issues {
		if issue.Path == path {
			repaired = issue
			break
		}
	}
	if !repaired.Repaired || repaired.BackupPath == "" {
		t.Fatalf("expected repaired workspace metadata issue, got %#v", repaired)
	}
	content := readFile(t, path)
	if strings.Contains(content, secret) {
		t.Fatalf("secret remained after repair:\n%s", content)
	}
	assertContains(t, content, "[redacted GitHub token]")
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}
