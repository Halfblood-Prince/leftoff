package leftoff

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RememberRequest struct {
	Query   string
	Project string
	Limit   int
}

type RememberResult struct {
	Output       string
	Decisions    []DecisionMatch
	Solutions    []SolutionMatch
	Recipes      []RecipeMatch
	EvidenceGaps []string
}

type DecisionMatch struct {
	Record      MarkdownRecord
	Score       int
	Freshness   string
	StaleReason []string
}

type SolutionMatch struct {
	Record   MarkdownRecord
	Score    int
	Verified string
}

type RecipeMatch struct {
	Title string
	Path  string
	Text  string
	Score int
}

type SolutionCapturePlan struct {
	Output        string
	Saved         bool
	Similar       []SolutionMatch
	CaptureResult CaptureResult
	EvidenceGaps  []string
}

type SaveSolutionRequest struct {
	Project  string
	Problem  string
	Solution string
	Confirm  bool
}

func (s *Store) RememberWhy(req RememberRequest) (RememberResult, error) {
	if strings.TrimSpace(req.Query) == "" {
		req.Query = "recent decisions and solutions"
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}

	decisionTypes := map[RecordType]bool{RecordDecision: true}
	decisions, err := s.LoadRecords(RecordQuery{Project: req.Project, Types: decisionTypes})
	if err != nil {
		return RememberResult{}, err
	}
	solutionTypes := map[RecordType]bool{RecordProblem: true, RecordSolution: true}
	solutions, err := s.LoadRecords(RecordQuery{Project: req.Project, Types: solutionTypes})
	if err != nil {
		return RememberResult{}, err
	}
	recipes := s.SearchReusableRecipes(req.Query, req.Limit)

	var decisionMatches []DecisionMatch
	for _, record := range decisions {
		score := tokenOverlapScore(req.Query, record.Text())
		if score == 0 && !strings.Contains(strings.ToLower(record.Text()), strings.ToLower(req.Query)) {
			continue
		}
		freshness, reasons := decisionFreshness(record, req.Query, s.now())
		decisionMatches = append(decisionMatches, DecisionMatch{Record: record, Score: score, Freshness: freshness, StaleReason: reasons})
	}

	var solutionMatches []SolutionMatch
	for _, record := range solutions {
		score := tokenOverlapScore(req.Query, record.Text())
		if score == 0 && !strings.Contains(strings.ToLower(record.Text()), strings.ToLower(req.Query)) {
			continue
		}
		solutionMatches = append(solutionMatches, SolutionMatch{Record: record, Score: score, Verified: verifiedLabel(record)})
	}

	sort.SliceStable(decisionMatches, func(i int, j int) bool {
		if decisionMatches[i].Score != decisionMatches[j].Score {
			return decisionMatches[i].Score > decisionMatches[j].Score
		}
		return decisionMatches[i].Record.EffectiveDate().After(decisionMatches[j].Record.EffectiveDate())
	})
	sort.SliceStable(solutionMatches, func(i int, j int) bool {
		if solutionMatches[i].Score != solutionMatches[j].Score {
			return solutionMatches[i].Score > solutionMatches[j].Score
		}
		return solutionMatches[i].Record.EffectiveDate().After(solutionMatches[j].Record.EffectiveDate())
	})

	if len(decisionMatches) > req.Limit {
		decisionMatches = decisionMatches[:req.Limit]
	}
	if len(solutionMatches) > req.Limit {
		solutionMatches = solutionMatches[:req.Limit]
	}
	if len(recipes) > req.Limit {
		recipes = recipes[:req.Limit]
	}

	gaps := []string{}
	if len(decisionMatches) == 0 {
		gaps = append(gaps, "No matching decision record was found.")
	}
	if len(solutionMatches) == 0 {
		gaps = append(gaps, "No matching solved-problem record was found.")
	}
	if len(recipes) == 0 {
		gaps = append(gaps, "No matching reusable recipe was found.")
	}

	result := RememberResult{
		Decisions:    decisionMatches,
		Solutions:    solutionMatches,
		Recipes:      recipes,
		EvidenceGaps: gaps,
	}
	result.Output = renderRememberWhy(req, result)
	return result, nil
}

func decisionFreshness(record MarkdownRecord, query string, now time.Time) (string, []string) {
	var reasons []string
	status := NormalizeStatus(record.Status)
	if lowerContainsAny(status, "superseded", "stale", "rejected", "abandoned") {
		reasons = append(reasons, "decision status is "+record.Status)
	}
	revisit := record.Field("revisit when")
	if date := extractISODate(revisit); !date.IsZero() && !date.After(startOfDay(now)) {
		reasons = append(reasons, "listed revisit date has passed: "+date.Format("2006-01-02"))
	}
	if lowerContainsAny(query, "reconsider", "revisit", "stale") {
		reasons = append(reasons, "user query asks to reconsider or revisit")
	}
	if len(reasons) > 0 {
		return "needs review", reasons
	}
	if revisit == "" || revisit == "none recorded" {
		return "uncertain", []string{"no revisit condition is recorded"}
	}
	return "current from local record", nil
}

func verifiedLabel(record MarkdownRecord) string {
	verified := strings.ToLower(record.Field("verified"))
	switch {
	case verified == "yes" || verified == "true" || strings.Contains(verified, "verified"):
		return "verified"
	case verified == "":
		return "not marked verified"
	default:
		return record.Field("verified")
	}
}

func renderRememberWhy(req RememberRequest, result RememberResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "DECISIONS\n")
	if len(result.Decisions) == 0 {
		fmt.Fprintf(&b, "- No matching decision found.\n")
	} else {
		for _, match := range result.Decisions {
			record := match.Record
			fmt.Fprintf(&b, "- %s\n", record.Title)
			fmt.Fprintf(&b, "  Project: %s\n", record.Project)
			fmt.Fprintf(&b, "  Freshness: %s\n", match.Freshness)
			if decision := record.Field("decision"); decision != "" {
				fmt.Fprintf(&b, "  Decision: %s\n", decision)
			}
			if context := record.Field("context"); context != "" {
				fmt.Fprintf(&b, "  Context: %s\n", context)
			}
			if alternatives := record.Field("alternatives rejected"); alternatives != "" {
				fmt.Fprintf(&b, "  Alternatives rejected: %s\n", alternatives)
			}
			if evidence := record.Field("evidence"); evidence != "" {
				fmt.Fprintf(&b, "  Evidence: %s\n", evidence)
			}
			if revisit := record.Field("revisit when"); revisit != "" {
				fmt.Fprintf(&b, "  Revisit when: %s\n", revisit)
			}
			if len(match.StaleReason) > 0 {
				fmt.Fprintf(&b, "  Review signal: %s\n", strings.Join(match.StaleReason, "; "))
			}
		}
	}

	fmt.Fprintf(&b, "\nPROBLEMS AND SOLUTIONS\n")
	if len(result.Solutions) == 0 {
		fmt.Fprintf(&b, "- No matching problem or solution found.\n")
	} else {
		for _, match := range result.Solutions {
			record := match.Record
			fmt.Fprintf(&b, "- %s\n", record.Title)
			fmt.Fprintf(&b, "  Project: %s\n", record.Project)
			fmt.Fprintf(&b, "  Verified: %s\n", match.Verified)
			if problem := record.Field("problem"); problem != "" {
				fmt.Fprintf(&b, "  Problem: %s\n", problem)
			}
			if solution := record.Field("solution"); solution != "" {
				fmt.Fprintf(&b, "  Solution: %s\n", solution)
			}
			if linked := record.Field("linked problem"); linked != "" {
				fmt.Fprintf(&b, "  Linked problem: %s\n", linked)
			}
			if evidence := record.Field("evidence"); evidence != "" {
				fmt.Fprintf(&b, "  Evidence: %s\n", evidence)
			}
		}
	}

	fmt.Fprintf(&b, "\nREUSABLE RECIPES\n")
	if len(result.Recipes) == 0 {
		fmt.Fprintf(&b, "- No matching reusable recipe found.\n")
	} else {
		for _, recipe := range result.Recipes {
			fmt.Fprintf(&b, "- %s\n", recipe.Title)
			fmt.Fprintf(&b, "  Source: %s\n", recipe.Path)
			fmt.Fprintf(&b, "  Summary: %s\n", cleanSummary(recipe.Text, 220))
		}
	}

	fmt.Fprintf(&b, "\nEVIDENCE GAPS\n")
	if len(result.EvidenceGaps) == 0 {
		fmt.Fprintf(&b, "- Matching local records included rationale, evidence, and revisit fields where available.\n")
	} else {
		for _, gap := range result.EvidenceGaps {
			fmt.Fprintf(&b, "- %s\n", gap)
		}
	}
	fmt.Fprintf(&b, "- Remote issue, PR, and CI history were not queried.\n")

	return b.String()
}

func (s *Store) SearchReusableRecipes(query string, limit int) []RecipeMatch {
	if limit <= 0 {
		limit = 5
	}
	paths := []string{
		filepath.Join(s.Root, "patterns", "reusable-recipes.md"),
		filepath.Join(s.Root, "patterns", "recurring-friction.md"),
	}
	var matches []RecipeMatch
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, section := range splitMarkdownSections(string(content)) {
			score := tokenOverlapScore(query, section.Title+" "+section.Text)
			if score == 0 && query != "" {
				continue
			}
			matches = append(matches, RecipeMatch{Title: section.Title, Path: path, Text: section.Text, Score: score})
		}
	}
	sort.SliceStable(matches, func(i int, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

type markdownSection struct {
	Title string
	Text  string
}

func splitMarkdownSections(content string) []markdownSection {
	var sections []markdownSection
	var current markdownSection
	var body strings.Builder
	flush := func() {
		if current.Title == "" {
			return
		}
		current.Text = strings.TrimSpace(body.String())
		if current.Text != "" {
			sections = append(sections, current)
		}
		body.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = markdownSection{Title: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			continue
		}
		if current.Title != "" {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}
	flush()
	return sections
}

func (s *Store) SimilarSolutions(candidate string, project string, limit int) ([]SolutionMatch, error) {
	records, err := s.LoadRecords(RecordQuery{
		Project: project,
		Types: map[RecordType]bool{
			RecordSolution: true,
			RecordProblem:  true,
		},
	})
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	var matches []SolutionMatch
	for _, record := range records {
		text := record.Field("solution")
		if text == "" {
			text = record.PrimaryText()
		}
		similarity := jaccardSimilarity(candidate, text)
		if similarity >= 0.78 {
			matches = append(matches, SolutionMatch{
				Record:   record,
				Score:    int(similarity * 100),
				Verified: verifiedLabel(record),
			})
		}
	}
	sort.SliceStable(matches, func(i int, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Record.EffectiveDate().After(matches[j].Record.EffectiveDate())
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func (s *Store) SaveSolutionCandidate(ctx context.Context, req SaveSolutionRequest) (SolutionCapturePlan, error) {
	text := strings.TrimSpace(req.Solution)
	if text == "" {
		return SolutionCapturePlan{}, fmt.Errorf("solution text is empty")
	}
	if req.Problem != "" {
		text = text + " | Problem: " + strings.TrimSpace(req.Problem)
	}

	similar, err := s.SimilarSolutions(text, req.Project, 5)
	if err != nil {
		return SolutionCapturePlan{}, err
	}

	plan := SolutionCapturePlan{Similar: similar}
	var b strings.Builder
	fmt.Fprintf(&b, "SAVE SOLUTION\n")
	if len(similar) > 0 {
		fmt.Fprintf(&b, "- Similar saved solutions found. They are suggested for review, not merged automatically.\n")
		for _, match := range similar {
			fmt.Fprintf(&b, "  - %s [%s]\n", match.Record.Title, match.Verified)
		}
	} else {
		fmt.Fprintf(&b, "- No near-duplicate saved solution found.\n")
	}

	if !req.Confirm {
		fmt.Fprintf(&b, "- Not saved. Rerun with `--confirm` to explicitly persist this solution.\n")
		plan.Output = b.String()
		return plan, nil
	}

	result, err := s.Capture(ctx, CaptureRequest{
		Kind:    string(RecordSolution),
		Text:    text,
		Project: req.Project,
		Source:  "save-solution",
	})
	if err != nil {
		return plan, err
	}
	plan.Saved = true
	plan.CaptureResult = result
	fmt.Fprintf(&b, "- Saved %s to %s\n", result.ID, result.Path)
	fmt.Fprintf(&b, "- Evidence: Explicit save-solution confirmation.\n")
	plan.Output = b.String()
	return plan, nil
}
