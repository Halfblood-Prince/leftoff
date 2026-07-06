package leftoff

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResumeRequest struct {
	Project   string `json:"project,omitempty"`
	RepoPath  string `json:"repo_path,omitempty"`
	SaveState bool   `json:"save_state,omitempty"`
}

type ResumeResult struct {
	Output         string      `json:"output"`
	ProjectSlug    string      `json:"project_slug,omitempty"`
	StateSavedPath string      `json:"state_saved_path,omitempty"`
	Snapshot       GitSnapshot `json:"snapshot"`
}

func (s *Store) Resume(ctx context.Context, req ResumeRequest) (ResumeResult, error) {
	if err := s.Init(); err != nil {
		return ResumeResult{}, err
	}

	projectSlug := Slugify(req.Project)
	repoPath := strings.TrimSpace(req.RepoPath)
	shouldInspect := repoPath != "" || projectSlug == ""
	if shouldInspect && repoPath == "" {
		repoPath = "."
	}

	var snapshot GitSnapshot
	var haveSnapshot bool
	if shouldInspect {
		snapshot = InspectRepository(ctx, repoPath, s.now)
		haveSnapshot = true
		if snapshot.IsRepo && projectSlug == "" {
			projectSlug = ProjectFromSnapshot(snapshot).Slug
		}
	}

	saved := SavedState{}
	if projectSlug != "" {
		saved = s.LoadSavedState(projectSlug)
	}

	if !haveSnapshot && saved.Worktree != "" && pathExists(saved.Worktree) {
		snapshot = InspectRepository(ctx, saved.Worktree, s.now)
		haveSnapshot = true
	}

	if snapshot.IsRepo && projectSlug == "" {
		projectSlug = ProjectFromSnapshot(snapshot).Slug
	}

	stateSavedPath := ""
	if req.SaveState && snapshot.IsRepo {
		path, err := s.SaveGitState(snapshot)
		if err != nil {
			return ResumeResult{}, err
		}
		stateSavedPath = path
		if projectSlug != "" {
			saved = s.LoadSavedState(projectSlug)
		}
	}

	openLoops := s.LoadRecentHeadings(projectSlug, "open-loops.md", 5)
	decisions := s.LoadRecentHeadings(projectSlug, "decisions.md", 3)
	solvedProblems := s.LoadRecentHeadings(projectSlug, "solved-problems.md", 3)
	activities := s.LoadRecentActivities(projectSlug, 5)

	output := renderResume(projectSlug, req.Project, snapshot, saved, openLoops, decisions, solvedProblems, activities, haveSnapshot, stateSavedPath)
	return ResumeResult{Output: output, ProjectSlug: projectSlug, StateSavedPath: stateSavedPath, Snapshot: snapshot}, nil
}

func (s *Store) LoadSavedState(projectSlug string) SavedState {
	if projectSlug == "" {
		return SavedState{}
	}
	path := s.ProjectPaths(projectSlug).State
	content, err := os.ReadFile(path)
	if err != nil {
		return SavedState{}
	}
	state := ParseSavedState(string(content))
	state.Exists = true
	state.Path = path
	return state
}

func ParseSavedState(content string) SavedState {
	state := SavedState{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "- ") || !strings.Contains(line, ":") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		parts := strings.SplitN(line, ":", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		switch key {
		case "last updated":
			state.LastUpdated = value
		case "repository":
			state.Repository = value
		case "remote":
			state.Remote = value
		case "worktree":
			state.Worktree = value
		case "branch":
			state.Branch = value
		case "head":
			state.Head = value
		case "dirty files":
			state.DirtyFiles = value
		}
	}
	return state
}

func (s *Store) LoadRecentHeadings(projectSlug string, fileName string, limit int) []string {
	if projectSlug == "" || limit <= 0 {
		return nil
	}
	path := filepath.Join(s.ProjectPaths(projectSlug).Dir, fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var headings []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, strings.TrimSpace(strings.TrimPrefix(line, "## ")))
		}
	}

	if len(headings) <= limit {
		return headings
	}
	return headings[len(headings)-limit:]
}

func (s *Store) LoadRecentActivities(projectSlug string, limit int) []ActivityEvent {
	if projectSlug == "" || limit <= 0 {
		return nil
	}
	path := s.ProjectPaths(projectSlug).Activity
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var events []ActivityEvent
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event ActivityEvent
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			events = append(events, event)
		}
	}

	if len(events) <= limit {
		return events
	}
	return events[len(events)-limit:]
}

func CompareSavedState(saved SavedState, snapshot GitSnapshot) []string {
	var changes []string

	if !saved.Exists {
		return []string{"No saved project state was found."}
	}

	if saved.Worktree != "" && saved.Worktree != "unknown" && !pathExists(saved.Worktree) {
		changes = append(changes, fmt.Sprintf("Saved worktree no longer exists: %s", saved.Worktree))
	}

	if !snapshot.IsRepo {
		if snapshot.Available {
			changes = append(changes, "Current path is not a Git repository, so branch and commit freshness could not be verified.")
		} else {
			changes = append(changes, "Current Git state is unavailable.")
		}
		if len(changes) == 0 {
			changes = append(changes, "Saved state exists, but no current Git state was available for comparison.")
		}
		return changes
	}

	if saved.Branch != "" && saved.Branch != "unknown" && saved.Branch != snapshot.Branch {
		changes = append(changes, fmt.Sprintf("Branch changed from %s to %s.", saved.Branch, snapshot.Branch))
	}
	if saved.Head != "" && saved.Head != "unknown" && saved.Head != snapshot.Head {
		changes = append(changes, fmt.Sprintf("Head changed from %s to %s.", saved.Head, snapshot.Head))
	}
	if saved.Worktree != "" && saved.Worktree != "unknown" && filepath.Clean(saved.Worktree) != filepath.Clean(snapshot.Worktree) {
		changes = append(changes, fmt.Sprintf("Worktree changed from %s to %s.", saved.Worktree, snapshot.Worktree))
	}
	if len(changes) == 0 {
		changes = append(changes, "Saved branch, head, and worktree still match the current Git context.")
	}
	return changes
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" || path == "unknown" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func renderResume(projectSlug string, requestedProject string, snapshot GitSnapshot, saved SavedState, openLoops []string, decisions []string, solvedProblems []string, activities []ActivityEvent, haveSnapshot bool, stateSavedPath string) string {
	label := projectSlug
	if label == "" {
		label = strings.TrimSpace(requestedProject)
	}
	if label == "" && snapshot.RepoName != "" {
		label = snapshot.RepoName
	}
	if label == "" {
		label = "current work"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Goal\n")
	fmt.Fprintf(&b, "- Inference: Resume %s using saved leftoff records and current local evidence.\n", label)
	if stateSavedPath != "" {
		fmt.Fprintf(&b, "- Evidence: Updated compact Git state at %s.\n", stateSavedPath)
	}

	fmt.Fprintf(&b, "\nCurrent state\n")
	if haveSnapshot {
		if snapshot.IsRepo {
			fmt.Fprintf(&b, "- Verified Git repository: %s\n", valueOr(snapshot.RepoName, "unknown"))
			fmt.Fprintf(&b, "- Verified branch: %s\n", valueOr(snapshot.Branch, "unknown"))
			fmt.Fprintf(&b, "- Verified head: %s\n", valueOr(snapshot.Head, "unknown"))
			fmt.Fprintf(&b, "- Verified changed paths: %d\n", len(snapshot.ChangedFiles))
			if len(snapshot.ChangedFiles) > 0 {
				for _, changed := range firstChanged(snapshot.ChangedFiles, 8) {
					fmt.Fprintf(&b, "  - %s %s\n", valueOr(changed.Status, "?"), changed.Path)
				}
				if len(snapshot.ChangedFiles) > 8 {
					fmt.Fprintf(&b, "  - ...and %d more path(s)\n", len(snapshot.ChangedFiles)-8)
				}
			}
			if len(snapshot.RecentCommits) == 0 {
				fmt.Fprintf(&b, "- Recent commits: none recorded\n")
			} else {
				fmt.Fprintf(&b, "- Recent commits:\n")
				for _, commit := range firstCommits(snapshot.RecentCommits, 5) {
					fmt.Fprintf(&b, "  - %s %s\n", commit.Hash, commit.Summary)
				}
			}
		} else if snapshot.Available {
			fmt.Fprintf(&b, "- Verified: current path is not a Git repository.\n")
		} else {
			fmt.Fprintf(&b, "- Unverified: Git context is unavailable.\n")
		}
	} else {
		fmt.Fprintf(&b, "- Unverified: no repository path was inspected.\n")
	}

	if saved.Exists {
		fmt.Fprintf(&b, "- Saved state: last updated %s at %s\n", valueOr(saved.LastUpdated, "unknown"), saved.Path)
	} else {
		fmt.Fprintf(&b, "- Saved state: none found\n")
	}

	fmt.Fprintf(&b, "\nWhat changed since the last session\n")
	for _, change := range CompareSavedState(saved, snapshot) {
		fmt.Fprintf(&b, "- %s\n", change)
	}

	fmt.Fprintf(&b, "\nWhat is verified\n")
	if snapshot.IsRepo {
		fmt.Fprintf(&b, "- Local Git metadata was read using safe commands only.\n")
		fmt.Fprintf(&b, "- Changed-path reporting is file-path metadata only; file contents and full diffs were not read.\n")
	}
	if len(openLoops) > 0 {
		fmt.Fprintf(&b, "- Recent open loops:\n")
		for _, item := range openLoops {
			fmt.Fprintf(&b, "  - %s\n", item)
		}
	}
	if len(decisions) > 0 {
		fmt.Fprintf(&b, "- Recent decisions:\n")
		for _, item := range decisions {
			fmt.Fprintf(&b, "  - %s\n", item)
		}
	}
	if len(solvedProblems) > 0 {
		fmt.Fprintf(&b, "- Recent solved-problem records:\n")
		for _, item := range solvedProblems {
			fmt.Fprintf(&b, "  - %s\n", item)
		}
	}
	if len(activities) > 0 {
		fmt.Fprintf(&b, "- Recent leftoff activity:\n")
		for _, event := range activities {
			fmt.Fprintf(&b, "  - %s %s: %s\n", event.Timestamp, valueOr(event.Kind, "activity"), event.Summary)
		}
	}
	if !snapshot.IsRepo && len(openLoops) == 0 && len(decisions) == 0 && len(solvedProblems) == 0 && len(activities) == 0 {
		fmt.Fprintf(&b, "- No project-specific records were found.\n")
	}

	fmt.Fprintf(&b, "\nWhat remains uncertain\n")
	if !snapshot.IsRepo {
		fmt.Fprintf(&b, "- Current branch, commit, changed paths, and recent commits are unknown.\n")
	}
	if len(openLoops) == 0 {
		fmt.Fprintf(&b, "- No active open loop is recorded for this project.\n")
	}
	fmt.Fprintf(&b, "- Remote PRs, CI status, issues, and calendars were not queried.\n")
	fmt.Fprintf(&b, "- Effort estimates are not inferred in resume mode.\n")

	fmt.Fprintf(&b, "\nRecommended next action\n")
	switch {
	case snapshot.IsRepo && len(snapshot.ChangedFiles) > 0:
		fmt.Fprintf(&b, "- Review the changed-path summary, then connect the current worktree to the most relevant open loop before editing further.\n")
	case len(openLoops) > 0:
		fmt.Fprintf(&b, "- Open the most recent active open loop and decide the next concrete command or file to inspect.\n")
	default:
		fmt.Fprintf(&b, "- Capture the current goal or run a read-only scan so future resume output has stronger evidence.\n")
	}

	fmt.Fprintf(&b, "\nSafe commands to run\n")
	if snapshot.IsRepo {
		fmt.Fprintf(&b, "- `git status --short`\n")
		fmt.Fprintf(&b, "- `git diff --stat`\n")
		fmt.Fprintf(&b, "- `git log --oneline -n 5`\n")
	} else {
		fmt.Fprintf(&b, "- `leftoff capture \"task: <next step>\"`\n")
		fmt.Fprintf(&b, "- `leftoff scan --repo <path>`\n")
	}

	return b.String()
}

func firstChanged(files []ChangedFile, limit int) []ChangedFile {
	if len(files) <= limit {
		return files
	}
	return files[:limit]
}

func firstCommits(commits []Commit, limit int) []Commit {
	if len(commits) <= limit {
		return commits
	}
	return commits[:limit]
}
