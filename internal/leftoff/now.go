package leftoff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type NowRequest struct {
	Project string `json:"project,omitempty"`
	Focus   string `json:"focus,omitempty"`
	Minutes int    `json:"minutes,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	All     bool   `json:"all,omitempty"`
}

type NowResult struct {
	Output          string       `json:"output"`
	Ready           []ScoredTask `json:"ready"`
	Parked          []ScoredTask `json:"parked"`
	EvidenceGaps    []string     `json:"evidence_gaps,omitempty"`
	TemporaryInputs []string     `json:"temporary_inputs,omitempty"`
}

type ScoredTask struct {
	Record       MarkdownRecord `json:"record"`
	Score        int            `json:"score"`
	Band         string         `json:"band"`
	Confidence   string         `json:"confidence"`
	Reasons      []string       `json:"reasons,omitempty"`
	Evidence     []string       `json:"evidence,omitempty"`
	Gaps         []string       `json:"gaps,omitempty"`
	ParkedReason string         `json:"parked_reason,omitempty"`
	Effort       string         `json:"effort"`
}

type PriorityWeights struct {
	Urgency                int
	BlockerResolutionValue int
	ReleaseImpact          int
	Recency                int
	UserFocusMatch         int
	FitForAvailableTime    int
	UncertaintyPenalty     int
	DependencyPenalty      int
}

func DefaultPriorityWeights() PriorityWeights {
	return PriorityWeights{
		Urgency:                2,
		BlockerResolutionValue: 2,
		ReleaseImpact:          2,
		Recency:                1,
		UserFocusMatch:         2,
		FitForAvailableTime:    2,
		UncertaintyPenalty:     2,
		DependencyPenalty:      2,
	}
}

func (s *Store) Now(req NowRequest) (NowResult, error) {
	project := req.Project
	if req.All {
		project = ""
	}
	records, err := s.LoadRecords(RecordQuery{
		Project:      project,
		IncludeInbox: strings.TrimSpace(project) == "",
	})
	if err != nil {
		return NowResult{}, err
	}

	weights := s.LoadPriorityWeights()
	var ready []ScoredTask
	var parked []ScoredTask
	var allGaps []string

	for _, record := range records {
		if !isTaskLike(record) {
			continue
		}
		status := NormalizeStatus(record.Status)
		if status == string(StatusDone) || status == "done" || status == "closed" || status == "completed" || status == string(StatusAbandoned) {
			continue
		}
		scored := scoreTask(record, req, weights, s.now())
		allGaps = append(allGaps, scored.Gaps...)
		if isParkedStatus(status) {
			parked = append(parked, scored)
		} else {
			ready = append(ready, scored)
		}
	}

	sortScoredTasks(ready)
	sortScoredTasks(parked)

	temporary := []string{}
	if req.Minutes > 0 {
		temporary = append(temporary, fmt.Sprintf("available time: %d minutes", req.Minutes))
	}
	if strings.TrimSpace(req.Focus) != "" {
		temporary = append(temporary, "focus: "+strings.TrimSpace(req.Focus))
	}
	if req.All {
		temporary = append(temporary, "scope: all projects and registered workspace repositories")
	}

	result := NowResult{
		Ready:           ready,
		Parked:          parked,
		EvidenceGaps:    uniqueStrings(allGaps),
		TemporaryInputs: temporary,
	}
	result.Output = renderNowResult(req, result)
	return result, nil
}

func isTaskLike(record MarkdownRecord) bool {
	switch record.Type {
	case RecordTask, RecordOpenLoop, RecordReleaseIntent, RecordIdea:
		return true
	default:
		return false
	}
}

func isParkedStatus(status string) bool {
	switch NormalizeStatus(status) {
	case string(StatusBlocked), string(StatusWaiting), string(StatusParked):
		return true
	default:
		return false
	}
}

func (s *Store) LoadPriorityWeights() PriorityWeights {
	weights := DefaultPriorityWeights()
	content, err := os.ReadFile(filepath.Join(s.Root, "config.yml"))
	if err != nil {
		return weights
	}

	inSection := false
	for _, line := range strings.Split(string(content), "\n") {
		raw := line
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "priority_weights:" {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if !strings.HasPrefix(raw, " ") && strings.Contains(line, ":") {
			inSection = false
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := parseWeightValue(strings.TrimSpace(parts[1]))
		switch key {
		case "urgency":
			weights.Urgency = value
		case "blocker_resolution_value":
			weights.BlockerResolutionValue = value
		case "release_impact":
			weights.ReleaseImpact = value
		case "recency":
			weights.Recency = value
		case "user_focus_match":
			weights.UserFocusMatch = value
		case "fit_for_available_time":
			weights.FitForAvailableTime = value
		case "uncertainty_penalty":
			weights.UncertaintyPenalty = value
		case "dependency_penalty":
			weights.DependencyPenalty = value
		}
	}
	return weights
}

func parseWeightValue(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "off", "none", "zero":
		return 0
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 && parsed <= 5 {
			return parsed
		}
		return 2
	}
}

func scoreTask(record MarkdownRecord, req NowRequest, weights PriorityWeights, now time.Time) ScoredTask {
	scored := ScoredTask{
		Record:     record,
		Confidence: "medium",
		Effort:     valueOr(record.Field("effort"), "unknown"),
	}
	text := record.Text()
	status := NormalizeStatus(record.Status)

	add := func(points int, reason string, evidence string) {
		if points == 0 {
			return
		}
		scored.Score += points
		scored.Reasons = append(scored.Reasons, reason)
		if evidence != "" {
			scored.Evidence = append(scored.Evidence, evidence)
		}
	}
	subtract := func(points int, reason string, gap string) {
		if points == 0 {
			return
		}
		scored.Score -= points
		scored.Reasons = append(scored.Reasons, reason)
		if gap != "" {
			scored.Gaps = append(scored.Gaps, gap)
		}
	}

	priority := strings.ToLower(record.Field("priority"))
	switch priority {
	case "high", "urgent":
		add(weights.Urgency*2, "high urgency", "Priority is high.")
	case "medium", "normal":
		add(weights.Urgency, "some urgency", "Priority is medium.")
	case "low":
		add(weights.Urgency/2, "low explicit priority", "Priority is low.")
	case "", "unspecified":
		subtract(weights.UncertaintyPenalty/2, "priority is unclear", "Missing explicit priority.")
	}
	if lowerContainsAny(text, "urgent", "deadline", "due", "release blocker") {
		add(weights.Urgency, "urgency keyword present", "Record text mentions urgency, deadline, or release blocker.")
	}

	if due := dueDate(record); !due.IsZero() {
		days := int(due.Sub(startOfDay(now)).Hours() / 24)
		switch {
		case days < 0:
			add(weights.Urgency*2, "deadline has passed", "A due date is before today.")
		case days <= 3:
			add(weights.Urgency*2, "deadline is close", "A due date is within three days.")
		case days <= 14:
			add(weights.Urgency, "deadline is upcoming", "A due date is within two weeks.")
		}
	}

	if lowerContainsAny(text, "unblock", "blocker", "release blocker") {
		add(weights.BlockerResolutionValue*2, "could remove a blocker", "Record text mentions a blocker or unblock path.")
	}

	if record.Type == RecordReleaseIntent || lowerContainsAny(text, "release", "ship", "v1.0", "version") {
		add(weights.ReleaseImpact*2, "release impact", "Record is a release intent or mentions shipping/release work.")
	}

	if touched := record.EffectiveDate(); !touched.IsZero() {
		ageDays := int(startOfDay(now).Sub(startOfDay(touched)).Hours() / 24)
		switch {
		case ageDays <= 7:
			add(weights.Recency*2, "touched this week", "Last touched or created within seven days.")
		case ageDays <= 30:
			add(weights.Recency, "touched this month", "Last touched or created within thirty days.")
		}
	} else {
		subtract(weights.UncertaintyPenalty/2, "recency unknown", "Missing created or last-touched date.")
	}

	if focus := strings.TrimSpace(req.Focus); focus != "" {
		overlap := tokenOverlapScore(focus, text)
		if overlap > 0 {
			add(weights.UserFocusMatch*min(overlap, 2), "matches requested focus", "Temporary focus terms matched this record.")
		} else {
			subtract(weights.UserFocusMatch/2, "does not match requested focus", "Temporary focus did not match this record.")
		}
	}

	if req.Minutes > 0 {
		minEffort, maxEffort, ok := parseEffortMinutes(record.Field("effort"))
		switch {
		case ok && maxEffort <= req.Minutes:
			add(weights.FitForAvailableTime*2, "fits the time budget", fmt.Sprintf("Effort %d-%d min fits %d min.", minEffort, maxEffort, req.Minutes))
		case ok && minEffort <= req.Minutes:
			add(weights.FitForAvailableTime, "partly fits the time budget", fmt.Sprintf("Effort starts at %d min for a %d min budget.", minEffort, req.Minutes))
			scored.Gaps = append(scored.Gaps, "Task may not finish inside the requested time budget.")
		case ok:
			subtract(weights.FitForAvailableTime*2, "too large for the time budget", fmt.Sprintf("Effort %d-%d min exceeds %d min.", minEffort, maxEffort, req.Minutes))
		default:
			subtract(weights.UncertaintyPenalty, "fit for time budget is unknown", "Missing effort estimate.")
		}
	}

	nextAction := strings.ToLower(record.Field("next action"))
	if nextAction == "" || strings.Contains(nextAction, "clarify the next concrete step") {
		subtract(weights.UncertaintyPenalty/2, "next action is weak", "Missing specific next action.")
	}

	blockedBy := strings.ToLower(record.Field("blocked by"))
	if status == string(StatusBlocked) || status == string(StatusWaiting) {
		subtract(weights.DependencyPenalty*2, "not currently ready", "Status is "+status+".")
		scored.ParkedReason = "Status is " + status + "."
	}
	if blockedBy != "" && blockedBy != "none" && blockedBy != "none recorded" {
		subtract(weights.DependencyPenalty*2, "has an unresolved dependency", "Blocked by: "+record.Field("blocked by"))
		scored.ParkedReason = "Blocked by: " + record.Field("blocked by")
	}
	if status == string(StatusParked) {
		scored.ParkedReason = "Status is parked."
	}

	scored.Band = scoreBand(scored.Score)
	scored.Confidence = confidenceFor(scored)
	scored.Reasons = uniqueStrings(scored.Reasons)
	scored.Evidence = uniqueStrings(scored.Evidence)
	scored.Gaps = uniqueStrings(scored.Gaps)
	return scored
}

func dueDate(record MarkdownRecord) time.Time {
	for _, field := range []string{"deadline", "due", "due date"} {
		if value := record.Field(field); value != "" {
			if parsed := extractISODate(value); !parsed.IsZero() {
				return parsed
			}
		}
	}
	return extractDeadlineDate(recordDeadlineSearchText(record))
}

func recordDeadlineSearchText(record MarkdownRecord) string {
	var parts []string
	if record.Title != "" {
		parts = append(parts, record.Title)
	}

	for _, field := range []string{
		"summary",
		"next action",
		"decision",
		"problem",
		"solution",
		"intent",
		"observation",
		"notes",
	} {
		if value := record.Field(field); value != "" {
			parts = append(parts, value)
		}
	}

	for _, line := range strings.Split(record.Body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, _, ok := parseBulletField(line); ok {
			continue
		}
		parts = append(parts, line)
	}

	return strings.Join(parts, " ")
}

func startOfDay(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func scoreBand(score int) string {
	switch {
	case score >= 8:
		return "high"
	case score >= 4:
		return "medium"
	default:
		return "low"
	}
}

func confidenceFor(task ScoredTask) string {
	gaps := len(task.Gaps)
	evidence := len(task.Evidence)
	switch {
	case gaps == 0 && evidence >= 3:
		return "high"
	case gaps >= 3:
		return "low"
	default:
		return "medium"
	}
}

func sortScoredTasks(tasks []ScoredTask) {
	sort.SliceStable(tasks, func(i int, j int) bool {
		if tasks[i].Score != tasks[j].Score {
			return tasks[i].Score > tasks[j].Score
		}
		left := tasks[i].Record.EffectiveDate()
		right := tasks[j].Record.EffectiveDate()
		if !left.Equal(right) {
			return left.After(right)
		}
		return tasks[i].Record.ID < tasks[j].Record.ID
	})
}

func renderNowResult(req NowRequest, result NowResult) string {
	var b strings.Builder
	nextLimit := req.Limit
	if nextLimit <= 0 {
		nextLimit = 2
	}
	if nextLimit > 3 {
		nextLimit = 3
	}

	fmt.Fprintf(&b, "NOW\n")
	if len(result.Ready) == 0 {
		fmt.Fprintf(&b, "- No ready task has enough local evidence to recommend confidently.\n")
	} else {
		writeTaskRecommendation(&b, result.Ready[0], true)
	}

	fmt.Fprintf(&b, "\nNEXT\n")
	if len(result.Ready) <= 1 {
		fmt.Fprintf(&b, "- No additional ready task found.\n")
	} else {
		count := min(len(result.Ready)-1, nextLimit)
		for i := 0; i < count; i++ {
			writeTaskRecommendation(&b, result.Ready[i+1], false)
		}
	}

	fmt.Fprintf(&b, "\nPARKED\n")
	if len(result.Parked) == 0 {
		fmt.Fprintf(&b, "- No blocked, waiting, or parked task found.\n")
	} else {
		count := min(len(result.Parked), 5)
		for i := 0; i < count; i++ {
			task := result.Parked[i]
			reason := valueOr(task.ParkedReason, "Not ready.")
			fmt.Fprintf(&b, "- %s [%s, confidence %s] - %s\n", task.Record.Title, task.Band, task.Confidence, reason)
		}
	}

	fmt.Fprintf(&b, "\nWHY THIS ORDER\n")
	if len(result.TemporaryInputs) > 0 {
		fmt.Fprintf(&b, "- Temporary inputs used for this ranking: %s. These were not persisted.\n", strings.Join(result.TemporaryInputs, "; "))
	}
	if len(result.Ready) > 0 {
		top := result.Ready[0]
		fmt.Fprintf(&b, "- Top recommendation scored %s because: %s\n", top.Band, valueOr(strings.Join(firstStrings(top.Reasons, 3), "; "), "local evidence was stronger than alternatives"))
	}
	if len(result.Ready) > 1 {
		fmt.Fprintf(&b, "- Alternatives ranked lower because they had weaker fit, weaker urgency, more uncertainty, or less direct focus match.\n")
	}
	if req.All {
		fmt.Fprintf(&b, "- `--all` ranks local Markdown records from every project and inbox; run `workspace scan` first to refresh registered repository state.\n")
	}
	fmt.Fprintf(&b, "- Ranking uses local Markdown records only; remote PRs, CI, calendars, and issue trackers were not queried.\n")

	fmt.Fprintf(&b, "\nEVIDENCE GAPS\n")
	if len(result.EvidenceGaps) == 0 {
		fmt.Fprintf(&b, "- No major evidence gaps found in the ranked task records.\n")
	} else {
		for _, gap := range firstStrings(result.EvidenceGaps, 8) {
			fmt.Fprintf(&b, "- %s\n", gap)
		}
	}

	return b.String()
}

func writeTaskRecommendation(b *strings.Builder, task ScoredTask, primary bool) {
	prefix := "-"
	if primary {
		prefix = "- Recommended:"
	}
	fmt.Fprintf(b, "%s %s\n", prefix, task.Record.Title)
	fmt.Fprintf(b, "  Project: %s\n", task.Record.Project)
	fmt.Fprintf(b, "  Score band: %s\n", task.Band)
	fmt.Fprintf(b, "  Confidence: %s\n", task.Confidence)
	fmt.Fprintf(b, "  Effort: %s\n", task.Effort)
	if next := task.Record.Field("next action"); next != "" {
		fmt.Fprintf(b, "  Next action: %s\n", next)
	}
	if len(task.Reasons) > 0 {
		fmt.Fprintf(b, "  Why: %s\n", strings.Join(firstStrings(task.Reasons, 3), "; "))
	}
	if len(task.Evidence) > 0 {
		fmt.Fprintf(b, "  Evidence: %s\n", strings.Join(firstStrings(task.Evidence, 3), "; "))
	}
	if len(task.Gaps) > 0 {
		fmt.Fprintf(b, "  Uncertainty: %s\n", strings.Join(firstStrings(task.Gaps, 2), "; "))
	}
}

func firstStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
