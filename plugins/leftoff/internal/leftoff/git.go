package leftoff

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func InspectRepository(ctx context.Context, repoPath string, clock func() time.Time) GitSnapshot {
	now := time.Now
	if clock != nil {
		now = clock
	}

	snapshot := GitSnapshot{
		Available:   true,
		InspectedAt: now(),
	}

	if strings.TrimSpace(repoPath) == "" {
		repoPath = "."
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		snapshot.Available = false
		snapshot.HealthNotes = append(snapshot.HealthNotes, "repository path could not be resolved")
		return snapshot
	}
	snapshot.Worktree = filepath.Clean(abs)

	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		snapshot.Available = false
		snapshot.HealthNotes = append(snapshot.HealthNotes, "repository path is not a readable directory")
		return snapshot
	}

	if _, err := exec.LookPath("git"); err != nil {
		snapshot.Available = false
		snapshot.HealthNotes = append(snapshot.HealthNotes, "Git is not available on PATH")
		return snapshot
	}

	root, ok := runGitText(ctx, abs, &snapshot, "rev-parse", "--show-toplevel")
	if !ok {
		snapshot.IsRepo = false
		snapshot.HealthNotes = append(snapshot.HealthNotes, "path is not a Git repository")
		return snapshot
	}

	snapshot.IsRepo = true
	snapshot.Root = filepath.Clean(strings.TrimSpace(root))
	snapshot.Worktree = snapshot.Root
	snapshot.RepoName = filepath.Base(snapshot.Root)

	if branch, ok := runGitText(ctx, snapshot.Root, &snapshot, "branch", "--show-current"); ok {
		snapshot.Branch = strings.TrimSpace(branch)
	}
	if snapshot.Branch == "" {
		snapshot.Branch = "detached or unknown"
	}

	if head, ok := runGitText(ctx, snapshot.Root, &snapshot, "rev-parse", "--short", "HEAD"); ok {
		snapshot.Head = strings.TrimSpace(head)
	}
	if snapshot.Head == "" {
		snapshot.Head = "unknown"
	}

	if remotes, ok := runGitText(ctx, snapshot.Root, &snapshot, "remote", "-v"); ok {
		snapshot.Remote = firstFetchRemote(remotes)
		if name := repoNameFromRemote(snapshot.Remote); name != "" {
			snapshot.RepoName = name
		}
	}

	if status, ok := runGitText(ctx, snapshot.Root, &snapshot, "status", "--short"); ok {
		changed, skipped := parseChangedFiles(status, snapshot.Root)
		snapshot.ChangedFiles = changed
		if skipped > 0 {
			snapshot.HealthNotes = append(snapshot.HealthNotes, fmt.Sprintf("excluded %d sensitive or ignored changed path(s)", skipped))
		}
	}
	if len(snapshot.ChangedFiles) > 0 {
		snapshot.WorktreeStatus = "dirty"
	} else {
		snapshot.WorktreeStatus = "clean"
	}

	if counts, ok := runGitText(ctx, snapshot.Root, &snapshot, "rev-list", "--left-right", "--count", "@{upstream}...HEAD"); ok {
		snapshot.Ahead, snapshot.Behind = parseAheadBehind(counts)
		snapshot.UnpushedCommits = snapshot.Ahead
	} else if unpushed, ok := runGitText(ctx, snapshot.Root, &snapshot, "log", "--branches", "--not", "--remotes", "--format=%H"); ok {
		snapshot.UnpushedCommits = countNonEmptyLines(unpushed)
	}

	if log, ok := runGitText(ctx, snapshot.Root, &snapshot, "log", "--oneline", "-n", "20"); ok {
		snapshot.RecentCommits = parseCommits(log)
	}
	if len(snapshot.RecentCommits) == 0 {
		snapshot.HealthNotes = append(snapshot.HealthNotes, "recent commit history is empty or unavailable")
	}

	if worktrees, ok := runGitText(ctx, snapshot.Root, &snapshot, "worktree", "list", "--porcelain"); ok {
		snapshot.Worktrees = parseWorktrees(worktrees)
	}
	if branches, ok := runGitText(ctx, snapshot.Root, &snapshot, "for-each-ref", "--format=%(refname:short)|%(committerdate:iso8601)", "refs/heads"); ok {
		snapshot.StaleBranches = parseStaleBranches(branches, now(), 45)
	}

	if len(snapshot.ChangedFiles) > 0 {
		snapshot.HealthNotes = append(snapshot.HealthNotes, fmt.Sprintf("working tree has %d changed path(s)", len(snapshot.ChangedFiles)))
	} else {
		snapshot.HealthNotes = append(snapshot.HealthNotes, "working tree has no changed paths reported by git status")
	}

	return snapshot
}

func runGitText(ctx context.Context, repoPath string, snapshot *GitSnapshot, args ...string) (string, bool) {
	command := "git -C " + sanitizeExternalMetadata(repoPath) + " " + strings.Join(args, " ")
	snapshot.Commands = append(snapshot.Commands, command)

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmdArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.CommandContext(cmdCtx, "git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return string(output), true
}

func firstFetchRemote(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == "(fetch)" {
			return RedactRemoteURL(fields[1])
		}
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			return RedactRemoteURL(fields[1])
		}
	}
	return ""
}

func repoNameFromRemote(remote string) string {
	remote = strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	if remote == "" || strings.HasPrefix(remote, "[redacted") {
		return ""
	}
	if slash := strings.LastIndex(remote, "/"); slash >= 0 && slash < len(remote)-1 {
		return remote[slash+1:]
	}
	if colon := strings.LastIndex(remote, ":"); colon >= 0 && colon < len(remote)-1 {
		return remote[colon+1:]
	}
	return ""
}

func parseChangedFiles(output string, repoRoot string) ([]ChangedFile, int) {
	rules := LoadLeftoffIgnore(repoRoot)
	var changed []ChangedFile
	skipped := 0

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		status := strings.TrimSpace(line[:min(2, len(line))])
		rel := strings.TrimSpace(line[min(3, len(line)):])
		if rel == "" {
			continue
		}
		if strings.Contains(rel, " -> ") {
			parts := strings.Split(rel, " -> ")
			if len(parts) == 2 {
				rel = strings.TrimSpace(parts[1])
			}
		}
		if ShouldExcludePath(rel, rules) {
			skipped++
			continue
		}
		changed = append(changed, ChangedFile{Status: status, Path: sanitizeExternalMetadata(filepath.ToSlash(rel))})
	}

	return changed, skipped
}

func parseCommits(output string) []Commit {
	var commits []Commit
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		hash := fields[0]
		summary := strings.TrimSpace(strings.TrimPrefix(line, hash))
		commits = append(commits, Commit{Hash: hash, Summary: cleanSummary(sanitizeExternalMetadata(summary), 160)})
	}
	return commits
}

func parseAheadBehind(output string) (int, int) {
	fields := strings.Fields(output)
	if len(fields) < 2 {
		return 0, 0
	}
	ahead, _ := strconv.Atoi(fields[0])
	behind, _ := strconv.Atoi(fields[1])
	if ahead < 0 {
		ahead = 0
	}
	if behind < 0 {
		behind = 0
	}
	return ahead, behind
}

func countNonEmptyLines(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func parseWorktrees(output string) []Worktree {
	var worktrees []Worktree
	current := Worktree{}
	flush := func() {
		if current.Path != "" {
			worktrees = append(worktrees, current)
		}
		current = Worktree{}
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current.Path = sanitizeExternalMetadata(strings.TrimSpace(strings.TrimPrefix(line, "worktree ")))
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		case strings.HasPrefix(line, "branch "):
			branch := strings.TrimSpace(strings.TrimPrefix(line, "branch "))
			current.Branch = sanitizeExternalMetadata(strings.TrimPrefix(branch, "refs/heads/"))
		case line == "detached":
			current.Branch = "detached"
		}
	}
	flush()
	return worktrees
}

func parseStaleBranches(output string, now time.Time, staleAfterDays int) []StaleBranch {
	var branches []StaleBranch
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		name := sanitizeExternalMetadata(parts[0])
		commitTime, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		ageDays := int(startOfDay(now).Sub(startOfDay(commitTime)).Hours() / 24)
		if ageDays < staleAfterDays {
			continue
		}
		branches = append(branches, StaleBranch{
			Name:       name,
			LastCommit: commitTime.Format(timeFormatRFC3339()),
			AgeDays:    ageDays,
		})
	}
	return branches
}

type IgnoreRules []string

func LoadLeftoffIgnore(repoRoot string) IgnoreRules {
	path := filepath.Join(repoRoot, ".leftoffignore")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var rules IgnoreRules
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rules = append(rules, filepath.ToSlash(line))
	}
	return rules
}

func ShouldExcludePath(rel string, rules IgnoreRules) bool {
	clean := filepath.ToSlash(strings.TrimSpace(rel))
	lower := strings.ToLower(clean)
	base := strings.ToLower(path.Base(lower))

	builtInDirs := []string{
		".git/",
		"node_modules/",
		"vendor/",
		".ssh/",
	}
	for _, dir := range builtInDirs {
		if strings.Contains(lower, dir) || strings.HasPrefix(lower, dir) {
			return true
		}
	}

	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.Contains(base, "private_key") {
		return true
	}
	if strings.Contains(lower, "credential") || strings.Contains(lower, "secret") {
		return true
	}

	for _, rule := range rules {
		rule = filepath.ToSlash(strings.TrimSpace(rule))
		if rule == "" {
			continue
		}
		if strings.HasSuffix(rule, "/") && strings.HasPrefix(clean, strings.TrimSuffix(rule, "/")+"/") {
			return true
		}
		if clean == strings.TrimPrefix(rule, "/") {
			return true
		}
		if ok, _ := path.Match(rule, clean); ok {
			return true
		}
		if ok, _ := path.Match(rule, base); ok {
			return true
		}
	}

	return false
}

func ProjectFromSnapshot(snapshot GitSnapshot) ProjectMeta {
	name := snapshot.RepoName
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(snapshot.Root)
	}
	return ProjectMeta{
		Name:      sanitizeExternalMetadata(name),
		Slug:      Slugify(name),
		Remote:    snapshot.Remote,
		LocalPath: sanitizeExternalMetadata(snapshot.Root),
	}
}

func (s *Store) SaveGitState(snapshot GitSnapshot) (string, error) {
	if !snapshot.IsRepo {
		return "", errors.New("cannot save Git state for a non-repository path")
	}
	snapshot = sanitizeGitSnapshot(snapshot)
	if err := s.Init(); err != nil {
		return "", err
	}

	meta := ProjectFromSnapshot(snapshot)
	meta.Created = s.now()
	paths, err := s.EnsureProject(meta)
	if err != nil {
		return "", err
	}

	if err := atomicWriteFile(paths.State, []byte(RenderGitState(snapshot)), 0o600); err != nil {
		return "", err
	}

	event := ActivityEvent{
		Timestamp:  s.now().Format(timeFormatRFC3339()),
		Kind:       "local_git_scan",
		RecordType: string(RecordActivityEvent),
		Project:    meta.Slug,
		Summary:    fmt.Sprintf("Captured branch %s at %s with %d changed path(s)", snapshot.Branch, snapshot.Head, len(snapshot.ChangedFiles)),
		Evidence:   "User-invoked read-only Git scan.",
	}
	if err := s.AppendJSONL(paths.Activity, event); err != nil {
		return "", err
	}

	return paths.State, nil
}

func RenderGitState(snapshot GitSnapshot) string {
	snapshot = sanitizeGitSnapshot(snapshot)

	var b strings.Builder
	fmt.Fprintf(&b, "# State\n\n")
	fmt.Fprintf(&b, "- Last updated: %s\n", snapshot.InspectedAt.Format(timeFormatRFC3339()))
	fmt.Fprintf(&b, "- Repository: %s\n", valueOr(snapshot.RepoName, "unknown"))
	fmt.Fprintf(&b, "- Remote: %s\n", valueOr(snapshot.Remote, "unknown"))
	fmt.Fprintf(&b, "- Worktree: %s\n", valueOr(snapshot.Worktree, "unknown"))
	fmt.Fprintf(&b, "- Branch: %s\n", valueOr(snapshot.Branch, "unknown"))
	fmt.Fprintf(&b, "- Head: %s\n", valueOr(snapshot.Head, "unknown"))
	fmt.Fprintf(&b, "- Dirty files: %d\n", len(snapshot.ChangedFiles))
	fmt.Fprintf(&b, "\n## Changed paths\n\n")
	if len(snapshot.ChangedFiles) == 0 {
		fmt.Fprintf(&b, "- none reported\n")
	} else {
		for _, file := range snapshot.ChangedFiles {
			fmt.Fprintf(&b, "- %s %s\n", valueOr(file.Status, "?"), file.Path)
		}
	}

	fmt.Fprintf(&b, "\n## Repository position\n\n")
	fmt.Fprintf(&b, "- Worktree status: %s\n", valueOr(snapshot.WorktreeStatus, "unknown"))
	fmt.Fprintf(&b, "- Ahead: %d\n", snapshot.Ahead)
	fmt.Fprintf(&b, "- Behind: %d\n", snapshot.Behind)
	fmt.Fprintf(&b, "- Unpushed commits: %d\n", snapshot.UnpushedCommits)

	fmt.Fprintf(&b, "\n## Recent commits\n\n")
	if len(snapshot.RecentCommits) == 0 {
		fmt.Fprintf(&b, "- none recorded\n")
	} else {
		for _, commit := range snapshot.RecentCommits {
			fmt.Fprintf(&b, "- %s %s\n", commit.Hash, commit.Summary)
		}
	}

	fmt.Fprintf(&b, "\n## Stale branches\n\n")
	if len(snapshot.StaleBranches) == 0 {
		fmt.Fprintf(&b, "- none recorded\n")
	} else {
		for _, branch := range snapshot.StaleBranches {
			fmt.Fprintf(&b, "- %s last commit %s (%d day(s) old)\n", branch.Name, branch.LastCommit, branch.AgeDays)
		}
	}

	fmt.Fprintf(&b, "\n## Worktrees\n\n")
	if len(snapshot.Worktrees) == 0 {
		fmt.Fprintf(&b, "- none recorded\n")
	} else {
		for _, worktree := range snapshot.Worktrees {
			fmt.Fprintf(&b, "- %s [%s] %s\n", worktree.Path, valueOr(worktree.Branch, "unknown"), valueOr(worktree.Head, "unknown"))
		}
	}

	fmt.Fprintf(&b, "\n## Health notes\n\n")
	if len(snapshot.HealthNotes) == 0 {
		fmt.Fprintf(&b, "- none\n")
	} else {
		for _, note := range snapshot.HealthNotes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}

	fmt.Fprintf(&b, "\n## Commands used\n\n")
	for _, command := range snapshot.Commands {
		fmt.Fprintf(&b, "- `%s`\n", command)
	}

	return b.String()
}

func sanitizeGitSnapshot(snapshot GitSnapshot) GitSnapshot {
	snapshot.Root = sanitizeExternalMetadata(snapshot.Root)
	snapshot.RepoName = sanitizeExternalMetadata(snapshot.RepoName)
	snapshot.Remote = sanitizeExternalMetadata(snapshot.Remote)
	snapshot.Branch = sanitizeExternalMetadata(snapshot.Branch)
	snapshot.Worktree = sanitizeExternalMetadata(snapshot.Worktree)
	snapshot.WorktreeStatus = sanitizeExternalMetadata(snapshot.WorktreeStatus)
	for i := range snapshot.ChangedFiles {
		snapshot.ChangedFiles[i].Path = sanitizeExternalMetadata(snapshot.ChangedFiles[i].Path)
	}
	for i := range snapshot.RecentCommits {
		snapshot.RecentCommits[i].Summary = sanitizeExternalMetadata(snapshot.RecentCommits[i].Summary)
	}
	for i := range snapshot.Worktrees {
		snapshot.Worktrees[i].Path = sanitizeExternalMetadata(snapshot.Worktrees[i].Path)
		snapshot.Worktrees[i].Branch = sanitizeExternalMetadata(snapshot.Worktrees[i].Branch)
	}
	for i := range snapshot.StaleBranches {
		snapshot.StaleBranches[i].Name = sanitizeExternalMetadata(snapshot.StaleBranches[i].Name)
	}
	for i := range snapshot.HealthNotes {
		snapshot.HealthNotes[i] = sanitizeExternalMetadata(snapshot.HealthNotes[i])
	}
	for i := range snapshot.Commands {
		snapshot.Commands[i] = sanitizeExternalMetadata(snapshot.Commands[i])
	}
	return snapshot
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
