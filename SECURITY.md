# Security

Please report suspected vulnerabilities through GitHub Security Advisories for
`Halfblood-Prince/leftoff` when possible. Avoid posting secrets, private logs,
or exploit details in public issues.

`leftoff` treats local records as user-owned data. The core workflow avoids
network access, rejects likely secrets before persistence, and inspects Git
metadata read-only. Cleanup is report-only by default and does not delete Git
branches or worktrees.

Release binary setup must be explicit. Agents and scripts should never download
or execute installers silently, and should never use `curl | sh`. Use the
included setup scripts so release artifacts are downloaded, checksum-verified,
and provenance-verified before installation.
