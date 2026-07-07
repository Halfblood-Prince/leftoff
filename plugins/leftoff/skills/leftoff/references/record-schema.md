# Record Schema

The default store root is `~/.leftoff/`.

```text
~/.leftoff/
|-- profile.md
|-- config.yml
|-- inbox.md
|-- projects/
|   `-- <project-slug>/
|       |-- project.md
|       |-- state.md
|       |-- decisions.md
|       |-- open-loops.md
|       |-- solved-problems.md
|       |-- releases.md
|       |-- friction.md
|       `-- activity.jsonl
|-- patterns/
|   |-- recurring-friction.md
|   `-- reusable-recipes.md
|-- weekly/
|-- workspace/
|   `-- repos.json
|-- cache/
|   |-- local-scan-metadata.json
|   |-- workspace-scan.json
|   `-- github/<project-slug>.json
`-- backups/
```

The store root also contains `.leftoff-store`, a small marker used by guarded delete-data.

## Markdown Records

Each durable Markdown record starts with a stable ID:

```markdown
## TASK-2026-07-06-001 - Add Windows installation smoke test

- Type: task
- Status: active
- Project: sample-app
- Created: 2026-07-06
- Last touched: 2026-07-06
- Summary: Add Windows installation smoke test.
- Evidence: User capture.
- Next action: Clarify the next concrete step.
- Blocked by: none recorded
```

## Activity Events

Activity events are append-only JSONL objects with minimal metadata:

```json
{"timestamp":"2026-07-06T13:00:00+02:00","kind":"capture","record_id":"TASK-2026-07-06-001","record_type":"task","project":"sample-app","summary":"Add Windows installation smoke test","evidence":"User capture"}
```

## Record Types

The initial record types are:

- `task`
- `idea`
- `decision`
- `problem`
- `solution`
- `open_loop`
- `release_intent`
- `friction_event`
- `activity_event`

Task statuses are limited to:

```text
inbox | active | blocked | waiting | parked | done | abandoned
```
