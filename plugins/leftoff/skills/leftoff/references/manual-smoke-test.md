# Manual Smoke Test

Use a temporary store so the smoke test does not touch real records.

```sh
export LEFTOFF_STORE="$(mktemp -d)/.leftoff"
./scripts/leftoff init --store "$LEFTOFF_STORE"
./scripts/leftoff capture --store "$LEFTOFF_STORE" --project sample "task: Add release smoke test"
./scripts/leftoff now --store "$LEFTOFF_STORE" --minutes 30
./scripts/leftoff now --store "$LEFTOFF_STORE" --minutes 30 --json
./scripts/leftoff scan --store "$LEFTOFF_STORE" --repo . --json
./scripts/leftoff resume --store "$LEFTOFF_STORE" --repo . --json
./scripts/leftoff workspace add --store "$LEFTOFF_STORE" .
./scripts/leftoff workspace scan --store "$LEFTOFF_STORE"
./scripts/leftoff now --store "$LEFTOFF_STORE" --all --minutes 45 --json
./scripts/leftoff remember-why --store "$LEFTOFF_STORE" "release smoke"
./scripts/leftoff review-week --store "$LEFTOFF_STORE" --write
./scripts/leftoff friction --store "$LEFTOFF_STORE"
./scripts/leftoff clean-up --store "$LEFTOFF_STORE"
./scripts/leftoff export --store "$LEFTOFF_STORE" --out "$LEFTOFF_STORE/export.zip"
./scripts/leftoff delete-data --store "$LEFTOFF_STORE" --dry-run
```

Expected result:

- commands run without network access;
- records are created only inside the temporary store;
- cleanup is report-only by default;
- export creates a zip archive;
- delete-data dry run prints a preview and deletes nothing.

Optional GitHub metadata smoke test:

```sh
./scripts/leftoff github --store "$LEFTOFF_STORE" --repo . --refresh
./scripts/leftoff github --store "$LEFTOFF_STORE" --forget-cache
```

This is opt-in and requires `gh` to be installed and authenticated.
