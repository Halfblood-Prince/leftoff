package leftoff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRememberWhyMarksExpiredDecisionForReview(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "leftoff", Slug: "leftoff", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMarkdownSection(paths.Decisions, `## DECISION-2026-07-01-001 - Use Markdown and JSONL

- Type: decision
- Status: accepted
- Project: leftoff
- Date: 2026-07-01
- Decision: Use Markdown and JSONL for local records.
- Context: Users must inspect and edit records.
- Alternatives rejected: SQLite; hosted database
- Evidence: product principle 3.1
- Revisit when: 2026-07-05`); err != nil {
		t.Fatal(err)
	}

	result, err := store.RememberWhy(RememberRequest{Project: "leftoff", Query: "markdown records"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("expected one decision match, got %d", len(result.Decisions))
	}
	if result.Decisions[0].Freshness != "needs review" {
		t.Fatalf("expected stale decision, got %s", result.Decisions[0].Freshness)
	}
	assertContains(t, result.Output, "Alternatives rejected: SQLite; hosted database")
	assertContains(t, result.Output, "listed revisit date has passed")
}

func TestSaveSolutionRequiresConfirmationAndSuggestsSimilar(t *testing.T) {
	store := fixedStore(t)
	if _, err := store.Capture(context.Background(), CaptureRequest{
		Project: "sample-app",
		Text:    "solution: Docker requires a running daemon before build commands work",
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := store.SaveSolutionCandidate(context.Background(), SaveSolutionRequest{
		Project:  "sample-app",
		Problem:  "Docker build failed",
		Solution: "Docker requires a running daemon before build commands work",
		Confirm:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Saved {
		t.Fatalf("solution was saved without confirmation")
	}
	if len(plan.Similar) == 0 {
		t.Fatalf("expected similar saved solution")
	}
	assertContains(t, plan.Output, "Not saved")
}

func TestFrictionRequiresRecurringEvidence(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "sample", Slug: "sample", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMarkdownSection(paths.Friction, `## FRICTION-2026-07-06-001 - Docker setup failed in runtime

- Type: friction_event
- Status: observed
- Project: sample
- Created: 2026-07-06
- Last touched: 2026-07-06
- Observation: Docker setup failed in runtime.
- Evidence: User capture.
- Impact: not estimated`); err != nil {
		t.Fatal(err)
	}

	result, err := store.Friction(FrictionRequest{Project: "sample"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("single event should not become recurring friction")
	}

	if err := store.AppendMarkdownSection(paths.Friction, `## FRICTION-2026-07-06-002 - Docker setup failed in runtime again

- Type: friction_event
- Status: observed
- Project: sample
- Created: 2026-07-06
- Last touched: 2026-07-06
- Observation: Docker setup failed in runtime.
- Evidence: User capture.
- Impact: not estimated`); err != nil {
		t.Fatal(err)
	}

	result, err = store.Friction(FrictionRequest{Project: "sample"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatalf("expected recurring friction")
	}
	assertContains(t, result.Output, "Smallest countermeasure")
}

func TestReviewWeekWritesReport(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "sample", Slug: "sample", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMarkdownSection(paths.OpenLoops, `## TASK-2026-07-06-001 - Ship docs

- Type: task
- Status: done
- Project: sample
- Priority: high
- Effort: 30 min
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Ship docs.
- Evidence: User capture.
- Next action: none
- Blocked by: none`); err != nil {
		t.Fatal(err)
	}

	result, err := store.ReviewWeek(ReviewWeekRequest{Project: "sample", Week: "2026-W28", Write: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Path == "" {
		t.Fatalf("expected written report path")
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("expected report to exist: %v", err)
	}
	content := readFile(t, result.Path)
	assertContains(t, content, "## Shipped")
	assertContains(t, content, "Ship docs")
	assertContains(t, content, "Activity note")
	if strings.Contains(content, filepath.Clean(store.Root)+string(os.PathSeparator)+"projects"+string(os.PathSeparator)+"sample"+string(os.PathSeparator)+"open-loops.md") {
		t.Fatalf("review leaked internal file path in report")
	}
}
