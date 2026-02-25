## Debug reporter improvements

- Ported debug reporter enhancements from fb2cng: proper temp file cleanup on errors,
  atomic rename on close, and improved error handling throughout the report lifecycle.
- Added unit tests for reporter (5 tests covering creation, data storage, close, and error paths).

## History subcommand enhancements

- Expanded `history` command with 6 sub-subcommands: `list`, `steps`, `objects`, `diff`, `stats`, `orphans`.
- `list` — shows short database ID (8 hex chars), path, last step, and identifiers for each history DB.
- `steps` — tabular listing of all sync steps with timestamps, source/destination, and object counts.
- `objects` — lists objects in the latest (or specified via `--step`) sync step with path, size, modified time, and hash.
- `diff` — shows added/removed/changed objects between two steps (`--from`/`--to`, defaults to last two).
- `stats` — aggregate statistics: file/directory counts, total size, date range, breakdown by extension.
- `orphans` — identifies history databases with missing source directories or stale last sync (>180 days).
- All subcommands accept `--db` flag to filter by database ID prefix (any length, like git short hashes).
- Extracted shared query helpers into `history/queries.go`: `openReadOnly`, `identifiers`, `allSteps`, `stepObjects`.
