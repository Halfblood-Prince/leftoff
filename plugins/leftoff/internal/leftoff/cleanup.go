package leftoff

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CleanUpRequest struct {
	Project  string
	RepoPath string
	Action   string
	Apply    bool
	Confirm  bool
}

type CleanUpResult struct {
	Output   string
	Findings []CleanupFinding
	Applied  []string
}

type CleanupFinding struct {
	ID             string
	Category       string
	Title          string
	Reason         string
	Evidence       []string
	Risk           string
	ReversalPath   string
	CommandPreview string
	CanApply       bool
}

type BranchInfo struct {
	Name       string
	Date       time.Time
	Upstream   string
	Hash       string
	Subject    string
	TrackShort string
	Current    bool
	Protected  bool
	Unpushed   bool
	Stale      bool
}

type WorktreeCleanupInfo struct {
	Path        string
	Branch      string
	Head        string
	Exists      bool
	Dirty       bool
	Protected   bool
	Current     bool
	Stale       bool
	LastTouched time.Time
}

type GitCleanupSnapshot struct {
	Available   bool
	IsRepo      bool
	Root        string
	Current     string
	Branches    []BranchInfo
	Worktrees   []WorktreeCleanupInfo
	HealthNotes []string
	Commands    []string
}

func (s *Store) CleanUp(ctx context.Context, req CleanUpRequest) (CleanUpResult, error) {
	if err := s.Init(); err != nil {
		return CleanUpResult{}, err
	}

	records, err := s.LoadRecords(RecordQuery{Project: req.Project, IncludeInbox: req.Project == ""})
	if err != nil {
		return CleanUpResult{}, err
	}

	var findings []CleanupFinding
	findings = append(findings, cleanupRecordFindings(records, s.now())...)
	findings = append(findings, cleanupProjectFindings(s, req.Project)...)

	if strings.TrimSpace(req.RepoPath) != "" {
		gitSnapshot := InspectGitForCleanup(ctx, req.RepoPath, s.now)
		findings = append(findings, cleanupGitFindings(gitSnapshot, s.now())...)
		if !gitSnapshot.IsRepo && len(gitSnapshot.HealthNotes) > 0 {
			findings = append(findings, CleanupFinding{
				ID:       "git-context-unavailable",
				Category: "git",
				Title:    "Git cleanup context unavailable",
				Reason:   "Read-only Git cleanup inspection could not establish a repository context.",
				Evidence: gitSnapshot.HealthNotes,
				Risk:     "info",
			})
		}
	}

	sort.SliceStable(findings, func(i int, j int) bool {
		if riskRank(findings[i].Risk) != riskRank(findings[j].Risk) {
			return riskRank(findings[i].Risk) > riskRank(findings[j].Risk)
		}
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].Title < findings[j].Title
	})

	result := CleanUpResult{Findings: findings}
	if req.Apply {
		applied, err := s.applyCleanup(req)
		if err != nil {
			return CleanUpResult{}, err
		}
		result.Applied = applied
	}
	result.Output = renderCleanUp(result, req)
	return result, nil
}

func cleanupRecordFindings(records []MarkdownRecord, now time.Time) []CleanupFinding {
	var findings []CleanupFinding
	seen := map[string]MarkdownRecord{}

	for _, record := range records {
		status := NormalizeStatus(record.Status)
		if record.Project == "inbox" && isTaskLike(record) && status != string(StatusDone) && status != string(StatusAbandoned) {
			findings = append(findings, CleanupFinding{
				ID:           "unresolved-inbox-" + record.ID,
				Category:     "records",
				Title:        "Unresolved inbox item: " + record.Title,
				Reason:       "Inbox records have no confirmed project owner.",
				Evidence:     []string{record.ID, "Project: inbox", "Status: " + valueOr(record.Status, "unknown")},
				Risk:         "low",
				ReversalPath: "No mutation is proposed; capture a project link or leave the record as-is.",
			})
		}

		if record.Type == RecordDecision {
			revisit := record.Field("revisit when")
			if date := extractISODate(revisit); !date.IsZero() && !date.After(startOfDay(now)) {
				findings = append(findings, CleanupFinding{
					ID:           "expired-decision-" + record.ID,
					Category:     "records",
					Title:        "Decision revisit condition is due: " + record.Title,
					Reason:       "The decision has a revisit date that has passed.",
					Evidence:     []string{record.ID, "Revisit when: " + revisit},
					Risk:         "low",
					ReversalPath: "Review only; no record changes are made by default.",
				})
			}
			if record.Field("decision") == "" || record.Field("evidence") == "" {
				findings = append(findings, CleanupFinding{
					ID:           "thin-decision-" + record.ID,
					Category:     "records",
					Title:        "Decision record lacks rationale or evidence: " + record.Title,
					Reason:       "Decisions are less useful without a durable decision and evidence field.",
					Evidence:     []string{record.ID},
					Risk:         "low",
					ReversalPath: "Add missing rationale manually; no automatic rewrite is proposed.",
				})
			}
		}

		if record.Type == RecordIdea && status == string(StatusParked) {
			touched := record.EffectiveDate()
			if !touched.IsZero() && startOfDay(now).Sub(startOfDay(touched)) >= 60*24*time.Hour {
				findings = append(findings, CleanupFinding{
					ID:           "dead-experiment-" + record.ID,
					Category:     "records",
					Title:        "Parked idea may be a dead experiment: " + record.Title,
					Reason:       "The idea has been parked for at least 60 days.",
					Evidence:     []string{record.ID, "Last touched: " + touched.Format("2006-01-02")},
					Risk:         "low",
					ReversalPath: "Leave parked, add a next action, or mark abandoned manually.",
				})
			}
		}

		key := duplicateRecordKey(record)
		if previous, ok := seen[key]; ok && key != "" {
			findings = append(findings, CleanupFinding{
				ID:           "duplicate-record-" + record.ID,
				Category:     "records",
				Title:        "Possible duplicate record: " + record.Title,
				Reason:       "Two records have the same type, project, and normalized primary text.",
				Evidence:     []string{previous.ID, record.ID},
				Risk:         "medium",
				ReversalPath: "Compare records manually; no automatic merge is performed.",
			})
		} else if key != "" {
			seen[key] = record
		}
	}

	return findings
}

func duplicateRecordKey(record MarkdownRecord) string {
	text := strings.Join(sortedTokens(record.PrimaryText()), " ")
	if text == "" {
		return ""
	}
	return string(record.Type) + "|" + record.Project + "|" + text
}

func cleanupProjectFindings(s *Store, projectFilter string) []CleanupFinding {
	slugs, err := s.ProjectSlugs(projectFilter)
	if err != nil {
		return nil
	}
	var findings []CleanupFinding
	for _, slug := range slugs {
		paths := s.ProjectPaths(slug)
		count := 0
		for _, path := range []string{paths.OpenLoops, paths.Decisions, paths.SolvedProblems, paths.Releases, paths.Friction} {
			records, err := s.loadRecordsFromFile(path, slug)
			if err == nil {
				count += len(records)
			}
		}
		if count == 0 {
			findings = append(findings, CleanupFinding{
				ID:           "dangling-project-" + slug,
				Category:     "projects",
				Title:        "Project record has no captured work: " + slug,
				Reason:       "Project directory exists but contains no Markdown records.",
				Evidence:     []string{"Project: " + slug},
				Risk:         "low",
				ReversalPath: "Keep the project, add context, or remove the project directory manually after backup.",
			})
		}
	}
	return findings
}

func InspectGitForCleanup(ctx context.Context, repoPath string, clock func() time.Time) GitCleanupSnapshot {
	base := InspectRepository(ctx, repoPath, clock)
	snapshot := GitCleanupSnapshot{
		Available:   base.Available,
		IsRepo:      base.IsRepo,
		Root:        base.Root,
		Current:     base.Branch,
		HealthNotes: append([]string{}, base.HealthNotes...),
		Commands:    append([]string{}, base.Commands...),
	}
	if !base.IsRepo {
		return snapshot
	}

	if output, ok := runGitText(ctx, base.Root, &base, "for-each-ref", "--format=%(refname:short)|%(committerdate:short)|%(upstream:short)|%(objectname:short)|%(subject)|%(upstream:trackshort)", "refs/heads"); ok {
		snapshot.Commands = append(snapshot.Commands, base.Commands[len(snapshot.Commands):]...)
		snapshot.Branches = parseBranchCleanupInfo(output, base.Branch, clock)
	}
	for _, worktree := range base.Worktrees {
		info := WorktreeCleanupInfo{
			Path:      worktree.Path,
			Branch:    worktree.Branch,
			Head:      worktree.Head,
			Exists:    pathExists(worktree.Path),
			Protected: IsProtectedBranch(worktree.Branch),
			Current:   filepath.Clean(worktree.Path) == filepath.Clean(base.Worktree),
		}
		if info.Exists {
			if output, ok := runGitText(ctx, worktree.Path, &base, "status", "--short"); ok {
				info.Dirty = strings.TrimSpace(output) != ""
			} else {
				info.Dirty = true
			}
		}
		for _, branch := range snapshot.Branches {
			if branch.Name == info.Branch {
				info.LastTouched = branch.Date
				info.Stale = branch.Stale
				break
			}
		}
		snapshot.Worktrees = append(snapshot.Worktrees, info)
	}
	snapshot.Commands = uniqueStrings(append(snapshot.Commands, base.Commands...))
	return snapshot
}

func parseBranchCleanupInfo(output string, current string, clock func() time.Time) []BranchInfo {
	now := time.Now
	if clock != nil {
		now = clock
	}
	var branches []BranchInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		for len(parts) < 6 {
			parts = append(parts, "")
		}
		date := parseDateField(parts[1])
		info := BranchInfo{
			Name:       sanitizeMetadataBranch(parts[0]),
			Date:       date,
			Upstream:   sanitizeMetadataBranch(parts[2]),
			Hash:       shortHash(strings.TrimSpace(parts[3])),
			Subject:    sanitizeMetadataTitle(parts[4]),
			TrackShort: sanitizeMetadataTitle(parts[5]),
		}
		info.Current = info.Name == current
		info.Protected = IsProtectedBranch(info.Name)
		info.Unpushed = info.Upstream == "" || strings.Contains(info.TrackShort, ">")
		info.Stale = !date.IsZero() && startOfDay(now()).Sub(startOfDay(date)) >= 45*24*time.Hour
		branches = append(branches, info)
	}
	return branches
}

func shortHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func IsProtectedBranch(name string) bool {
	name = strings.TrimPrefix(strings.TrimSpace(name), "refs/heads/")
	switch name {
	case "main", "master", "trunk", "develop", "development", "dev", "release", "stable":
		return true
	default:
		return strings.HasPrefix(name, "release/") || strings.HasPrefix(name, "hotfix/")
	}
}

func cleanupGitFindings(snapshot GitCleanupSnapshot, now time.Time) []CleanupFinding {
	if !snapshot.IsRepo {
		return nil
	}
	var findings []CleanupFinding
	for _, branch := range snapshot.Branches {
		findings = append(findings, analyzeBranchForCleanup(branch, now)...)
	}
	for _, worktree := range snapshot.Worktrees {
		findings = append(findings, analyzeWorktreeForCleanup(worktree)...)
	}
	return findings
}

func analyzeBranchForCleanup(branch BranchInfo, now time.Time) []CleanupFinding {
	if branch.Current {
		return nil
	}
	if branch.Protected {
		return []CleanupFinding{{
			ID:           "protected-branch-" + Slugify(branch.Name),
			Category:     "git",
			Title:        "Protected branch retained: " + branch.Name,
			Reason:       "Protected branches are never suggested for deletion.",
			Evidence:     []string{"Branch: " + branch.Name},
			Risk:         "info",
			ReversalPath: "No action proposed.",
		}}
	}
	if branch.Unpushed {
		return []CleanupFinding{{
			ID:           "unpushed-branch-" + Slugify(branch.Name),
			Category:     "git",
			Title:        "Branch may contain unpushed work: " + branch.Name,
			Reason:       "The branch has no upstream or is ahead of upstream, so deletion would be risky.",
			Evidence:     []string{"Branch: " + branch.Name, "Upstream: " + valueOr(branch.Upstream, "none"), "Track: " + valueOr(branch.TrackShort, "unknown")},
			Risk:         "high",
			ReversalPath: "Push, back up, or inspect commits before considering cleanup.",
		}}
	}
	if branch.Stale {
		date := "unknown"
		if !branch.Date.IsZero() {
			date = branch.Date.Format("2006-01-02")
		}
		return []CleanupFinding{{
			ID:             "stale-branch-" + Slugify(branch.Name),
			Category:       "git",
			Title:          "Stale branch candidate: " + branch.Name,
			Reason:         "Branch is older than 45 days, has an upstream, and is not protected.",
			Evidence:       []string{"Last commit date: " + date, "Head: " + valueOr(branch.Hash, "unknown"), "Subject: " + valueOr(branch.Subject, "unknown")},
			Risk:           "medium",
			ReversalPath:   "Restore from remote or reflog if needed; leftoff does not delete Git branches.",
			CommandPreview: "git branch -d " + branch.Name,
		}}
	}
	return nil
}

func analyzeWorktreeForCleanup(worktree WorktreeCleanupInfo) []CleanupFinding {
	if worktree.Current || worktree.Protected {
		return nil
	}
	if !worktree.Exists {
		return []CleanupFinding{{
			ID:             "missing-worktree-" + Slugify(worktree.Path),
			Category:       "git",
			Title:          "Worktree path no longer exists: " + worktree.Path,
			Reason:         "Git still reports a worktree path that is missing locally.",
			Evidence:       []string{"Branch: " + valueOr(worktree.Branch, "unknown"), "Head: " + valueOr(shortHash(worktree.Head), "unknown")},
			Risk:           "medium",
			ReversalPath:   "Run `git worktree list` and inspect before pruning; leftoff does not prune Git worktrees.",
			CommandPreview: "git worktree prune --dry-run",
		}}
	}
	if worktree.Dirty {
		return []CleanupFinding{{
			ID:           "dirty-worktree-" + Slugify(worktree.Path),
			Category:     "git",
			Title:        "Dirty worktree needs review: " + worktree.Path,
			Reason:       "The worktree has uncommitted changes and must not be cleaned automatically.",
			Evidence:     []string{"Branch: " + valueOr(worktree.Branch, "unknown")},
			Risk:         "high",
			ReversalPath: "Inspect `git -C <worktree> status --short`; no cleanup command is proposed.",
		}}
	}
	if worktree.Stale {
		return []CleanupFinding{{
			ID:             "stale-worktree-" + Slugify(worktree.Path),
			Category:       "git",
			Title:          "Clean stale worktree candidate: " + worktree.Path,
			Reason:         "The worktree is clean, not protected, and its branch appears stale.",
			Evidence:       []string{"Branch: " + valueOr(worktree.Branch, "unknown")},
			Risk:           "medium",
			ReversalPath:   "Confirm branch state manually; leftoff does not remove Git worktrees.",
			CommandPreview: "git worktree remove " + worktree.Path,
		}}
	}
	return nil
}

func (s *Store) applyCleanup(req CleanUpRequest) ([]string, error) {
	if !req.Confirm {
		return nil, fmt.Errorf("cleanup apply requires --confirm")
	}
	switch strings.TrimSpace(req.Action) {
	case "dedupe-activity":
		return s.dedupeActivityFiles(req.Project)
	case "":
		return nil, fmt.Errorf("cleanup apply requires --action")
	default:
		return nil, fmt.Errorf("unsupported cleanup action %q", req.Action)
	}
}

func (s *Store) dedupeActivityFiles(project string) ([]string, error) {
	slugs, err := s.ProjectSlugs(project)
	if err != nil {
		return nil, err
	}
	var applied []string
	for _, slug := range slugs {
		path := s.ProjectPaths(slug).Activity
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return applied, err
		}
		seen := map[string]bool{}
		var lines []string
		removed := 0
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if seen[line] {
				removed++
				continue
			}
			seen[line] = true
			lines = append(lines, line)
		}
		if removed == 0 {
			continue
		}
		backup, err := s.BackupFile(path, "cleanup apply: dedupe activity JSONL")
		if err != nil {
			return applied, err
		}
		repaired := strings.Join(lines, "\n")
		if repaired != "" {
			repaired += "\n"
		}
		if err := atomicWriteFile(path, []byte(repaired), 0o600); err != nil {
			return applied, err
		}
		applied = append(applied, fmt.Sprintf("Removed %d duplicate activity event(s) from %s; backup: %s", removed, path, backup))
	}
	return applied, nil
}

func renderCleanUp(result CleanUpResult, req CleanUpRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CLEAN-UP\n")
	if len(result.Applied) > 0 {
		fmt.Fprintf(&b, "\nAPPLIED\n")
		for _, applied := range result.Applied {
			fmt.Fprintf(&b, "- %s\n", applied)
		}
	}
	fmt.Fprintf(&b, "\nPLAN\n")
	if len(result.Findings) == 0 {
		fmt.Fprintf(&b, "- No cleanup opportunities found from local evidence.\n")
	} else {
		for _, finding := range result.Findings {
			fmt.Fprintf(&b, "- %s [%s risk]\n", finding.Title, finding.Risk)
			fmt.Fprintf(&b, "  Reason: %s\n", finding.Reason)
			if len(finding.Evidence) > 0 {
				fmt.Fprintf(&b, "  Evidence: %s\n", strings.Join(firstStrings(finding.Evidence, 3), "; "))
			}
			if finding.CommandPreview != "" {
				fmt.Fprintf(&b, "  Command preview: `%s`\n", finding.CommandPreview)
			}
			if finding.ReversalPath != "" {
				fmt.Fprintf(&b, "  Reversal path: %s\n", finding.ReversalPath)
			}
		}
	}
	fmt.Fprintf(&b, "\nSAFETY\n")
	fmt.Fprintf(&b, "- Default mode is report-only.\n")
	fmt.Fprintf(&b, "- Git branch and worktree deletion are never performed by leftoff.\n")
	fmt.Fprintf(&b, "- Low-risk record maintenance requires `--apply --confirm --action dedupe-activity`.\n")
	if req.Apply && !req.Confirm {
		fmt.Fprintf(&b, "- Requested apply was not run because confirmation was missing.\n")
	}
	return b.String()
}

func riskRank(risk string) int {
	switch strings.ToLower(risk) {
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}
