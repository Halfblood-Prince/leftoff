package leftoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Store struct {
	Root  string
	Clock func() time.Time
}

type ProjectPaths struct {
	Dir            string
	Project        string
	State          string
	Decisions      string
	OpenLoops      string
	SolvedProblems string
	Releases       string
	Friction       string
	Activity       string
}

var rootMarkdownFiles = map[string]string{
	"profile.md":                     "# leftoff profile\n\nUse this file for durable preferences the user explicitly chooses to store.\n",
	"inbox.md":                       "# leftoff inbox\n\nExplicit captures that are not linked to a clear project live here.\n",
	"patterns/recurring-friction.md": "# Recurring friction\n\nNo recurring friction captured yet.\n",
	"patterns/reusable-recipes.md":   "# Reusable recipes\n\nNo reusable recipes captured yet.\n",
}

var requiredDirs = []string{
	"projects",
	"patterns",
	"weekly",
	"cache",
	"workspace",
	"backups",
}

var projectMarkdownFiles = map[string]string{
	"project.md":         "# Project\n\n",
	"state.md":           "# State\n\nNo local Git state captured yet.\n",
	"decisions.md":       "# Decisions\n\nNo decisions captured yet.\n",
	"open-loops.md":      "# Open loops\n\nNo open loops captured yet.\n",
	"solved-problems.md": "# Solved problems\n\nNo solved problems captured yet.\n",
	"releases.md":        "# Releases\n\nNo release intents captured yet.\n",
	"friction.md":        "# Friction\n\nNo friction events captured yet.\n",
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		defaultRoot, err := DefaultStoreRoot()
		if err != nil {
			return nil, err
		}
		root = defaultRoot
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	return &Store{Root: filepath.Clean(abs), Clock: time.Now}, nil
}

func DefaultStoreRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".leftoff"), nil
}

func (s *Store) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

func (s *Store) today() string {
	return s.now().Format("2006-01-02")
}

func (s *Store) Init() error {
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return fmt.Errorf("create store root: %w", err)
	}

	for _, dir := range requiredDirs {
		if err := s.ensureDir(filepath.Join(s.Root, dir)); err != nil {
			return err
		}
	}

	if err := s.ensureFile(filepath.Join(s.Root, ".leftoff-store"), []byte("leftoff store format: 1\n")); err != nil {
		return err
	}

	for rel, content := range rootMarkdownFiles {
		if err := s.ensureFile(filepath.Join(s.Root, rel), []byte(content)); err != nil {
			return err
		}
	}

	config := []byte(`# leftoff config

priority_weights:
  urgency: medium
  blocker_resolution_value: medium
  release_impact: medium
  recency: low
  user_focus_match: medium
  fit_for_available_time: medium
  uncertainty_penalty: medium
  dependency_penalty: medium
`)
	if err := s.ensureFile(filepath.Join(s.Root, "config.yml"), config); err != nil {
		return err
	}

	if err := s.ensureFile(filepath.Join(s.Root, "cache", "local-scan-metadata.json"), []byte("{}\n")); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureDir(path string) error {
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o700)
}

func (s *Store) ensureFile(path string, content []byte) error {
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicWriteFile(path, content, 0o600)
}

func (s *Store) requireInsideRoot(path string) error {
	absRoot, err := filepath.Abs(s.Root)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes leftoff store: %s", path)
	}
	return nil
}

func atomicWriteFile(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *Store) AppendMarkdownSection(path string, section string) error {
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}

	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(current)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if !strings.HasSuffix(content, "\n\n") {
		content += "\n"
	}
	content += strings.TrimRight(section, "\n") + "\n"

	return atomicWriteFile(path, []byte(content), 0o600)
}

func (s *Store) AppendJSONL(path string, event ActivityEvent) error {
	if err := s.requireInsideRoot(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func (s *Store) ProjectPaths(slug string) ProjectPaths {
	dir := filepath.Join(s.Root, "projects", slug)
	return ProjectPaths{
		Dir:            dir,
		Project:        filepath.Join(dir, "project.md"),
		State:          filepath.Join(dir, "state.md"),
		Decisions:      filepath.Join(dir, "decisions.md"),
		OpenLoops:      filepath.Join(dir, "open-loops.md"),
		SolvedProblems: filepath.Join(dir, "solved-problems.md"),
		Releases:       filepath.Join(dir, "releases.md"),
		Friction:       filepath.Join(dir, "friction.md"),
		Activity:       filepath.Join(dir, "activity.jsonl"),
	}
}

func (s *Store) EnsureProject(meta ProjectMeta) (ProjectPaths, error) {
	if strings.TrimSpace(meta.Slug) == "" {
		meta.Slug = Slugify(meta.Name)
	}
	if strings.TrimSpace(meta.Slug) == "" {
		return ProjectPaths{}, errors.New("project slug is empty")
	}
	if strings.TrimSpace(meta.Name) == "" {
		meta.Name = meta.Slug
	}
	if meta.Created.IsZero() {
		meta.Created = s.now()
	}

	paths := s.ProjectPaths(meta.Slug)
	if err := s.ensureDir(paths.Dir); err != nil {
		return ProjectPaths{}, err
	}

	projectContent := fmt.Sprintf(`# %s

- Name: %s
- Slug: %s
- Remote: %s
- Local path: %s
- Created: %s

## Notes

No project notes captured yet.
`, meta.Name, meta.Name, meta.Slug, valueOr(meta.Remote, "unknown"), valueOr(meta.LocalPath, "unknown"), meta.Created.Format("2006-01-02"))

	if err := s.ensureFile(paths.Project, []byte(projectContent)); err != nil {
		return ProjectPaths{}, err
	}

	for name, content := range projectMarkdownFiles {
		if name == "project.md" {
			continue
		}
		if err := s.ensureFile(filepath.Join(paths.Dir, name), []byte(content)); err != nil {
			return ProjectPaths{}, err
		}
	}

	if err := s.ensureFile(paths.Activity, []byte("")); err != nil {
		return ProjectPaths{}, err
	}

	return paths, nil
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func Slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *Store) NextRecordID(recordType RecordType, path string, date string) (string, error) {
	if err := s.requireInsideRoot(path); err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	prefix := recordType.Prefix()
	pattern := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(prefix) + `-` + regexp.QuoteMeta(date) + `-(\d{3})\b`)
	max := 0
	for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > max {
			max = n
		}
	}

	return fmt.Sprintf("%s-%s-%03d", prefix, date, max+1), nil
}

func (s *Store) BackupFile(path string, reason string) (string, error) {
	if err := s.requireInsideRoot(path); err != nil {
		return "", err
	}

	rel, err := filepath.Rel(s.Root, path)
	if err != nil {
		return "", err
	}
	stamp := s.now().Format("20060102T150405")
	dest := filepath.Join(s.Root, "backups", stamp, rel)
	if err := s.requireInsideRoot(dest); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", err
	}

	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}

	meta := filepath.Join(filepath.Dir(dest), filepath.Base(dest)+".backup-reason.txt")
	_ = atomicWriteFile(meta, []byte(reason+"\n"), 0o600)
	return dest, nil
}
