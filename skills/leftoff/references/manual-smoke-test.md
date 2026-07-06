# Manual Smoke Test

Use a temporary store so the smoke test does not touch real records.

```sh
export LEFTOFF_STORE="$(mktemp -d)/.leftoff"
./bin/leftoff init --store "$LEFTOFF_STORE"
./bin/leftoff capture --store "$LEFTOFF_STORE" --project sample "task: Add release smoke test"
./bin/leftoff now --store "$LEFTOFF_STORE" --minutes 30
./bin/leftoff now --store "$LEFTOFF_STORE" --minutes 30 --json
./bin/leftoff scan --store "$LEFTOFF_STORE" --repo . --json
./bin/leftoff resume --store "$LEFTOFF_STORE" --repo . --json
./bin/leftoff workspace add --store "$LEFTOFF_STORE" .
./bin/leftoff workspace scan --store "$LEFTOFF_STORE"
./bin/leftoff now --store "$LEFTOFF_STORE" --all --minutes 45 --json
./bin/leftoff remember-why --store "$LEFTOFF_STORE" "release smoke"
./bin/leftoff review-week --store "$LEFTOFF_STORE" --write
./bin/leftoff friction --store "$LEFTOFF_STORE"
./bin/leftoff clean-up --store "$LEFTOFF_STORE"
./bin/leftoff export --store "$LEFTOFF_STORE" --out "$LEFTOFF_STORE/export.zip"
./bin/leftoff delete-data --store "$LEFTOFF_STORE" --dry-run
```

Expected result:

- commands run without network access;
- records are created only inside the temporary store;
- cleanup is report-only by default;
- export creates a zip archive;
- delete-data dry run prints a preview and deletes nothing.

Optional GitHub metadata smoke test:

```sh
./bin/leftoff github --store "$LEFTOFF_STORE" --repo . --refresh
./bin/leftoff github --store "$LEFTOFF_STORE" --forget-cache
```

This is opt-in and requires `gh` to be installed and authenticated.
