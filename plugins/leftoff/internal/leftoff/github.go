package leftoff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type GitHubRequest struct {
	Project       string `json:"project,omitempty"`
	RepoPath      string `json:"repo_path,omitempty"`
	Refresh       bool   `json:"refresh,omitempty"`
	ForgetCache   bool   `json:"forget_cache,omitempty"`
	RetentionDays int    `json:"retention_days,omitempty"`
}

type GitHubResult struct {
	Output string      `json:"output"`
	Cache  GitHubCache `json:"cache"`
	Path   string      `json:"path,omitempty"`
}

type GitHubCache struct {
	FetchedAt     string               `json:"fetched_at"`
	Repository    string               `json:"repository"`
	RetentionDays int                  `json:"retention_days"`
	Commands      []string             `json:"commands"`
	PullRequests  []GitHubPRSummary    `json:"pull_requests"`
	Issues        []GitHubIssueSummary `json:"issues"`
	WorkflowRuns  []GitHubRunSummary   `json:"workflow_runs"`
	HealthNotes   []string             `json:"health_notes,omitempty"`
}

type GitHubPRSummary struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	IsDraft        bool   `json:"is_draft"`
	ReviewDecision string `json:"review_decision"`
	UpdatedAt      string `json:"updated_at"`
	HeadRefName    string `json:"head_ref_name"`
}

type GitHubIssueSummary struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	UpdatedAt string   `json:"updated_at"`
	Labels    []string `json:"labels,omitempty"`
}

type GitHubRunSummary struct {
	DatabaseID int64  `json:"database_id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Name       string `json:"name"`
	HeadBranch string `json:"head_branch"`
	UpdatedAt  string `json:"updated_at"`
}

func (s *Store) GitHub(ctx context.Context, req GitHubRequest) (GitHubResult, error) {
	if err := s.Init(); err != nil {
		return GitHubResult{}, err
	}
	if req.RetentionDays <= 0 {
		req.RetentionDays = 14
	}

	project := Slugify(req.Project)
	repository := ""
	if strings.TrimSpace(req.RepoPath) != "" {
		snapshot := InspectRepository(ctx, req.RepoPath, s.now)
		if snapshot.IsRepo {
			repository = githubRepoFromRemote(snapshot.Remote)
			if project == "" {
				project = ProjectFromSnapshot(snapshot).Slug
			}
		}
	}
	if project == "" {
		project = "github"
	}

	path := s.githubCachePath(project)
	if req.ForgetCache {
		forgot, err := s.ForgetGitHubCache(project)
		if err != nil {
			return GitHubResult{}, err
		}
		output := "GITHUB\n- Forgot GitHub cache: " + forgot + "\n"
		return GitHubResult{Output: output, Path: forgot}, nil
	}

	if !req.Refresh {
		cache, stale, err := s.LoadGitHubCache(project, req.RetentionDays)
		if err != nil {
			if os.IsNotExist(err) {
				output := "GITHUB\n- No GitHub cache found.\n- No remote query was run. Use `--refresh` to opt in to read-only gh queries.\n"
				return GitHubResult{Output: output, Path: path}, nil
			}
			return GitHubResult{}, err
		}
		output := renderGitHubCache(cache, path, stale, false)
		return GitHubResult{Output: output, Cache: cache, Path: path}, nil
	}

	cache := GitHubCache{
		FetchedAt:     s.now().Format(timeFormatRFC3339()),
		Repository:    repository,
		RetentionDays: req.RetentionDays,
	}
	if repository == "" {
		cache.HealthNotes = append(cache.HealthNotes, "Could not derive GitHub owner/repo from local remote; gh will use its current repository context.")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		cache.HealthNotes = append(cache.HealthNotes, "GitHub CLI `gh` is not available on PATH.")
		output := renderGitHubCache(cache, path, false, true)
		return GitHubResult{Output: output, Cache: cache, Path: path}, nil
	}

	if err := populateGitHubCache(ctx, &cache, repository); err != nil {
		cache.HealthNotes = append(cache.HealthNotes, err.Error())
		output := renderGitHubCache(cache, path, false, true)
		return GitHubResult{Output: output, Cache: cache, Path: path}, nil
	}
	if err := s.SaveGitHubCache(project, cache); err != nil {
		return GitHubResult{}, err
	}
	output := renderGitHubCache(cache, path, false, true)
	return GitHubResult{Output: output, Cache: cache, Path: path}, nil
}

func (s *Store) githubCachePath(project string) string {
	return filepath.Join(s.Root, "cache", "github", Slugify(project)+".json")
}

func (s *Store) SaveGitHubCache(project string, cache GitHubCache) error {
	cache = sanitizeGitHubCache(cache)
	path := s.githubCachePath(project)
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWriteFile(path, data, 0o600)
}

func (s *Store) LoadGitHubCache(project string, retentionDays int) (GitHubCache, bool, error) {
	path := s.githubCachePath(project)
	content, err := os.ReadFile(path)
	if err != nil {
		return GitHubCache{}, false, err
	}
	var cache GitHubCache
	if err := json.Unmarshal(content, &cache); err != nil {
		return GitHubCache{}, false, err
	}
	stale := false
	if fetched, err := time.Parse(timeFormatRFC3339(), cache.FetchedAt); err == nil {
		stale = s.now().Sub(fetched) > time.Duration(retentionDays)*24*time.Hour
	} else {
		stale = true
	}
	return cache, stale, nil
}

func (s *Store) ForgetGitHubCache(project string) (string, error) {
	path := s.githubCachePath(project)
	if err := s.requireInsideRoot(path); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", err
	}
	backup, err := s.BackupFile(path, "forget GitHub integration cache")
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return backup, nil
}

func populateGitHubCache(ctx context.Context, cache *GitHubCache, repository string) error {
	prJSON, command, err := runGHJSON(ctx, repository, "pr", "list", "--state", "open", "--limit", "20", "--json", "number,title,state,isDraft,reviewDecision,updatedAt,headRefName")
	cache.Commands = append(cache.Commands, command)
	if err != nil {
		return err
	}
	cache.PullRequests = parseGitHubPRs(prJSON)

	issueJSON, command, err := runGHJSON(ctx, repository, "issue", "list", "--state", "open", "--limit", "20", "--json", "number,title,state,updatedAt,labels")
	cache.Commands = append(cache.Commands, command)
	if err != nil {
		return err
	}
	cache.Issues = parseGitHubIssues(issueJSON)

	runJSON, command, err := runGHJSON(ctx, repository, "run", "list", "--limit", "20", "--json", "databaseId,status,conclusion,name,headBranch,updatedAt")
	cache.Commands = append(cache.Commands, command)
	if err != nil {
		return err
	}
	cache.WorkflowRuns = parseGitHubRuns(runJSON)
	return nil
}

func runGHJSON(ctx context.Context, repository string, args ...string) ([]byte, string, error) {
	cmdArgs := append([]string{}, args...)
	if repository != "" {
		cmdArgs = append(cmdArgs, "--repo", repository)
	}
	command := "gh " + strings.Join(cmdArgs, " ")
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "gh", cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, command, fmt.Errorf("read-only GitHub query failed for `%s`", command)
	}
	return output, command, nil
}

func parseGitHubPRs(data []byte) []GitHubPRSummary {
	var raw []struct {
		Number         int    `json:"number"`
		Title          string `json:"title"`
		State          string `json:"state"`
		IsDraft        bool   `json:"isDraft"`
		ReviewDecision string `json:"reviewDecision"`
		UpdatedAt      string `json:"updatedAt"`
		HeadRefName    string `json:"headRefName"`
	}
	_ = json.Unmarshal(data, &raw)
	var out []GitHubPRSummary
	for _, item := range raw {
		out = append(out, GitHubPRSummary{
			Number:         item.Number,
			Title:          sanitizeMetadataTitle(item.Title),
			State:          item.State,
			IsDraft:        item.IsDraft,
			ReviewDecision: item.ReviewDecision,
			UpdatedAt:      item.UpdatedAt,
			HeadRefName:    sanitizeMetadataBranch(item.HeadRefName),
		})
	}
	return out
}

func parseGitHubIssues(data []byte) []GitHubIssueSummary {
	var raw []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		UpdatedAt string `json:"updatedAt"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	_ = json.Unmarshal(data, &raw)
	var out []GitHubIssueSummary
	for _, item := range raw {
		var labels []string
		for _, label := range item.Labels {
			labels = append(labels, cleanSummary(sanitizeExternalMetadata(label.Name), 60))
		}
		sort.Strings(labels)
		out = append(out, GitHubIssueSummary{
			Number:    item.Number,
			Title:     sanitizeMetadataTitle(item.Title),
			State:     item.State,
			UpdatedAt: item.UpdatedAt,
			Labels:    labels,
		})
	}
	return out
}

func parseGitHubRuns(data []byte) []GitHubRunSummary {
	var raw []struct {
		DatabaseID int64  `json:"databaseId"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		Name       string `json:"name"`
		HeadBranch string `json:"headBranch"`
		UpdatedAt  string `json:"updatedAt"`
	}
	_ = json.Unmarshal(data, &raw)
	var out []GitHubRunSummary
	for _, item := range raw {
		out = append(out, GitHubRunSummary{
			DatabaseID: item.DatabaseID,
			Status:     item.Status,
			Conclusion: item.Conclusion,
			Name:       sanitizeMetadataTitle(item.Name),
			HeadBranch: sanitizeMetadataBranch(item.HeadBranch),
			UpdatedAt:  item.UpdatedAt,
		})
	}
	return out
}

func githubRepoFromRemote(remote string) string {
	remote = strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	if remote == "" {
		return ""
	}
	if strings.Contains(remote, "github.com/") {
		parts := strings.Split(remote, "github.com/")
		if len(parts) == 2 {
			return trimGitHubRepo(parts[1])
		}
	}
	if strings.Contains(remote, "github.com:") {
		parts := strings.Split(remote, "github.com:")
		if len(parts) == 2 {
			return trimGitHubRepo(parts[1])
		}
	}
	return ""
}

func trimGitHubRepo(value string) string {
	value = strings.Trim(value, "/")
	parts := strings.Split(value, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")
}

func renderGitHubCache(cache GitHubCache, path string, stale bool, refreshed bool) string {
	cache = sanitizeGitHubCache(cache)

	var b strings.Builder
	fmt.Fprintf(&b, "GITHUB\n")
	if refreshed {
		fmt.Fprintf(&b, "- Remote queries were opt-in for this command.\n")
	} else {
		fmt.Fprintf(&b, "- No remote query was run; showing local cache only.\n")
	}
	fmt.Fprintf(&b, "- Cache path: %s\n", path)
	fmt.Fprintf(&b, "- Repository: %s\n", valueOr(cache.Repository, "unknown"))
	fmt.Fprintf(&b, "- Fetched at: %s\n", valueOr(cache.FetchedAt, "unknown"))
	fmt.Fprintf(&b, "- Retention: %d day(s)\n", cache.RetentionDays)
	if stale {
		fmt.Fprintf(&b, "- Cache status: stale\n")
	}
	if len(cache.HealthNotes) > 0 {
		fmt.Fprintf(&b, "\nHEALTH\n")
		for _, note := range cache.HealthNotes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}
	fmt.Fprintf(&b, "\nPULL REQUESTS\n")
	if len(cache.PullRequests) == 0 {
		fmt.Fprintf(&b, "- No cached open PR metadata.\n")
	} else {
		for _, pr := range firstPRs(cache.PullRequests, 10) {
			fmt.Fprintf(&b, "- #%d %s [%s review:%s]\n", pr.Number, pr.Title, valueOr(pr.State, "unknown"), valueOr(pr.ReviewDecision, "unknown"))
		}
	}
	fmt.Fprintf(&b, "\nISSUES\n")
	if len(cache.Issues) == 0 {
		fmt.Fprintf(&b, "- No cached open issue metadata.\n")
	} else {
		for _, issue := range firstIssues(cache.Issues, 10) {
			fmt.Fprintf(&b, "- #%d %s [%s]\n", issue.Number, issue.Title, valueOr(issue.State, "unknown"))
		}
	}
	fmt.Fprintf(&b, "\nWORKFLOW RUNS\n")
	if len(cache.WorkflowRuns) == 0 {
		fmt.Fprintf(&b, "- No cached workflow run metadata.\n")
	} else {
		for _, run := range firstRuns(cache.WorkflowRuns, 10) {
			fmt.Fprintf(&b, "- %s on %s [%s/%s]\n", run.Name, valueOr(run.HeadBranch, "unknown"), valueOr(run.Status, "unknown"), valueOr(run.Conclusion, "unknown"))
		}
	}
	fmt.Fprintf(&b, "\nQUERIES\n")
	if len(cache.Commands) == 0 {
		fmt.Fprintf(&b, "- none\n")
	} else {
		for _, command := range cache.Commands {
			fmt.Fprintf(&b, "- `%s`\n", command)
		}
	}
	fmt.Fprintf(&b, "\nPRIVACY\n")
	fmt.Fprintf(&b, "- Cached fields are titles, numbers, states, labels, review decisions, branches, timestamps, and workflow status only.\n")
	fmt.Fprintf(&b, "- Full PR, issue, review, log, and artifact bodies are not stored.\n")
	return b.String()
}

func sanitizeGitHubCache(cache GitHubCache) GitHubCache {
	cache.Repository = sanitizeExternalMetadata(cache.Repository)
	for i := range cache.Commands {
		cache.Commands[i] = sanitizeExternalMetadata(cache.Commands[i])
	}
	for i := range cache.HealthNotes {
		cache.HealthNotes[i] = sanitizeExternalMetadata(cache.HealthNotes[i])
	}
	for i := range cache.PullRequests {
		cache.PullRequests[i].Title = sanitizeMetadataTitle(cache.PullRequests[i].Title)
		cache.PullRequests[i].HeadRefName = sanitizeMetadataBranch(cache.PullRequests[i].HeadRefName)
	}
	for i := range cache.Issues {
		cache.Issues[i].Title = sanitizeMetadataTitle(cache.Issues[i].Title)
		for j := range cache.Issues[i].Labels {
			cache.Issues[i].Labels[j] = sanitizeExternalMetadata(cache.Issues[i].Labels[j])
		}
	}
	for i := range cache.WorkflowRuns {
		cache.WorkflowRuns[i].Name = sanitizeMetadataTitle(cache.WorkflowRuns[i].Name)
		cache.WorkflowRuns[i].HeadBranch = sanitizeMetadataBranch(cache.WorkflowRuns[i].HeadBranch)
	}
	return cache
}

func firstPRs(values []GitHubPRSummary, limit int) []GitHubPRSummary {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func firstIssues(values []GitHubIssueSummary, limit int) []GitHubIssueSummary {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func firstRuns(values []GitHubRunSummary, limit int) []GitHubRunSummary {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
