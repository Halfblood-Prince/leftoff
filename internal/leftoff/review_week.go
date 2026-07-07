package leftoff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReviewWeekRequest struct {
	Project string
	Week    string
	Write   bool
}

type ReviewWeekResult struct {
	Output string
	Path   string
	Week   string
}

func (s *Store) ReviewWeek(req ReviewWeekRequest) (ReviewWeekResult, error) {
	records, err := s.LoadRecords(RecordQuery{Project: req.Project, IncludeInbox: req.Project == ""})
	if err != nil {
		return ReviewWeekResult{}, err
	}
	events, err := s.LoadAllActivities(req.Project)
	if err != nil {
		return ReviewWeekResult{}, err
	}

	start, end, label := weekRange(req.Week, s.now())
	friction := DetectFriction(records, events, s.now())
	output := renderWeeklyReview(records, events, friction, req.Project, label, start, end)

	result := ReviewWeekResult{Output: output, Week: label}
	if req.Write {
		path := filepath.Join(s.Root, "weekly", label+".md")
		if err := s.requireInsideRoot(path); err != nil {
			return ReviewWeekResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return ReviewWeekResult{}, err
		}
		if err := atomicWriteFile(path, []byte(output), 0o600); err != nil {
			return ReviewWeekResult{}, err
		}
		result.Path = path
	}
	return result, nil
}

func weekRange(label string, now time.Time) (time.Time, time.Time, string) {
	if strings.TrimSpace(label) == "" {
		year, week := now.ISOWeek()
		label = fmt.Sprintf("%04d-W%02d", year, week)
	}
	year := now.Year()
	week := 1
	_, _ = fmt.Sscanf(label, "%d-W%d", &year, &week)
	start := isoWeekStart(year, week, now.Location())
	return start, start.AddDate(0, 0, 7), fmt.Sprintf("%04d-W%02d", year, week)
}

func isoWeekStart(year int, week int, location *time.Location) time.Time {
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, location)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, 1-weekday)
	return monday.AddDate(0, 0, (week-1)*7)
}

func renderWeeklyReview(records []MarkdownRecord, events []ActivityEvent, friction []FrictionFinding, project string, label string, start time.Time, end time.Time) string {
	shipped := []MarkdownRecord{}
	active := []MarkdownRecord{}
	blocked := []MarkdownRecord{}
	neglected := []MarkdownRecord{}
	decisions := []MarkdownRecord{}
	lessons := []MarkdownRecord{}

	for _, record := range records {
		status := NormalizeStatus(record.Status)
		effective := record.EffectiveDate()
		switch {
		case record.Type == RecordDecision && withinRange(record.Date, start, end):
			decisions = append(decisions, record)
		case (record.Type == RecordSolution || record.Type == RecordProblem) && withinRange(effective, start, end):
			lessons = append(lessons, record)
		case status == "done" || status == "completed" || status == "closed":
			if withinRange(effective, start, end) {
				shipped = append(shipped, record)
			}
		case status == string(StatusBlocked) || status == string(StatusWaiting):
			blocked = append(blocked, record)
		case isTaskLike(record) && (status == string(StatusActive) || status == string(StatusInbox)):
			active = append(active, record)
			if !effective.IsZero() && startOfDay(start).Sub(startOfDay(effective)) >= 14*24*time.Hour {
				neglected = append(neglected, record)
			}
		}
	}

	sortRecordsForReview(shipped)
	sortRecordsForReview(active)
	sortRecordsForReview(blocked)
	sortRecordsForReview(neglected)
	sortRecordsForReview(decisions)
	sortRecordsForReview(lessons)

	var b strings.Builder
	fmt.Fprintf(&b, "# Weekly Review %s\n\n", label)
	if project != "" {
		fmt.Fprintf(&b, "- Project: %s\n", Slugify(project))
	}
	fmt.Fprintf(&b, "- Week start: %s\n", start.Format("2006-01-02"))
	fmt.Fprintf(&b, "- Week end: %s\n", end.AddDate(0, 0, -1).Format("2006-01-02"))
	fmt.Fprintf(&b, "- Evidence: local Markdown records and leftoff activity JSONL only.\n")
	fmt.Fprintf(&b, "- Activity note: command activity is not counted as meaningful progress unless a record, decision, solution, or state changed.\n\n")

	writeRecordSection(&b, "Shipped", shipped, "No done records were found for this week.")
	writeRecordSection(&b, "Active", active, "No active task records were found.")
	writeRecordSection(&b, "Blocked", blocked, "No blocked or waiting records were found.")
	writeRecordSection(&b, "Neglected", neglected, "No active task appears neglected from local dates.")
	writeRecordSection(&b, "Decisions Made", decisions, "No decisions were recorded this week.")
	writeRecordSection(&b, "Lessons Learned", lessons, "No problem or solution records were captured this week.")

	fmt.Fprintf(&b, "## Repeated Friction\n\n")
	if len(friction) == 0 {
		fmt.Fprintf(&b, "- No recurring friction pattern found. Single events were not promoted to patterns.\n\n")
	} else {
		for _, finding := range firstFriction(friction, 3) {
			fmt.Fprintf(&b, "- %s: %s impact, %s confidence\n", finding.Pattern, finding.Impact, finding.Confidence)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Suggested Priorities\n\n")
	if len(blocked) > 0 {
		fmt.Fprintf(&b, "- Resolve or re-scope the oldest blocked item: %s.\n", blocked[len(blocked)-1].Title)
	} else if len(active) > 0 {
		fmt.Fprintf(&b, "- Continue the highest-evidence active item: %s.\n", active[0].Title)
	} else {
		fmt.Fprintf(&b, "- Capture one concrete next task before prioritising the next week.\n")
	}
	if len(neglected) > 0 {
		fmt.Fprintf(&b, "- Review neglected work and either add a next action, park it, or abandon it.\n")
	}
	fmt.Fprintf(&b, "\n## Evidence Gaps\n\n")
	if len(events) == 0 {
		fmt.Fprintf(&b, "- No activity JSONL events were found for this scope.\n")
	}
	if len(records) == 0 {
		fmt.Fprintf(&b, "- No Markdown records were found for this scope.\n")
	}
	fmt.Fprintf(&b, "- Remote PRs, issues, CI, and calendars were not queried.\n")

	return b.String()
}

func writeRecordSection(b *strings.Builder, title string, records []MarkdownRecord, empty string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(records) == 0 {
		fmt.Fprintf(b, "- %s\n\n", empty)
		return
	}
	for _, record := range firstRecords(records, 8) {
		fmt.Fprintf(b, "- %s [%s, %s]\n", record.Title, record.Project, valueOr(record.Status, "unknown"))
	}
	fmt.Fprintf(b, "\n")
}

func sortRecordsForReview(records []MarkdownRecord) {
	sort.SliceStable(records, func(i int, j int) bool {
		left := records[i].EffectiveDate()
		right := records[j].EffectiveDate()
		if !left.Equal(right) {
			return left.After(right)
		}
		return records[i].ID < records[j].ID
	})
}

func firstRecords(records []MarkdownRecord, limit int) []MarkdownRecord {
	if len(records) <= limit {
		return records
	}
	return records[:limit]
}

func firstFriction(findings []FrictionFinding, limit int) []FrictionFinding {
	if len(findings) <= limit {
		return findings
	}
	return findings[:limit]
}
