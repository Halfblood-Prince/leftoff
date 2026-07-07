# Privacy

`leftoff` is local-first. Core commands write only to a local store under
`~/.leftoff/` unless the user explicitly passes another `--store` path.

The tool stores concise user-provided records, task metadata, and compact
read-only Git state such as branch names, changed file paths, recent commit
titles, worktree paths, and sanitized remote URLs. It is designed not to store
source-code contents, full diffs, secrets, arbitrary terminal history, or
unredacted command output.

No analytics, telemetry, cloud service, or network request is required for the
core workflow. Optional GitHub metadata is fetched only when the user explicitly
runs `leftoff github --refresh`.

Plugin binary setup is separate from the core workflow. The setup scripts ask
for confirmation before network access, download release artifacts from GitHub,
verify checksums and provenance, and install the selected binary into a
user-owned plugin directory.

Users can inspect, edit, export, import, or delete the local store. Destructive
data deletion requires an explicit marker and confirmation, and dry-run previews
are supported.
