# Examples

## Capture a Task

```sh
./bin/leftoff capture --project sample-app "task: Add Windows installation smoke test before v1.0"
```

Expected result:

```text
Saved TASK-2026-07-06-001 (task)
Destination: ~/.leftoff/projects/sample-app/open-loops.md
Evidence: User capture.
```

## Capture a Decision

```sh
./bin/leftoff capture --project leftoff "decision: Use Markdown plus JSONL instead of SQLite because records must stay editable"
```

## Save Local Git State

```sh
./bin/leftoff scan --repo .
```

The scan stores compact metadata only. It does not read file contents or full diffs.

## Resume Work

```sh
./bin/leftoff resume --repo .
```

Resume combines saved project state, current read-only Git metadata, recent open loops, recent decisions, solved problems, and activity events.

## Pick Work for 45 Minutes

```sh
./bin/leftoff now --minutes 45 --focus release
```

The output explains the recommended task, alternatives, parked items, ordering rationale, and evidence gaps.

Use structured output when another agent will parse the recommendation:

```sh
./bin/leftoff now --minutes 45 --focus release --json
```

## Track a Workspace

```sh
./bin/leftoff workspace add .
./bin/leftoff workspace scan
./bin/leftoff now --all --minutes 45 --json
```

Workspace scan stores safe metadata only: dirty state, branch, ahead/behind status, unpushed commit count, redacted recent commit titles, stale branch names, worktree status, and saved leftoff records.

## Recall a Decision

```sh
./bin/leftoff remember-why "storage format"
```

The output includes decision rationale, alternatives rejected, evidence, and revisit signals where recorded.

## Weekly Review

```sh
./bin/leftoff review-week --write
```

This writes `weekly/YYYY-Www.md` inside the leftoff store and prints the report.

## Cleanup Advisor

```sh
./bin/leftoff clean-up --repo .
```

Cleanup is report-only by default. It may show Git command previews, but it does not delete branches or worktrees.

## Optional GitHub Metadata

```sh
./bin/leftoff github --repo . --refresh
```

This is opt-in and requires `gh`. Cached metadata can be removed with:

```sh
./bin/leftoff github --project leftoff --forget-cache
```

## Export and Delete

```sh
./bin/leftoff export --out leftoff-export.zip
./bin/leftoff delete-data --dry-run
```
