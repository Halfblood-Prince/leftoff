# Privacy Policy

`leftoff` is local-first. The core workflow stores data only under `~/.leftoff/` or a user-provided `--store` path.

## What Can Be Stored

- concise user-captured summaries;
- project slugs and project names;
- sanitized Git remote URLs;
- dirty state, branch names, ahead/behind status, unpushed commit counts, commit hashes, redacted commit titles, stale branch names, worktree status, worktree paths, and changed file paths;
- record IDs, creation dates, statuses, evidence labels, and activity metadata.

## What Must Not Be Stored

- secrets, tokens, passwords, certificates, private keys, or `.env` values;
- private source-code contents;
- full diffs or command output;
- full issue or pull request contents;
- arbitrary terminal history;
- browser profiles, SSH directories, credential stores, or unbounded home-directory scans.

## Network Access

The core workflow does not use the network. GitHub metadata is optional and queried only when the user explicitly runs `github --refresh`.

The GitHub cache stores minimal metadata only: numbers, titles, states, labels, review decisions, branches, timestamps, workflow names, and workflow statuses. Full PR bodies, issue bodies, reviews, logs, artifacts, and comments are not stored.

## Repair and Backups

When validation repairs malformed files, it first copies the original file into `backups/` inside the store.

Import also backs up existing files before overwrite. Delete-data requires explicit confirmation and a `.leftoff-store` marker.
