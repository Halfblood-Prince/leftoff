# Delete All Local Data

The local store can be removed with:

```sh
./bin/leftoff delete-data --confirm
```

Safer preview:

```sh
./bin/leftoff delete-data --dry-run
```

The command refuses to run unless the target directory contains `.leftoff-store`. Skill installation directories are separate from user data and are not deleted by this command.
