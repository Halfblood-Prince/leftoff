# Threat Model

## Assets

- User-authored records under `~/.leftoff/`.
- Local repository metadata such as branch names, changed file paths, and commit titles.
- Optional GitHub metadata cache.

## Non-Goals

`leftoff` does not protect against a compromised local machine, malicious shell, or hostile user with filesystem access.

## Main Risks

- Accidentally storing secrets.
- Accidentally storing source contents or full command output.
- Accidentally deleting useful work.
- Silently contacting remote services.
- Importing archives with path traversal.
- Running hostile Git helpers configured by an unfamiliar repository.

## Controls

- Secret-pattern rejection during capture.
- Metadata-only Git inspection.
- Read-only Git scans disable filesystem-monitor helpers, ignore system/global Git config, and override interactive credential prompts.
- Branch names, file paths, commit titles, PR titles, and issue titles are redacted and length-capped before persistence.
- Report-only cleanup by default.
- GitHub integration requires explicit `github --refresh`.
- Zip import rejects absolute and parent-traversal paths.
- Delete-data requires both a marker file and `--confirm`.
- Repair and import create backups before overwrite.
