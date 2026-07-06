# Data Format and Migration Policy

Current data format version: `1`.

The store root contains a marker file:

```text
.leftoff-store
```

The marker identifies a directory as a leftoff store and is required by guarded deletion.

## Stability

The v1 format is Markdown-first:

- user-editable Markdown records;
- append-only JSONL activity events;
- JSON cache files for optional integrations.

Record IDs are stable and human-readable. Existing Markdown content should not be reformatted by later commands.

## Migration Policy

Future migrations must:

- create a backup before rewriting files;
- preserve user-authored Markdown where possible;
- report exactly what changed;
- be runnable without network access;
- never delete source data as part of migration.
