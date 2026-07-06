package leftoff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ValidateOptions struct {
	Repair bool
}

type ValidationIssue struct {
	Path       string
	Problem    string
	Repaired   bool
	BackupPath string
}

func (s *Store) Validate(opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue

	if _, err := os.Stat(s.Root); err != nil {
		if os.IsNotExist(err) && opts.Repair {
			if initErr := s.Init(); initErr != nil {
				return nil, initErr
			}
			return []ValidationIssue{{Path: s.Root, Problem: "store was missing", Repaired: true}}, nil
		}
		return []ValidationIssue{{Path: s.Root, Problem: fmt.Sprintf("store is not readable: %v", err)}}, nil
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(s.Root, dir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			issue := ValidationIssue{Path: path, Problem: "required directory is missing"}
			if opts.Repair {
				if err := s.ensureDir(path); err != nil {
					return issues, err
				}
				issue.Repaired = true
			}
			issues = append(issues, issue)
		}
	}

	for rel, content := range rootMarkdownFiles {
		path := filepath.Join(s.Root, rel)
		next, err := s.validateMarkdownFile(path, []byte(content), opts)
		if err != nil {
			return issues, err
		}
		issues = append(issues, next...)
	}

	config := filepath.Join(s.Root, "config.yml")
	next, err := s.validateMarkdownFile(config, []byte("# leftoff config\n"), opts)
	if err != nil {
		return issues, err
	}
	issues = append(issues, next...)

	projectsDir := filepath.Join(s.Root, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return issues, nil
		}
		return issues, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		paths := s.ProjectPaths(slug)
		for name, content := range projectMarkdownFiles {
			path := filepath.Join(paths.Dir, name)
			next, err := s.validateMarkdownFile(path, []byte(content), opts)
			if err != nil {
				return issues, err
			}
			issues = append(issues, next...)
		}

		next, err := s.validateJSONL(paths.Activity, opts)
		if err != nil {
			return issues, err
		}
		issues = append(issues, next...)

		next, err = s.validateExternalMetadataMarkdown(paths.State, opts)
		if err != nil {
			return issues, err
		}
		issues = append(issues, next...)
	}

	next, err = s.validateGitHubCacheFiles(opts)
	if err != nil {
		return issues, err
	}
	issues = append(issues, next...)

	next, err = s.validateWorkspaceMetadataFiles(opts)
	if err != nil {
		return issues, err
	}
	issues = append(issues, next...)

	return issues, nil
}

func (s *Store) validateMarkdownFile(path string, defaultContent []byte, opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	if err := s.requireInsideRoot(path); err != nil {
		return issues, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			issue := ValidationIssue{Path: path, Problem: "required file is missing"}
			if opts.Repair {
				if err := s.ensureFile(path, defaultContent); err != nil {
					return issues, err
				}
				issue.Repaired = true
			}
			return append(issues, issue), nil
		}
		return issues, err
	}

	if firstMarkdownLine(content) == "" || !strings.HasPrefix(firstMarkdownLine(content), "#") {
		issue := ValidationIssue{Path: path, Problem: "markdown file does not start with a heading"}
		if opts.Repair {
			backup, err := s.BackupFile(path, "markdown repair: missing leading heading")
			if err != nil {
				return issues, err
			}
			repaired := strings.TrimRight(string(defaultContent), "\n") + "\n\n" + strings.TrimLeft(string(content), "\n")
			if err := atomicWriteFile(path, []byte(repaired), 0o600); err != nil {
				return issues, err
			}
			issue.Repaired = true
			issue.BackupPath = backup
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

func firstMarkdownLine(content []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

func (s *Store) validateJSONL(path string, opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	if err := s.requireInsideRoot(path); err != nil {
		return issues, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			issue := ValidationIssue{Path: path, Problem: "activity JSONL is missing"}
			if opts.Repair {
				if err := s.ensureFile(path, []byte("")); err != nil {
					return issues, err
				}
				issue.Repaired = true
			}
			return append(issues, issue), nil
		}
		return issues, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNo := 0
	var validLines []string
	var invalid []string
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			invalid = append(invalid, fmt.Sprintf("line %d", lineNo))
			continue
		}
		validLines = append(validLines, line)
	}
	if err := scanner.Err(); err != nil {
		return issues, err
	}

	if len(invalid) == 0 {
		return issues, nil
	}

	issue := ValidationIssue{Path: path, Problem: "activity JSONL has invalid entries: " + strings.Join(invalid, ", ")}
	if opts.Repair {
		backup, err := s.BackupFile(path, "jsonl repair: removed invalid activity entries")
		if err != nil {
			return issues, err
		}
		repaired := strings.Join(validLines, "\n")
		if repaired != "" {
			repaired += "\n"
		}
		if err := atomicWriteFile(path, []byte(repaired), 0o600); err != nil {
			return issues, err
		}
		issue.Repaired = true
		issue.BackupPath = backup
	}

	return append(issues, issue), nil
}

func (s *Store) validateExternalMetadataMarkdown(path string, opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	if err := s.requireInsideRoot(path); err != nil {
		return issues, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return issues, nil
		}
		return issues, err
	}
	if !HasLikelySecret(string(content)) {
		return issues, nil
	}

	issue := ValidationIssue{Path: path, Problem: "external metadata contains secret-like text"}
	if opts.Repair {
		backup, err := s.BackupFile(path, "external metadata repair: redacted secret-like text")
		if err != nil {
			return issues, err
		}
		if err := atomicWriteFile(path, []byte(sanitizeExternalMetadata(string(content))), 0o600); err != nil {
			return issues, err
		}
		issue.Repaired = true
		issue.BackupPath = backup
	}
	return append(issues, issue), nil
}

func (s *Store) validateGitHubCacheFiles(opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	cacheDir := filepath.Join(s.Root, "cache", "github")
	if err := s.requireInsideRoot(cacheDir); err != nil {
		return issues, err
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return issues, nil
		}
		return issues, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(cacheDir, entry.Name())
		next, err := s.validateGitHubCacheFile(path, opts)
		if err != nil {
			return issues, err
		}
		issues = append(issues, next...)
	}
	return issues, nil
}

func (s *Store) validateGitHubCacheFile(path string, opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	if err := s.requireInsideRoot(path); err != nil {
		return issues, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return issues, err
	}
	if !HasLikelySecret(string(content)) {
		return issues, nil
	}

	issue := ValidationIssue{Path: path, Problem: "GitHub cache contains secret-like external metadata"}
	if opts.Repair {
		backup, err := s.BackupFile(path, "GitHub cache repair: redacted secret-like external metadata")
		if err != nil {
			return issues, err
		}
		repaired := sanitizeExternalMetadata(string(content))
		var cache GitHubCache
		if err := json.Unmarshal(content, &cache); err == nil {
			cache = sanitizeGitHubCache(cache)
			data, marshalErr := json.MarshalIndent(cache, "", "  ")
			if marshalErr != nil {
				return issues, marshalErr
			}
			data = append(data, '\n')
			repaired = string(data)
		}
		if err := atomicWriteFile(path, []byte(repaired), 0o600); err != nil {
			return issues, err
		}
		issue.Repaired = true
		issue.BackupPath = backup
	}
	return append(issues, issue), nil
}

func (s *Store) validateWorkspaceMetadataFiles(opts ValidateOptions) ([]ValidationIssue, error) {
	var issues []ValidationIssue
	paths := []string{
		s.workspaceRegistryPath(),
		s.workspaceScanCachePath(),
	}
	for _, path := range paths {
		if err := s.requireInsideRoot(path); err != nil {
			return issues, err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return issues, err
		}
		if !HasLikelySecret(string(content)) {
			continue
		}
		issue := ValidationIssue{Path: path, Problem: "workspace metadata contains secret-like external metadata"}
		if opts.Repair {
			backup, err := s.BackupFile(path, "workspace metadata repair: redacted secret-like external metadata")
			if err != nil {
				return issues, err
			}
			if err := atomicWriteFile(path, []byte(sanitizeExternalMetadata(string(content))), 0o600); err != nil {
				return issues, err
			}
			issue.Repaired = true
			issue.BackupPath = backup
		}
		issues = append(issues, issue)
	}
	return issues, nil
}
