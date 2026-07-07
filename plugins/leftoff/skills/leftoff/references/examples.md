# Examples

## Capture a Task

```sh
./scripts/leftoff capture --project sample-app "task: Add Windows installation smoke test before v1.0"
```

Expected result:

```text
Saved TASK-2026-07-06-001 (task)
Destination: ~/.leftoff/projects/sample-app/open-loops.md
Evidence: User capture.
```

## Capture a Decision

```sh
./scripts/leftoff capture --project leftoff "decision: Use Markdown plus JSONL instead of SQLite because records must stay editable"
```

## Save Local Git State

```sh
./scripts/leftoff scan --repo .
```

The scan stores compact metadata only. It does not read file contents or full diffs.

## Resume Work

```sh
./scripts/leftoff resume --repo .
```

Resume combines saved project state, current read-only Git metadata, recent open loops, recent decisions, solved problems, and activity events.

## Pick Work for 45 Minutes

```sh
./scripts/leftoff now --minutes 45 --focus release
```

The output explains the recommended task, alternatives, parked items, ordering rationale, and evidence gaps.

Use structured output when another agent will parse the recommendation:

```sh
./scripts/leftoff now --minutes 45 --focus release --json
```

## Track a Workspace

```sh
./scripts/leftoff workspace add .
./scripts/leftoff workspace scan
./scripts/leftoff now --all --minutes 45 --json
```

Workspace scan stores safe metadata only: dirty state, branch, ahead/behind status, unpushed commit count, redacted recent commit titles, stale branch names, worktree status, and saved leftoff records.

## Recall a Decision

```sh
./scripts/leftoff remember-why "storage format"
```

The output includes decision rationale, alternatives rejected, evidence, and revisit signals where recorded.

## Weekly Review

```sh
./scripts/leftoff review-week --write
```

This writes `weekly/YYYY-Www.md` inside the leftoff store and prints the report.

## Cleanup Advisor

```sh
./scripts/leftoff clean-up --repo .
```

Cleanup is report-only by default. It may show Git command previews, but it does not delete branches or worktrees.

## Optional GitHub Metadata

```sh
./scripts/leftoff github --repo . --refresh
```

This is opt-in and requires `gh`. Cached metadata can be removed with:

```sh
./scripts/leftoff github --project leftoff --forget-cache
```

## Export and Delete

```sh
./scripts/leftoff export --out leftoff-export.zip
./scripts/leftoff delete-data --dry-run
```
