# Export, Import, and Delete Data

## Export

```sh
./scripts/leftoff export --out leftoff-export.zip
```

The archive is a zip file containing the local store and a `.leftoff-export-manifest.json` entry with tool and data format versions.

## Import

```sh
./scripts/leftoff import --from leftoff-export.zip --confirm
```

Import rejects unsafe archive paths. Existing files are backed up before overwrite.

## Delete Local Data

Preview:

```sh
./scripts/leftoff delete-data --dry-run
```

Delete:

```sh
./scripts/leftoff delete-data --confirm
```

Deletion requires the `.leftoff-store` marker created by `init`.
