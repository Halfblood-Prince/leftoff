# Export, Import, and Delete Data

## Export

```sh
./bin/leftoff export --out leftoff-export.zip
```

The archive is a zip file containing the local store and a `.leftoff-export-manifest.json` entry with tool and data format versions.

## Import

```sh
./bin/leftoff import --from leftoff-export.zip --confirm
```

Import rejects unsafe archive paths. Existing files are backed up before overwrite.

## Delete Local Data

Preview:

```sh
./bin/leftoff delete-data --dry-run
```

Delete:

```sh
./bin/leftoff delete-data --confirm
```

Deletion requires the `.leftoff-store` marker created by `init`.
