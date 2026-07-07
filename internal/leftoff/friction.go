package leftoff

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type FrictionRequest struct {
	Project string
}

type FrictionResult struct {
	Output   string
	Findings []FrictionFinding
}

type FrictionFinding struct {
	Pattern         string
	Observations    []string
	Impact          string
	LikelyRootCause string
	Countermeasure  string
	Confidence      string
}

func (s *Store) Friction(req FrictionRequest) (FrictionResult, error) {
	records, err := s.LoadRecords(RecordQuery{Project: req.Project, IncludeInbox: req.Project == ""})
	if err != nil {
		return FrictionResult{}, err
	}
	events, err := s.LoadAllActivities(req.Project)
	if err != nil {
		return FrictionResult{}, err
	}

	findings := DetectFriction(records, events, s.now())
	result := FrictionResult{Findings: findings}
	result.Output = renderFriction(result)
	return result, nil
}

func DetectFriction(records []MarkdownRecord, events []ActivityEvent, now time.Time) []FrictionFinding {
	groups := map[string][]string{}
	add := func(pattern string, observation string) {
		pattern = strings.TrimSpace(pattern)
		observation = strings.TrimSpace(observation)
		if pattern == "" || observation == "" {
			return
		}
		groups[pattern] = append(groups[pattern], observation)
	}

	stalled := []string{}
	blocked := []string{}
	for _, record := range records {
		status := NormalizeStatus(record.Status)
		text := record.Text()
		switch record.Type {
		case RecordFrictionEvent:
			add(normalizeFrictionPattern(record.PrimaryText()), frictionRecordObservation(record))
		case RecordProblem:
			if lowerContainsAny(text, "failed", "failure", "error", "conflict", "unavailable", "timeout") {
				add("repeated problem: "+normalizeFrictionPattern(record.PrimaryText()), frictionRecordObservation(record))
			}
		case RecordTask, RecordOpenLoop:
			if status == string(StatusBlocked) || status == string(StatusWaiting) {
				blocked = append(blocked, fmt.Sprintf("%s: %s", record.Project, record.Title))
			}
			if status == string(StatusActive) || status == string(StatusInbox) {
				touched := record.EffectiveDate()
				if !touched.IsZero() && startOfDay(now).Sub(startOfDay(touched)) >= 14*24*time.Hour {
					stalled = append(stalled, fmt.Sprintf("%s: %s", record.Project, record.Title))
				}
			}
		}
	}

	if len(blocked) >= 2 {
		groups["blocked or waiting tasks"] = append(groups["blocked or waiting tasks"], blocked...)
	}
	if len(stalled) >= 2 {
		groups["stalled active tasks"] = append(groups["stalled active tasks"], stalled...)
	}

	captureBySummary := map[string][]string{}
	for _, event := range events {
		if event.Kind != "capture" {
			continue
		}
		key := normalizeFrictionPattern(event.Summary)
		if key == "" {
			continue
		}
		captureBySummary[key] = append(captureBySummary[key], frictionActivityObservation(event))
	}
	for key, observations := range captureBySummary {
		if len(observations) >= 2 && lowerContainsAny(key, "failed", "failure", "error", "blocked", "setup", "conflict") {
			groups["repeated capture: "+key] = append(groups["repeated capture: "+key], observations...)
		}
	}

	var findings []FrictionFinding
	for pattern, observations := range groups {
		observations = uniqueStrings(observations)
		if len(observations) < 2 {
			continue
		}
		findings = append(findings, buildFrictionFinding(pattern, observations))
	}

	sort.SliceStable(findings, func(i int, j int) bool {
		if len(findings[i].Observations) != len(findings[j].Observations) {
			return len(findings[i].Observations) > len(findings[j].Observations)
		}
		return findings[i].Pattern < findings[j].Pattern
	})
	return findings
}

func frictionRecordObservation(record MarkdownRecord) string {
	return fmt.Sprintf("%s %s: %s", record.ID, valueOr(record.Project, "unknown"), record.PrimaryText())
}

func frictionActivityObservation(event ActivityEvent) string {
	label := strings.TrimSpace(event.RecordID)
	if label == "" {
		label = strings.TrimSpace(event.Timestamp)
	}
	if label == "" {
		label = "activity"
	}
	return fmt.Sprintf("%s %s: %s", label, valueOr(event.Project, "unknown"), event.Summary)
}

func normalizeFrictionPattern(text string) string {
	tokens := sortedTokens(text)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) > 6 {
		tokens = tokens[:6]
	}
	return strings.Join(tokens, " ")
}

func buildFrictionFinding(pattern string, observations []string) FrictionFinding {
	impact := "medium"
	confidence := "medium"
	if len(observations) >= 4 {
		impact = "high"
		confidence = "high"
	}

	root := "The same kind of work is being revisited without a recorded countermeasure."
	counter := "Capture the smallest reusable recipe or next diagnostic command after the next occurrence."
	if strings.Contains(pattern, "blocked") || strings.Contains(pattern, "waiting") {
		root = "Several tasks are waiting on unresolved dependencies or unclear ownership."
		counter = "Pick one blocked task and record the exact owner, dependency, or unblock condition."
	}
	if strings.Contains(pattern, "stalled") {
		root = "Active tasks are aging without a fresh next action."
		counter = "Review the oldest stalled task and either add a concrete next action, park it, or abandon it."
	}
	if lowerContainsAny(pattern, "setup", "install", "dependency", "conflict") {
		root = "Setup or dependency steps are not yet captured as a reusable path."
		counter = "Write a short reusable recipe with the verified command sequence and known failure mode."
	}

	return FrictionFinding{
		Pattern:         pattern,
		Observations:    observations,
		Impact:          impact,
		LikelyRootCause: root,
		Countermeasure:  counter,
		Confidence:      confidence,
	}
}

func renderFriction(result FrictionResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "FRICTION\n")
	if len(result.Findings) == 0 {
		fmt.Fprintf(&b, "- No recurring friction pattern found from local records.\n")
		fmt.Fprintf(&b, "- A single event is not treated as recurring friction.\n")
		fmt.Fprintf(&b, "- Evidence gap: local records may be sparse or not yet tagged as problems/friction.\n")
		return b.String()
	}

	for _, finding := range result.Findings {
		fmt.Fprintf(&b, "- Pattern: %s\n", finding.Pattern)
		fmt.Fprintf(&b, "  Impact: %s\n", finding.Impact)
		fmt.Fprintf(&b, "  Confidence: %s\n", finding.Confidence)
		fmt.Fprintf(&b, "  Supporting observations:\n")
		for _, observation := range firstStrings(finding.Observations, 5) {
			fmt.Fprintf(&b, "    - %s\n", observation)
		}
		fmt.Fprintf(&b, "  Likely root cause: %s\n", finding.LikelyRootCause)
		fmt.Fprintf(&b, "  Smallest countermeasure: %s\n", finding.Countermeasure)
	}
	return b.String()
}
