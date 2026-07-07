package leftoff

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestNowRanksTimeFitReleaseBlocker(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "sample-app", Slug: "sample-app", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.AppendMarkdownSection(paths.OpenLoops, `## TASK-2026-07-06-001 - Add Windows installation smoke test

- Type: task
- Status: active
- Project: sample-app
- Priority: high
- Effort: 20-40 min
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Add Windows installation smoke test for the release.
- Evidence: User capture.
- Next action: Add a Windows job that runs the installer and checks help output.
- Blocked by: none`); err != nil {
		t.Fatal(err)
	}

	if err := store.AppendMarkdownSection(paths.OpenLoops, `## TASK-2026-07-06-002 - Rewrite dashboard layout

- Type: task
- Status: active
- Project: sample-app
- Priority: medium
- Effort: 2 hours
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Rewrite dashboard layout.
- Evidence: User capture.
- Next action: Sketch a new layout.
- Blocked by: none`); err != nil {
		t.Fatal(err)
	}

	if err := store.AppendMarkdownSection(paths.OpenLoops, `## TASK-2026-07-06-003 - Wait for package signing

- Type: task
- Status: waiting
- Project: sample-app
- Priority: high
- Effort: 15 min
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Wait for package signing before release.
- Evidence: User capture.
- Next action: Check the signing status.
- Blocked by: signing job`); err != nil {
		t.Fatal(err)
	}

	result, err := store.Now(NowRequest{Project: "sample-app", Minutes: 45, Focus: "release windows"})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Ready) == 0 {
		t.Fatalf("expected ready recommendations")
	}
	if result.Ready[0].Record.ID != "TASK-2026-07-06-001" {
		t.Fatalf("expected smoke test first, got %s", result.Ready[0].Record.ID)
	}
	if len(result.Parked) == 0 || result.Parked[0].Record.ID != "TASK-2026-07-06-003" {
		t.Fatalf("expected waiting task parked, got %#v", result.Parked)
	}
	assertContains(t, result.Output, "NOW")
	assertContains(t, result.Output, "WHY THIS ORDER")
	assertContains(t, result.Output, "Temporary inputs used")
}

func TestNowReportsMissingEffortAsEvidenceGap(t *testing.T) {
	store := fixedStore(t)
	paths, err := store.EnsureProject(ProjectMeta{Name: "sample", Slug: "sample", Created: store.now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMarkdownSection(paths.OpenLoops, `## TASK-2026-07-06-001 - Clarify task

- Type: task
- Status: active
- Project: sample
- Priority: unspecified
- Effort: unknown
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Clarify task.
- Evidence: User capture.
- Next action: Clarify the next concrete step.
- Blocked by: none`); err != nil {
		t.Fatal(err)
	}

	result, err := store.Now(NowRequest{Project: "sample", Minutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, result.Output, "Missing effort estimate")
	assertContains(t, result.Output, "Missing explicit priority")
}

func TestNowDoesNotTreatRecordIDDateAsDeadline(t *testing.T) {
	store := fixedStore(t)
	if _, err := store.Capture(context.Background(), CaptureRequest{
		Project: "sample",
		Text:    "task: Add release smoke test",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Now(NowRequest{Project: "sample", Minutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Ready) == 0 {
		t.Fatalf("expected ready recommendation")
	}
	if result.Ready[0].Record.ID != "TASK-2026-07-06-001" {
		t.Fatalf("unexpected recommendation: %s", result.Ready[0].Record.ID)
	}
	for _, reason := range result.Ready[0].Reasons {
		if strings.Contains(reason, "deadline") {
			t.Fatalf("record ID date was treated as a deadline: reasons=%v output=\n%s", result.Ready[0].Reasons, result.Output)
		}
	}
	for _, evidence := range result.Ready[0].Evidence {
		if strings.Contains(evidence, "due date") {
			t.Fatalf("record ID date produced due-date evidence: evidence=%v output=\n%s", result.Ready[0].Evidence, result.Output)
		}
	}
	if strings.Contains(result.Output, "deadline is close") || strings.Contains(result.Output, "A due date is within three days.") {
		t.Fatalf("output contained false deadline evidence:\n%s", result.Output)
	}
}

func TestNowDoesNotTreatHistoricalDateAsDeadline(t *testing.T) {
	store := fixedStore(t)
	if _, err := store.Capture(context.Background(), CaptureRequest{
		Project: "sample",
		Text:    "task: Investigate regression introduced on 2026-06-01",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Now(NowRequest{Project: "sample", Minutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Ready) == 0 {
		t.Fatalf("expected ready recommendation")
	}
	for _, reason := range result.Ready[0].Reasons {
		if strings.Contains(reason, "deadline") {
			t.Fatalf("historical date was treated as a deadline: reasons=%v output=\n%s", result.Ready[0].Reasons, result.Output)
		}
	}
	if strings.Contains(result.Output, "deadline has passed") {
		t.Fatalf("output contained false deadline evidence:\n%s", result.Output)
	}
}

func TestNowTreatsExplicitDeadlineTextAsDeadline(t *testing.T) {
	store := fixedStore(t)
	if _, err := store.Capture(context.Background(), CaptureRequest{
		Project: "sample",
		Text:    "task: Deadline: 2026-07-10 finish release smoke test",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Now(NowRequest{Project: "sample", Minutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Ready) == 0 {
		t.Fatalf("expected ready recommendation")
	}
	if !strings.Contains(strings.Join(result.Ready[0].Reasons, "\n"), "deadline") {
		t.Fatalf("expected explicit deadline reason, got reasons=%v output=\n%s", result.Ready[0].Reasons, result.Output)
	}
	assertContains(t, result.Output, "A due date is within")
}

func TestLoadPriorityWeightsReadsConfig(t *testing.T) {
	store := fixedStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(store.Root, "config.yml")
	if err := atomicWriteFile(config, []byte(`# leftoff config

priority_weights:
  urgency: high
  release_impact: low
  dependency_penalty: 4
`), 0o600); err != nil {
		t.Fatal(err)
	}
	weights := store.LoadPriorityWeights()
	if weights.Urgency != 3 {
		t.Fatalf("expected high urgency weight, got %d", weights.Urgency)
	}
	if weights.ReleaseImpact != 1 {
		t.Fatalf("expected low release weight, got %d", weights.ReleaseImpact)
	}
	if weights.DependencyPenalty != 4 {
		t.Fatalf("expected numeric dependency penalty, got %d", weights.DependencyPenalty)
	}
}
