# Contributing to sync2kindle

Contributions are welcome, including bug reports, documentation improvements, tests, and code changes. Please read these guidelines before opening an issue or pull request.

## Before you begin

- Read the [user guide](docs/guide.md) and relevant documentation.
- Search existing [issues](https://github.com/rupor-github/sync2kindle/issues) and [pull requests](https://github.com/rupor-github/sync2kindle/pulls) to avoid duplicates.
- Any code change, including a simple bug fix, requires an open issue first. Open an issue or join an existing one to discuss the problem and proposed approach before working on a code contribution.
- Documentation-only corrections may be submitted directly as pull requests without an issue.

An issue should explain the problem and intended outcome without requiring maintainers to infer them from a patch. Reviewing code, especially AI-generated code, can take more effort than writing a fix. Discussing the intent first helps relate a proposed change to the project's history and direction.

## Reporting bugs

The most valuable diagnostic information is the complete **`sync2kindle-report.zip`** archive produced by running the failing command with **`--debug`** (or `-d`). For example:

```sh
s2k --debug mtp
```

Use the same subcommand (`mtp`, `usb`, or `mail`), configuration, and options as the failing command. Debug mode is not a dry run: synchronization can still copy or remove files. Use `--dry-run` to preview actions when appropriate, and keep backups when investigating synchronization problems.

Attach the full archive rather than selected files or log excerpts. It covers most, if not all, of the technical information needed to investigate the problem, so you do not need to repeat information already captured there.

Also describe the expected behavior and actual result, any reproduction steps not captured in the report, and the Kindle model, connection method, and relevant firmware version. For synchronization issues, explain which files were added, changed, or removed locally and on the device before the failing run.

If no report is produced, include the sync2kindle version (`s2k --version`), operating system and architecture, exact command, relevant configuration, and available error messages. For book-specific issues, provide a minimal shareable sample when possible.

See [Troubleshooting](docs/guide.md#troubleshooting) for debug-report and log details. The report defaults to `sync2kindle-report.zip` in the current directory; its path is configurable through `reporting.destination`. **Review the archive before sharing:** it can contain configuration, synchronization history, device identifiers, book names, and local paths. Only share material you have permission to distribute, and remove private information, including credentials and e-mail addresses; if necessary, reproduce the issue with shareable data and generate a new report.

## Suggesting features

Describe the use case, the problem with the current behavior, and the result you would like to see. Include examples and alternatives where useful. Opening an issue does not guarantee that a feature will be implemented. Keep the project's focus on simple, reliable day-to-day Kindle synchronization in mind.

## Development and testing

The main development tasks support Linux and macOS hosts; cross-release tasks run on Linux. Use a locally installed Go version matching [`go.mod`](go.mod); tasks set `GOTOOLCHAIN=local+path`. The project includes [Task](https://taskfile.dev/) as a Go tool.

Full Linux and macOS builds require a C toolchain, `pkg-config`, and the libmtp development files (for example, `libmtp-dev` on Ubuntu, or `brew install pkg-config libmtp` on macOS). Windows MTP uses native COM/WPD and does not require CGO.

From the repository root:

```sh
# Build build/s2k, compile tests, and run static analysis.
go tool task

# Run tests.
go tool task test

# Run static analysis.
go tool task lint

# Run focused synchronization tests.
PACKAGES='./sync' go tool task test -- -run=TestPrepareActions
```

The default build requires `pre-commit` and installs Git hooks. Tasks also generate `misc/version.go`; do not edit it manually. See [`Taskfile.yaml`](Taskfile.yaml) and [`.pre-commit-config.yaml`](.pre-commit-config.yaml) for prerequisites and details. Maintainers use [Jujutsu (`jj`)](https://jj-vcs.dev/) as the primary version-control workflow; Git-based forks and pull requests are also welcome.

Follow the surrounding code style. Format Go changes with `goimports-reviser`:

```sh
GOFLAGS=-mod=mod goimports-reviser -format -set-alias -company-prefixes github.com/rupor-github <changed.go files>
```

If it is unavailable, use `gofmt` and check imports manually. Run `gopls check -severity=hint` on changed Go files and fix diagnostics. Add or update tests for behavior changes, and include a regression test for bug fixes where practical. Update documentation when changing user-visible behavior. After dependency changes, run `go tool task tidy` and `go tool task vendor` to keep module metadata and vendored release dependencies consistent.

## Submitting a pull request

For any pull request containing code changes, link the issue where the problem and proposed approach were discussed first. Only documentation-only corrections are exempt from this requirement.

1. Fork the repository and create a branch for your change.
2. Keep the pull request focused on one issue; avoid unrelated refactoring or formatting changes.
3. Use clear commit messages explaining the purpose of the changes.
4. Run the relevant tests and checks. State which checks you ran and explain any failures or checks you could not run.
5. Open a pull request describing the problem, your solution, and any compatibility implications. Link related issues and include before-and-after examples when helpful.
6. Respond to review feedback and update your branch as needed.

Do not include build artifacts, private data, or unrelated generated-file changes. For synchronization changes, test action planning and history handling, and verify that device-side files unknown to the history remain untouched. For driver changes, state which operating systems, connection methods, and devices you tested; use `--dry-run` before testing operations that may remove files.

Maintainers may request changes or decline contributions that do not fit the project's scope or maintenance needs. Reviews are provided as time permits; there is no guaranteed response time.

## AI-assisted contributions

AI-assisted contributions are welcome, provided a human remains actively involved throughout the process. You are responsible for everything you submit, regardless of how it was produced.

- Review and fully understand all submitted code, documentation, and tests.
- Verify the result and its relevance to the issue, rather than relying on the AI's claims.
- Be prepared to explain your choices, discuss trade-offs, and respond meaningfully to review feedback.
- The issue-first requirement for code changes applies equally to AI-assisted contributions.

Blindly copying AI-generated material does not help the project. Do not submit contributions you cannot understand, explain, or support in discussion; maintainers should not have to reverse-engineer their intent or act as the first human reviewer of unchecked AI output.

## Community conduct

Be respectful and constructive in issues, pull requests, and reviews. Discuss ideas and code rather than making personal remarks. Harassment and abusive behavior are not acceptable.

## Licensing

By submitting a contribution, you agree that it may be distributed under the project's [GNU General Public License v3.0](LICENSE). Only submit work you have the right to contribute, and preserve applicable copyright notices and third-party license information.
