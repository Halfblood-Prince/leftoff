# leftoff v1.0.0 Release Notes

`leftoff` v1.0.0 is the first trust-focused release of the local-first skill and CLI.

## Highlights

- Plain Markdown and JSONL store under `~/.leftoff/`.
- Shared `plugins/leftoff/skills/leftoff/SKILL.md` contract for AI agents.
- Dedicated `plugins/leftoff/` Claude and Codex plugin package with repo-local marketplace catalogues.
- GitHub Agent Skills support through `gh skill` discovery of the plugin-contained skill.
- Adapter notes and installer aliases with evidence-based status categories in `plugins/leftoff/agents/supported.md`.
- CI, CodeQL, fuzzing, and annotated-tag release workflows.
- macOS Intel and Apple Silicon release bundles.
- Explicit capture only; no silent surveillance.
- Read-only Git context and resume packets.
- Explainable next-task prioritisation.
- Decision recall and solved-problem search.
- Weekly review, recurring friction detection, and cleanup advisor.
- Optional read-only GitHub CLI metadata cache.
- Export, import, and delete-data procedures.

## Privacy

Core commands do not require network access. GitHub metadata is queried only when `github --refresh` is used.
