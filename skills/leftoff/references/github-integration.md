# Optional GitHub CLI Integration

GitHub metadata is optional enrichment. The core leftoff workflow does not require GitHub, authentication, or network access.

## Requirements

- `gh` installed.
- `gh auth login` completed by the user.
- A local Git repository with a GitHub remote, or a current `gh` repository context.

## Refresh Cache

```sh
./bin/leftoff github --repo . --refresh
```

This runs read-only queries:

```text
gh pr list --state open --limit 20 --json number,title,state,isDraft,reviewDecision,updatedAt,headRefName
gh issue list --state open --limit 20 --json number,title,state,updatedAt,labels
gh run list --limit 20 --json databaseId,status,conclusion,name,headBranch,updatedAt
```

## Show Cache Without Network

```sh
./bin/leftoff github --project leftoff
```

Without `--refresh`, no remote query is run.

## Retention

The default cache retention is 14 days:

```sh
./bin/leftoff github --project leftoff --retention-days 7
```

## Forget Cache

```sh
./bin/leftoff github --project leftoff --forget-cache
```

The cache file is backed up before removal.

## Stored Fields

The cache stores minimal metadata only: numbers, titles, states, labels, review decisions, branch names, timestamps, workflow names, and workflow statuses. It does not store bodies, comments, reviews, logs, artifacts, or secrets.
