package leftoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceRepository struct {
	Name          string `json:"name"`
	ProjectSlug   string `json:"project_slug"`
	Path          string `json:"path"`
	AddedAt       string `json:"added_at"`
	LastScannedAt string `json:"last_scanned_at,omitempty"`
}

type WorkspaceRegistry struct {
	Version      int                   `json:"version"`
	Repositories []WorkspaceRepository `json:"repositories"`
}

type WorkspaceAddRequest struct {
	RepoPath string `json:"repo_path"`
}

type WorkspaceAddResult struct {
	Output       string              `json:"output"`
	RegistryPath string              `json:"registry_path"`
	Repository   WorkspaceRepository `json:"repository"`
	AlreadyAdded bool                `json:"already_added"`
}

type WorkspaceListResult struct {
	Output       string                `json:"output"`
	RegistryPath string                `json:"registry_path"`
	Repositories []WorkspaceRepository `json:"repositories"`
}

type WorkspaceScanRepository struct {
	Repository WorkspaceRepository `json:"repository"`
	Snapshot   GitSnapshot         `json:"snapshot"`
	StatePath  string              `json:"state_path,omitempty"`
}

type WorkspaceScanResult struct {
	Output       string                    `json:"output"`
	RegistryPath string                    `json:"registry_path"`
	CachePath    string                    `json:"cache_path"`
	Repositories []WorkspaceScanRepository `json:"repositories"`
	HealthNotes  []string                  `json:"health_notes,omitempty"`
}

func (s *Store) AddWorkspaceRepo(ctx context.Context, req WorkspaceAddRequest) (WorkspaceAddResult, error) {
	if err := s.Init(); err != nil {
		return WorkspaceAddResult{}, err
	}
	repoPath := strings.TrimSpace(req.RepoPath)
	if repoPath == "" {
		return WorkspaceAddResult{}, errors.New("workspace add requires a repository path")
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return WorkspaceAddResult{}, err
	}
	abs = filepath.Clean(abs)
	if sanitizeMetadataPath(abs) != abs {
		return WorkspaceAddResult{}, errors.New("repository path contains secret-like or oversized metadata; refusing to persist it")
	}

	snapshot := InspectRepository(ctx, abs, s.now)
	if !snapshot.IsRepo {
		return WorkspaceAddResult{}, fmt.Errorf("workspace add requires a Git repository: %s", strings.Join(snapshot.HealthNotes, "; "))
	}
	root := filepath.Clean(snapshot.Root)
	if sanitizeMetadataPath(root) != root {
		return WorkspaceAddResult{}, errors.New("repository root contains secret-like or oversized metadata; refusing to persist it")
	}

	registry, registryPath, err := s.loadWorkspaceRegistry()
	if err != nil {
		return WorkspaceAddResult{}, err
	}
	meta := ProjectFromSnapshot(snapshot)
	repo := WorkspaceRepository{
		Name:        meta.Name,
		ProjectSlug: meta.Slug,
		Path:        root,
		AddedAt:     s.now().Format(timeFormatRFC3339()),
	}

	alreadyAdded := false
	for i := range registry.Repositories {
		if samePath(registry.Repositories[i].Path, repo.Path) {
			alreadyAdded = true
			repo.AddedAt = registry.Repositories[i].AddedAt
			repo.LastScannedAt = registry.Repositories[i].LastScannedAt
			registry.Repositories[i] = repo
			break
		}
	}
	if !alreadyAdded {
		registry.Repositories = append(registry.Repositories, repo)
	}
	sortWorkspaceRepositories(registry.Repositories)
	if err := s.saveWorkspaceRegistry(registry); err != nil {
		return WorkspaceAddResult{}, err
	}

	result := WorkspaceAddResult{
		RegistryPath: registryPath,
		Repository:   repo,
		AlreadyAdded: alreadyAdded,
	}
	result.Output = renderWorkspaceAddResult(result)
	return result, nil
}

func (s *Store) ListWorkspace() (WorkspaceListResult, error) {
	if err := s.Init(); err != nil {
		return WorkspaceListResult{}, err
	}
	registry, registryPath, err := s.loadWorkspaceRegistry()
	if err != nil {
		return WorkspaceListResult{}, err
	}
	result := WorkspaceListResult{
		RegistryPath: registryPath,
		Repositories: registry.Repositories,
	}
	result.Output = renderWorkspaceListResult(result)
	return result, nil
}

func (s *Store) ScanWorkspace(ctx context.Context) (WorkspaceScanResult, error) {
	if err := s.Init(); err != nil {
		return WorkspaceScanResult{}, err
	}
	registry, registryPath, err := s.loadWorkspaceRegistry()
	if err != nil {
		return WorkspaceScanResult{}, err
	}
	result := WorkspaceScanResult{
		RegistryPath: registryPath,
		CachePath:    s.workspaceScanCachePath(),
	}
	for i := range registry.Repositories {
		repo := registry.Repositories[i]
		snapshot := InspectRepository(ctx, repo.Path, s.now)
		item := WorkspaceScanRepository{
			Repository: repo,
			Snapshot:   snapshot,
		}
		if snapshot.IsRepo {
			meta := ProjectFromSnapshot(snapshot)
			repo.Name = meta.Name
			repo.ProjectSlug = meta.Slug
			repo.LastScannedAt = s.now().Format(timeFormatRFC3339())
			item.Repository = repo
			statePath, err := s.SaveGitState(snapshot)
			if err != nil {
				note := fmt.Sprintf("%s: could not save Git state: %v", valueOr(repo.Name, repo.Path), err)
				result.HealthNotes = append(result.HealthNotes, sanitizeExternalMetadata(note))
			} else {
				item.StatePath = statePath
			}
		} else {
			note := fmt.Sprintf("%s: %s", valueOr(repo.Name, repo.Path), strings.Join(snapshot.HealthNotes, "; "))
			result.HealthNotes = append(result.HealthNotes, sanitizeExternalMetadata(note))
		}
		registry.Repositories[i] = repo
		result.Repositories = append(result.Repositories, item)
	}
	sortWorkspaceRepositories(registry.Repositories)
	if err := s.saveWorkspaceRegistry(registry); err != nil {
		return WorkspaceScanResult{}, err
	}
	result.Output = renderWorkspaceScanResult(result)
	if err := s.saveWorkspaceScanResult(result); err != nil {
		return WorkspaceScanResult{}, err
	}
	return result, nil
}

func (s *Store) workspaceRegistryPath() string {
	return filepath.Join(s.Root, "workspace", "repos.json")
}

func (s *Store) workspaceScanCachePath() string {
	return filepath.Join(s.Root, "cache", "workspace-scan.json")
}

func (s *Store) loadWorkspaceRegistry() (WorkspaceRegistry, string, error) {
	path := s.workspaceRegistryPath()
	if err := s.requireInsideRoot(path); err != nil {
		return WorkspaceRegistry{}, "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return WorkspaceRegistry{Version: 1}, path, nil
		}
		return WorkspaceRegistry{}, "", err
	}
	var registry WorkspaceRegistry
	if err := json.Unmarshal(content, &registry); err != nil {
		return WorkspaceRegistry{}, "", err
	}
	if registry.Version == 0 {
		registry.Version = 1
	}
	for i := range registry.Repositories {
		registry.Repositories[i] = sanitizeWorkspaceRepository(registry.Repositories[i])
	}
	sortWorkspaceRepositories(registry.Repositories)
	return registry, path, nil
}

func (s *Store) saveWorkspaceRegistry(registry WorkspaceRegistry) error {
	registry.Version = 1
	for i := range registry.Repositories {
		registry.Repositories[i] = sanitizeWorkspaceRepository(registry.Repositories[i])
	}
	sortWorkspaceRepositories(registry.Repositories)
	path := s.workspaceRegistryPath()
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, append(data, '\n'), 0o600)
}

func (s *Store) saveWorkspaceScanResult(result WorkspaceScanResult) error {
	result = sanitizeWorkspaceScanResult(result)
	path := s.workspaceScanCachePath()
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, append(data, '\n'), 0o600)
}

func sanitizeWorkspaceRepository(repo WorkspaceRepository) WorkspaceRepository {
	repo.Name = sanitizeMetadataTitle(repo.Name)
	repo.ProjectSlug = Slugify(repo.ProjectSlug)
	repo.Path = sanitizeMetadataPath(repo.Path)
	repo.AddedAt = sanitizeExternalMetadata(repo.AddedAt)
	repo.LastScannedAt = sanitizeExternalMetadata(repo.LastScannedAt)
	return repo
}

func sanitizeWorkspaceScanResult(result WorkspaceScanResult) WorkspaceScanResult {
	result.Output = sanitizeExternalMetadata(result.Output)
	result.RegistryPath = sanitizeMetadataPath(result.RegistryPath)
	result.CachePath = sanitizeMetadataPath(result.CachePath)
	for i := range result.Repositories {
		result.Repositories[i].Repository = sanitizeWorkspaceRepository(result.Repositories[i].Repository)
		result.Repositories[i].Snapshot = sanitizeGitSnapshot(result.Repositories[i].Snapshot)
		result.Repositories[i].StatePath = sanitizeMetadataPath(result.Repositories[i].StatePath)
	}
	for i := range result.HealthNotes {
		result.HealthNotes[i] = sanitizeExternalMetadata(result.HealthNotes[i])
	}
	return result
}

func sortWorkspaceRepositories(repos []WorkspaceRepository) {
	sort.SliceStable(repos, func(i int, j int) bool {
		return strings.ToLower(repos[i].Path) < strings.ToLower(repos[j].Path)
	})
}

func samePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	return strings.EqualFold(left, right)
}

func renderWorkspaceAddResult(result WorkspaceAddResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "WORKSPACE\n")
	if result.AlreadyAdded {
		fmt.Fprintf(&b, "- Already tracked: %s\n", valueOr(result.Repository.Name, result.Repository.Path))
	} else {
		fmt.Fprintf(&b, "- Added: %s\n", valueOr(result.Repository.Name, result.Repository.Path))
	}
	fmt.Fprintf(&b, "- Project: %s\n", valueOr(result.Repository.ProjectSlug, "unknown"))
	fmt.Fprintf(&b, "- Path: %s\n", result.Repository.Path)
	fmt.Fprintf(&b, "- Registry: %s\n", result.RegistryPath)
	return b.String()
}

func renderWorkspaceListResult(result WorkspaceListResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "WORKSPACE\n")
	fmt.Fprintf(&b, "- Registry: %s\n", result.RegistryPath)
	if len(result.Repositories) == 0 {
		fmt.Fprintf(&b, "- No repositories are registered.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "\nREPOSITORIES\n")
	for _, repo := range result.Repositories {
		fmt.Fprintf(&b, "- %s [%s]\n", valueOr(repo.Name, repo.Path), valueOr(repo.ProjectSlug, "unknown"))
		fmt.Fprintf(&b, "  Path: %s\n", repo.Path)
		if repo.LastScannedAt != "" {
			fmt.Fprintf(&b, "  Last scanned: %s\n", repo.LastScannedAt)
		}
	}
	return b.String()
}

func renderWorkspaceScanResult(result WorkspaceScanResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "WORKSPACE SCAN\n")
	fmt.Fprintf(&b, "- Registry: %s\n", result.RegistryPath)
	fmt.Fprintf(&b, "- Cache: %s\n", result.CachePath)
	fmt.Fprintf(&b, "- Repositories: %d\n", len(result.Repositories))
	if len(result.Repositories) == 0 {
		fmt.Fprintf(&b, "- No repositories are registered. Add one with `leftoff workspace add <repo>`.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "\nREPOSITORIES\n")
	for _, item := range result.Repositories {
		repo := item.Repository
		snapshot := item.Snapshot
		if snapshot.IsRepo {
			fmt.Fprintf(&b, "- %s [%s]\n", valueOr(repo.Name, repo.Path), valueOr(snapshot.Branch, "unknown"))
			fmt.Fprintf(&b, "  Status: %s, changed paths: %d, ahead/behind: %d/%d, unpushed commits: %d\n", valueOr(snapshot.WorktreeStatus, "unknown"), len(snapshot.ChangedFiles), snapshot.Ahead, snapshot.Behind, snapshot.UnpushedCommits)
			fmt.Fprintf(&b, "  Stale branches: %d\n", len(snapshot.StaleBranches))
			if item.StatePath != "" {
				fmt.Fprintf(&b, "  State: %s\n", item.StatePath)
			}
		} else {
			fmt.Fprintf(&b, "- %s [unavailable]\n", valueOr(repo.Name, repo.Path))
			for _, note := range snapshot.HealthNotes {
				fmt.Fprintf(&b, "  %s\n", note)
			}
		}
	}

	if len(result.HealthNotes) > 0 {
		fmt.Fprintf(&b, "\nHEALTH\n")
		for _, note := range result.HealthNotes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}
	fmt.Fprintf(&b, "\nPRIVACY\n")
	fmt.Fprintf(&b, "- Workspace scan stores dirty state, branch, ahead/behind counts, unpushed commit count, redacted commit titles, stale branch names, worktree status, and saved leftoff records only.\n")
	return b.String()
}
