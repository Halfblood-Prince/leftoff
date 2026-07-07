# Contributing

`leftoff` is a local-first skill and CLI for unfinished developer work. Contributions should preserve the product promise: useful local evidence without surveillance or destructive defaults.

## Rules

- Do not add third-party dependencies without a concrete tested reason.
- Do not add hidden network access.
- Do not persist secrets, source contents, full diffs, or unredacted command output.
- Add or update tests for behavior changes.
- Keep output grounded in evidence, inference, and uncertainty.
- Preserve editable Markdown and append-only JSONL formats.

## Development

```sh
cd plugins/leftoff
gofmt -w cmd internal
go test ./...
```

Use `--store <temp-path>` for manual testing.
