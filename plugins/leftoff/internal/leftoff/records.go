package leftoff

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type MarkdownRecord struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Type        RecordType        `json:"type"`
	Status      string            `json:"status"`
	Project     string            `json:"project,omitempty"`
	Path        string            `json:"path"`
	Fields      map[string]string `json:"fields,omitempty"`
	Body        string            `json:"body,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
	LastTouched time.Time         `json:"last_touched,omitempty"`
	Date        time.Time         `json:"date,omitempty"`
	Index       int               `json:"index"`
}

type RecordQuery struct {
	Project      string
	Types        map[RecordType]bool
	IncludeInbox bool
}

var recordHeadingPattern = regexp.MustCompile(`^##\s+([A-Z]+(?:-[A-Z]+)*-\d{4}-\d{2}-\d{2}-\d{3})\s*(?:-\s*(.*))?$`)

func (r MarkdownRecord) Field(name string) string {
	if r.Fields == nil {
		return ""
	}
	return strings.TrimSpace(r.Fields[strings.ToLower(name)])
}

func (r MarkdownRecord) Text() string {
	var parts []string
	parts = append(parts, r.ID, r.Title, string(r.Type), r.Status, r.Project)
	keys := make([]string, 0, len(r.Fields))
	for key := range r.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, key, r.Fields[key])
	}
	return strings.Join(parts, " ")
}

func (r MarkdownRecord) PrimaryText() string {
	for _, field := range []string{"summary", "decision", "problem", "solution", "intent", "observation"} {
		if value := r.Field(field); value != "" {
			return value
		}
	}
	return r.Title
}

func (r MarkdownRecord) EffectiveDate() time.Time {
	if !r.LastTouched.IsZero() {
		return r.LastTouched
	}
	if !r.Date.IsZero() {
		return r.Date
	}
	if !r.Created.IsZero() {
		return r.Created
	}
	return time.Time{}
}

func (r MarkdownRecord) HasType(recordType RecordType) bool {
	return r.Type == recordType
}

func (s *Store) LoadRecords(query RecordQuery) ([]MarkdownRecord, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	var records []MarkdownRecord
	includeInbox := query.IncludeInbox || strings.TrimSpace(query.Project) == ""
	if includeInbox {
		inbox, err := s.loadRecordsFromFile(filepath.Join(s.Root, "inbox.md"), "")
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		records = append(records, inbox...)
	}

	projectSlugs, err := s.ProjectSlugs(query.Project)
	if err != nil {
		return nil, err
	}

	for _, slug := range projectSlugs {
		paths := s.ProjectPaths(slug)
		for _, path := range []string{paths.OpenLoops, paths.Decisions, paths.SolvedProblems, paths.Releases, paths.Friction} {
			next, err := s.loadRecordsFromFile(path, slug)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			records = append(records, next...)
		}
	}

	if len(query.Types) > 0 {
		filtered := records[:0]
		for _, record := range records {
			if query.Types[record.Type] {
				filtered = append(filtered, record)
			}
		}
		records = filtered
	}

	sort.SliceStable(records, func(i int, j int) bool {
		left := records[i].EffectiveDate()
		right := records[j].EffectiveDate()
		if left.Equal(right) {
			return records[i].ID < records[j].ID
		}
		if left.IsZero() {
			return false
		}
		if right.IsZero() {
			return true
		}
		return left.After(right)
	})

	return records, nil
}

func (s *Store) ProjectSlugs(filter string) ([]string, error) {
	filter = Slugify(filter)
	if filter != "" {
		return []string{filter}, nil
	}

	projectsDir := filepath.Join(s.Root, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var slugs []string
	for _, entry := range entries {
		if entry.IsDir() {
			slugs = append(slugs, entry.Name())
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

func (s *Store) loadRecordsFromFile(path string, project string) ([]MarkdownRecord, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseMarkdownRecords(string(content), path, project), nil
}

func ParseMarkdownRecords(content string, sourcePath string, defaultProject string) []MarkdownRecord {
	var records []MarkdownRecord
	var current *MarkdownRecord
	var body strings.Builder
	index := 0

	flush := func() {
		if current == nil {
			return
		}
		current.Body = strings.TrimRight(body.String(), "\n")
		hydrateRecordFields(current)
		records = append(records, *current)
		current = nil
		body.Reset()
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if match := recordHeadingPattern.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			flush()
			recordType, ok := RecordTypeFromID(match[1])
			if !ok {
				continue
			}
			index++
			current = &MarkdownRecord{
				ID:      match[1],
				Title:   strings.TrimSpace(match[2]),
				Type:    recordType,
				Project: defaultProject,
				Path:    sourcePath,
				Fields:  map[string]string{},
				Index:   index,
			}
			continue
		}

		if current == nil {
			continue
		}
		body.WriteString(line)
		body.WriteString("\n")
		if key, value, ok := parseBulletField(line); ok {
			current.Fields[strings.ToLower(key)] = value
		}
	}
	flush()
	return records
}

func parseBulletField(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "- ") || !strings.Contains(line, ":") {
		return "", "", false
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
	parts := strings.SplitN(line, ":", 2)
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func hydrateRecordFields(record *MarkdownRecord) {
	if value := record.Field("type"); value != "" {
		if recordType, ok := ParseRecordType(value); ok {
			record.Type = recordType
		}
	}
	if value := record.Field("status"); value != "" {
		record.Status = NormalizeStatus(value)
	}
	if record.Status == "" {
		record.Status = NormalizeStatus(record.Type.DefaultStatus(record.Project != "" && record.Project != "inbox"))
	}
	if value := record.Field("project"); value != "" && record.Project == "" && value != "inbox" {
		record.Project = Slugify(value)
	}
	if record.Project == "" {
		record.Project = "inbox"
	}
	record.Created = parseDateField(record.Field("created"))
	record.LastTouched = parseDateField(record.Field("last touched"))
	record.Date = parseDateField(record.Field("date"))
}

func RecordTypeFromID(id string) (RecordType, bool) {
	prefix := id
	if idx := strings.Index(prefix, "-20"); idx > 0 {
		prefix = prefix[:idx]
	}
	switch prefix {
	case "TASK":
		return RecordTask, true
	case "IDEA":
		return RecordIdea, true
	case "DECISION":
		return RecordDecision, true
	case "PROBLEM":
		return RecordProblem, true
	case "SOLUTION":
		return RecordSolution, true
	case "OPEN-LOOP":
		return RecordOpenLoop, true
	case "RELEASE":
		return RecordReleaseIntent, true
	case "FRICTION":
		return RecordFrictionEvent, true
	default:
		return "", false
	}
}

func NormalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.ReplaceAll(status, "_", "-")
	status = strings.ReplaceAll(status, " ", "-")
	switch status {
	case "", "unknown":
		return ""
	case "todo", "to-do", "new":
		return string(StatusInbox)
	case "active", "doing", "in-progress", "inprogress", "open":
		return string(StatusActive)
	case "blocked", "blocker":
		return string(StatusBlocked)
	case "waiting", "waiting-on", "pending":
		return string(StatusWaiting)
	case "parked", "someday", "later", "backlog":
		return string(StatusParked)
	case "done", "closed", "complete", "completed", "shipped":
		return string(StatusDone)
	case "accepted", "verified":
		return status
	case "abandoned", "dropped", "wontfix":
		return string(StatusAbandoned)
	default:
		return status
	}
}

func parseDateField(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "unknown" {
		return time.Time{}
	}
	if len(value) >= len("2006-01-02") {
		value = value[:len("2006-01-02")]
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (s *Store) LoadAllActivities(project string) ([]ActivityEvent, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	slugs, err := s.ProjectSlugs(project)
	if err != nil {
		return nil, err
	}

	var events []ActivityEvent
	for _, slug := range slugs {
		path := s.ProjectPaths(slug).Activity
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var event ActivityEvent
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				if event.Project == "" {
					event.Project = slug
				}
				events = append(events, event)
			}
		}
	}

	sort.SliceStable(events, func(i int, j int) bool {
		return events[i].Timestamp > events[j].Timestamp
	})
	return events, nil
}

func withinRange(t time.Time, start time.Time, end time.Time) bool {
	if t.IsZero() {
		return false
	}
	return !t.Before(start) && t.Before(end)
}
