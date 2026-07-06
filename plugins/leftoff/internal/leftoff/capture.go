package leftoff

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrDuplicateCapture = errors.New("duplicate capture")
	ErrSecretCapture    = errors.New("capture contains a likely secret")
	ErrCaptureTooLarge  = errors.New("capture is too large; save a concise summary instead")
)

type parsedCapture struct {
	Type RecordType
	Text string
}

func (s *Store) Capture(ctx context.Context, req CaptureRequest) (CaptureResult, error) {
	if err := s.Init(); err != nil {
		return CaptureResult{}, err
	}

	parsed, err := ParseCapture(req.Kind, req.Text)
	if err != nil {
		return CaptureResult{}, err
	}

	if captureTooLarge(parsed.Text) {
		return CaptureResult{}, ErrCaptureTooLarge
	}

	if findings := FindSecrets(parsed.Text); len(findings) > 0 {
		names := make([]string, 0, len(findings))
		for _, finding := range findings {
			names = append(names, finding.Name)
		}
		return CaptureResult{}, fmt.Errorf("%w: %s", ErrSecretCapture, strings.Join(names, ", "))
	}

	project := s.resolveCaptureProject(ctx, req)
	var targetPath string
	projectLinked := project.Slug != ""

	if projectLinked {
		paths, err := s.EnsureProject(project)
		if err != nil {
			return CaptureResult{}, err
		}
		targetPath = projectTargetPath(paths, parsed.Type)
	} else {
		targetPath = filepath.Join(s.Root, "inbox.md")
	}

	fingerprint := captureFingerprint(parsed.Type, parsed.Text, project.Slug)
	if duplicate, err := s.hasFingerprint(targetPath, fingerprint); err != nil {
		return CaptureResult{}, err
	} else if duplicate {
		return CaptureResult{}, ErrDuplicateCapture
	}

	id, err := s.NextRecordID(parsed.Type, targetPath, s.today())
	if err != nil {
		return CaptureResult{}, err
	}

	section := renderCaptureRecord(id, parsed, project.Slug, projectLinked, s.today(), fingerprint)
	if err := s.AppendMarkdownSection(targetPath, section); err != nil {
		return CaptureResult{}, err
	}

	event := ActivityEvent{
		Timestamp:  s.now().Format(timeFormatRFC3339()),
		Kind:       "capture",
		RecordID:   id,
		RecordType: string(parsed.Type),
		Project:    project.Slug,
		Summary:    cleanSummary(parsed.Text, 180),
		Evidence:   "User capture.",
	}

	if projectLinked {
		if err := s.AppendJSONL(s.ProjectPaths(project.Slug).Activity, event); err != nil {
			return CaptureResult{}, err
		}
	}

	return CaptureResult{
		ID:          id,
		Type:        parsed.Type,
		ProjectSlug: project.Slug,
		Path:        targetPath,
		Activity:    event,
	}, nil
}

func ParseCapture(kind string, raw string) (parsedCapture, error) {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "/capture")
	text = strings.TrimSpace(text)
	if text == "" {
		return parsedCapture{}, errors.New("capture text is empty")
	}

	if strings.TrimSpace(kind) != "" {
		recordType, ok := ParseRecordType(kind)
		if !ok || recordType == RecordActivityEvent {
			return parsedCapture{}, fmt.Errorf("unknown capture type: %s", kind)
		}
		return parsedCapture{Type: recordType, Text: text}, nil
	}

	if idx := strings.Index(text, ":"); idx > 0 && idx <= 40 {
		prefix := strings.TrimSpace(text[:idx])
		if recordType, ok := ParseRecordType(prefix); ok && recordType != RecordActivityEvent {
			body := strings.TrimSpace(text[idx+1:])
			if body == "" {
				return parsedCapture{}, errors.New("capture body is empty")
			}
			return parsedCapture{Type: recordType, Text: body}, nil
		}
	}

	return parsedCapture{Type: inferRecordType(text), Text: text}, nil
}

func inferRecordType(text string) RecordType {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case strings.HasPrefix(lower, "idea ") || strings.HasPrefix(lower, "maybe ") || strings.HasPrefix(lower, "what if "):
		return RecordIdea
	case strings.HasPrefix(lower, "decision ") || strings.HasPrefix(lower, "decided ") || strings.HasPrefix(lower, "use ") && strings.Contains(lower, " because "):
		return RecordDecision
	case strings.HasPrefix(lower, "problem ") || strings.Contains(lower, " failed") || strings.Contains(lower, " error"):
		return RecordProblem
	case strings.HasPrefix(lower, "solution ") || strings.HasPrefix(lower, "fix ") || strings.HasPrefix(lower, "workaround "):
		return RecordSolution
	case strings.HasPrefix(lower, "follow up") || strings.HasPrefix(lower, "promise ") || strings.HasPrefix(lower, "waiting "):
		return RecordOpenLoop
	default:
		return RecordTask
	}
}

func captureTooLarge(text string) bool {
	if len(text) > 4000 {
		return true
	}
	lines := strings.Count(text, "\n") + 1
	return lines > 40
}

func (s *Store) resolveCaptureProject(ctx context.Context, req CaptureRequest) ProjectMeta {
	if strings.TrimSpace(req.Project) != "" {
		name := strings.TrimSpace(req.Project)
		return ProjectMeta{Name: name, Slug: Slugify(name), Created: s.now()}
	}

	if strings.TrimSpace(req.RepoPath) == "" {
		return ProjectMeta{}
	}

	snapshot := InspectRepository(ctx, req.RepoPath, s.now)
	if snapshot.IsRepo {
		meta := ProjectFromSnapshot(snapshot)
		meta.Created = s.now()
		return meta
	}

	if abs, err := filepath.Abs(req.RepoPath); err == nil {
		if info, statErr := os.Stat(abs); statErr == nil && info.IsDir() {
			name := filepath.Base(abs)
			return ProjectMeta{Name: name, Slug: Slugify(name), LocalPath: abs, Created: s.now()}
		}
	}

	return ProjectMeta{}
}

func projectTargetPath(paths ProjectPaths, recordType RecordType) string {
	switch recordType.DestinationFile() {
	case "decisions.md":
		return paths.Decisions
	case "solved-problems.md":
		return paths.SolvedProblems
	case "releases.md":
		return paths.Releases
	case "friction.md":
		return paths.Friction
	default:
		return paths.OpenLoops
	}
}

func captureFingerprint(recordType RecordType, text string, project string) string {
	sum := sha256.Sum256([]byte(string(recordType) + "\n" + strings.ToLower(strings.Join(strings.Fields(text), " ")) + "\n" + project))
	return fmt.Sprintf("%x", sum[:])[:16]
}

func (s *Store) hasFingerprint(path string, fingerprint string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(content), "sha256:"+fingerprint), nil
}

func renderCaptureRecord(id string, parsed parsedCapture, project string, projectLinked bool, date string, fingerprint string) string {
	summary := cleanSummary(parsed.Text, 180)
	status := parsed.Type.DefaultStatus(projectLinked)
	projectLabel := "inbox"
	if project != "" {
		projectLabel = project
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## %s - %s\n\n", id, summary)
	fmt.Fprintf(&b, "- Type: %s\n", parsed.Type)
	fmt.Fprintf(&b, "- Status: %s\n", status)
	fmt.Fprintf(&b, "- Project: %s\n", projectLabel)

	switch parsed.Type {
	case RecordDecision:
		fmt.Fprintf(&b, "- Date: %s\n", date)
		fmt.Fprintf(&b, "- Decision: %s\n", summary)
		fmt.Fprintf(&b, "- Context: User capture.\n")
		fmt.Fprintf(&b, "- Alternatives rejected: none recorded\n")
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Revisit when: assumptions change or the user requests reconsideration.\n")
	case RecordProblem:
		fmt.Fprintf(&b, "- Created: %s\n", date)
		fmt.Fprintf(&b, "- Last touched: %s\n", date)
		fmt.Fprintf(&b, "- Problem: %s\n", summary)
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Verified: not yet\n")
		fmt.Fprintf(&b, "- Linked solution: none recorded\n")
	case RecordSolution:
		fmt.Fprintf(&b, "- Created: %s\n", date)
		fmt.Fprintf(&b, "- Last touched: %s\n", date)
		fmt.Fprintf(&b, "- Solution: %s\n", summary)
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Verified: not marked verified\n")
		fmt.Fprintf(&b, "- Linked problem: none recorded\n")
	case RecordReleaseIntent:
		fmt.Fprintf(&b, "- Created: %s\n", date)
		fmt.Fprintf(&b, "- Last touched: %s\n", date)
		fmt.Fprintf(&b, "- Intent: %s\n", summary)
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Ship condition: not specified\n")
	case RecordFrictionEvent:
		fmt.Fprintf(&b, "- Created: %s\n", date)
		fmt.Fprintf(&b, "- Last touched: %s\n", date)
		fmt.Fprintf(&b, "- Observation: %s\n", summary)
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Impact: not estimated\n")
	default:
		fmt.Fprintf(&b, "- Priority: unspecified\n")
		fmt.Fprintf(&b, "- Effort: unknown\n")
		fmt.Fprintf(&b, "- Created: %s\n", date)
		fmt.Fprintf(&b, "- Last touched: %s\n", date)
		fmt.Fprintf(&b, "- Summary: %s\n", summary)
		fmt.Fprintf(&b, "- Evidence: User capture.\n")
		fmt.Fprintf(&b, "- Next action: Clarify the next concrete step.\n")
		fmt.Fprintf(&b, "- Blocked by: none recorded\n")
	}

	fmt.Fprintf(&b, "- Fingerprint: sha256:%s\n", fingerprint)
	return b.String()
}

func cleanSummary(text string, limit int) string {
	clean := strings.Join(strings.Fields(text), " ")
	if clean == "" {
		return "untitled"
	}

	runes := []rune(clean)
	if len(runes) <= limit {
		return clean
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func timeFormatRFC3339() string {
	return "2006-01-02T15:04:05Z07:00"
}
