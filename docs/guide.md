# S2K User Guide

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Synchronization Algorithm](#synchronization-algorithm)
- [History](#history)
- [Configuration](#configuration)
- [E-mail Delivery](#e-mail-delivery)
- [Thumbnails and Supplemental Artifacts](#thumbnails-and-supplemental-artifacts)
- [Troubleshooting](#troubleshooting)

## Introduction

**sync2kindle** (`s2k`) is a CLI tool for bidirectional synchronization of ebook files between a local directory and Amazon Kindle devices. It was designed for a single use case: run one command from the terminal to keep a local book collection and a Kindle device in sync, with no UI or extra complexity.

### Supported Connection Methods

- **MTP** - Media Transfer Protocol for newer Kindle devices (Scribe, Paperwhite 12, Colorsoft)
- **USB** - USB Mass Storage mount for older Kindle devices (Kindle 3/4, Paperwhite 2, Paperwhite 10, Voyage)
- **E-mail** - Amazon's Send-to-Kindle email delivery service (device agnostic)

### Key Features

- Bidirectional sync: new local books are sent to device, books deleted on device are removed locally
- Per-device, per-target history tracking via SQLite databases
- Automatic thumbnail extraction and synchronization for ebook-type books (.mobi, .azw3, .kfx)
- Supplemental artifact handling (`.apnx` page indexes, `.sdr` directories)
- Directory structure preservation on the device
- Dry-run mode for previewing changes
- Debug reporting for troubleshooting
- Flexible configuration via YAML files
- Support for multiple devices and target directories from the same local source

### Supported Book Formats (USB/MTP)

- `.mobi` - Mobipocket format
- `.azw3` - Kindle Format 8
- `.kfx` - Kindle Format X
- `.pdf` - Portable Document Format

### Supported Formats (E-mail Delivery)

Amazon's Send-to-Kindle service accepts: `.doc`, `.docx`, `.html`, `.htm`, `.rtf`, `.txt`, `.jpeg`, `.jpg`, `.gif`, `.png`, `.bmp`, `.pdf`, `.epub`

### Supported Kindle Devices

| USB VID:PID     | Device                          | Protocol |
|-----------------|---------------------------------|----------|
| `1949:0002`     | Kindle                          | USB      |
| `1949:0004`     | Kindle 3/4/Paperwhite           | USB      |
| `1949:9981`     | Kindle Scribe/Paperwhite 12/Colorsoft | MTP |

## Installation

1. Download the latest release from the [releases page](https://github.com/rupor-github/sync2kindle/releases)
2. Choose the archive for your platform and required functionality
3. Unpack the archive to a convenient location

Release archives are named after the binary they contain:

- `s2k-*` archives contain the full build with `mtp`, `usb`, `mail`, `history`, and `dumpconfig` commands.
- `s2km-*` archives contain the minimal build with only `mail`, `history`, and `dumpconfig` commands.

Use `s2k` on supported Windows, Linux, and macOS systems when direct USB/MTP device synchronization is needed. Use `s2km` when you only need e-mail delivery or want a portable no-CGO build for a platform where USB/MTP drivers are not available in this project.

**Platform Notes:**

- **Windows x64** - No additional dependencies required (uses COM/WPD for MTP)
- **Linux x64** - Requires `libmtp` installed on the system (uses CGO)
- **macOS x64/arm64** - Requires `libmtp` installed on the system for MTP; USB uses native Disk Arbitration and IOKit APIs
- **Minimal builds** - Do not require CGO or `libmtp`, but do not include `mtp` or `usb` commands

## Quick Start

### Sync books to a Kindle via MTP (newer devices)

Connect your Kindle via USB and run:

```bash
s2k mtp
```

This synchronizes books from the current directory to `documents/mybooks` on the device using default settings.

### Sync books to a Kindle via USB (older devices)

```bash
s2k usb
```

### Send books via e-mail

```bash
s2k -c kindle-mail.yaml mail
```

Requires a configuration file with SMTP settings and a target Kindle email address.

### Preview changes without modifying anything

```bash
s2k mtp --dry-run
```

### Dump default configuration to a file

```bash
s2k dumpconfig myconfig.yaml
```

## Basic Usage

### Command Line Syntax

```
s2k [global options] command [command options]
```

### Global Options

- `--config FILE, -c FILE` - Load configuration from YAML file
- `--debug, -d` - Enable debug mode, produces diagnostic report archive
- `--version, -v` - Print version information
- `--help, -h` - Show help

### Commands

The full `s2k` binary includes all commands below. The minimal `s2km` binary includes only `mail`, `history`, and `dumpconfig`.

#### mtp

Synchronizes books between a local source directory and a target directory on a Kindle device connected over MTP protocol.

```
s2k mtp [options]
```

**Options:**

- `--ignore-device-removals, -i` - Do not remove local books when they are deleted on the device (one-way sync)
- `--dry-run` - Preview changes without performing any actual operations

**Notes:** The Kindle device must be connected at the time of operation. The first supported MTP device found is used unless `device_serial` is specified in configuration.

#### usb

Synchronizes books between a local source directory and a target directory on a Kindle device mounted as USB mass storage.

```
s2k usb [options]
```

**Options:**

- `--ignore-device-removals, -i` - Do not remove local books when they are deleted on the device
- `--dry-run` - Preview changes without performing any actual operations
- `--unmount, -u` - Attempt to safely unmount the device storage after sync

**Notes on `--unmount`:** Results are OS-dependent. On Windows, it may fail if buffers haven't been flushed or if something still has the device open. On Linux, it requires admin privileges and will only unmount after the filesystem ceases to be busy. On macOS, USB unmount uses native Disk Arbitration APIs. On Linux, you can use `eject` or `udisksctl` commands instead.

#### mail

Synchronizes books between a local source directory and a Kindle device using Amazon's Send-to-Kindle email service.

```
s2k mail [options]
```

**Options:**

- `--dry-run` - Preview changes without sending any emails

**Notes:** Since there is no way to access device content via email, all decisions are based on local files and history only. Books removed on the device will not be detected. Proper SMTP configuration is required, including an authorized sender email address (configured in your Amazon account settings).

#### history

Reports on local history databases. When invoked without a subcommand, lists basic details for each database.

```
s2k history
s2k history <subcommand> [options]
```

Each history database is identified by a short hex ID (the first 8 characters of the SHA256 filename). Use `history list` to discover these IDs, then pass them to other subcommands with the `--db` flag to narrow the scope to a single database.

##### history list

Lists basic details for each history database: short ID, path, last step number, and identifiers (device, target, protocol).

```bash
s2k history list
```

##### history steps

Lists all sync steps with timestamps, source/destination paths, and object counts.

```bash
s2k history steps
s2k history steps --db a1b2c3d4
```

**Options:**

- `--db ID` - Filter by database ID prefix (any length)

##### history objects

Lists all objects in the latest (or specified) sync step, showing path, size, modification time, and content hash.

```bash
s2k history objects
s2k history objects --db a1b2c3d4 --step 3
```

**Options:**

- `--db ID` - Filter by database ID prefix (any length)
- `--step N, -s N` - Step number to inspect (default: latest)

##### history diff

Shows changes (added, removed, changed files) between two sync steps. Defaults to comparing the last two steps.

```bash
s2k history diff
s2k history diff --db a1b2c3d4 --from 2 --to 5
```

**Options:**

- `--db ID` - Filter by database ID prefix (any length)
- `--from N` - Starting step number
- `--to N` - Ending step number

If neither `--from` nor `--to` is specified, the last two steps are compared. If only `--to` is given, it is compared against its predecessor. If only `--from` is given, it is compared against the latest step.

##### history stats

Shows aggregate statistics for the latest (or specified) step: file and directory counts, total size, date range, breakdown by file extension, and thumbnail counts.

```bash
s2k history stats
s2k history stats --db a1b2c3d4 --step 3
```

**Options:**

- `--db ID` - Filter by database ID prefix (any length)
- `--step N, -s N` - Step number to inspect (default: latest)

##### history orphans

Identifies history databases that may be stale or no longer needed. A database is flagged as possibly orphaned when the last sync source directory no longer exists or the last sync was more than 180 days ago.

```bash
s2k history orphans
s2k history orphans --db a1b2c3d4
```

**Options:**

- `--db ID` - Filter by database ID prefix (any length)

#### dumpconfig

Dumps either default or active configuration in YAML format.

```
s2k dumpconfig [options] [DESTINATION]
```

**Options:**

- `--dry-run` - Output the active configuration that would be used in actual operations (including values merged from `--config` file)

**Arguments:**

- `DESTINATION` - File to write configuration to. If absent, outputs to STDOUT.

Without `--dry-run`, produces default configuration values. With `--dry-run`, produces the merged active configuration.

### Examples

**Sync with a custom configuration:**
```bash
s2k -c kindle-pw12.yaml mtp
```

**Preview what would happen:**
```bash
s2k -c myconfig.yaml mtp --dry-run
```

**One-way sync (local to device only):**
```bash
s2k mtp --ignore-device-removals
```

**Sync via USB and safely unmount:**
```bash
s2k usb --unmount
```

**Debug mode for troubleshooting:**
```bash
s2k -d mtp
```
Creates `sync2kindle-report.zip` with complete diagnostic information.

**Get the active configuration (merged defaults + custom):**
```bash
s2k -c myconfig.yaml dumpconfig --dry-run
```

## Synchronization Algorithm

The sync algorithm uses three-way comparison between **L**ocal files, **H**istory database, and **D**evice contents. Eight cases are defined:

| # | L | H | D | Cause | Operation |
|---|---|---|---|-------|-----------|
| 1 | - | - | - | Nothing | Ignore |
| 2 | - | - | + | Manually added to device | Ignore (leave untouched) |
| 3 | + | - | - | New local book | Copy to device |
| 4 | + | - | + | History error | Treat as case 3 |
| 5 | - | + | - | Manually removed from both | Ignore |
| 6 | - | + | + | Removed locally | Remove from device |
| 7 | + | + | - | Removed from device | Remove locally |
| 8 | + | + | + | In sync | Ignore (but check for changes) |

Where `+` means the book exists and `-` means it does not.

### Key Behaviors

- **Case 3** (new books): New or updated books are copied to the device. Directory structure is recreated on the device relative to the target path.
- **Case 6** (removed locally): When you delete a book from your local directory, it will be removed from the device on the next sync.
- **Case 7** (removed from device): When you delete a book from the Kindle (e.g., after finishing it), it will be removed locally on the next sync. Use `--ignore-device-removals` to prevent this.
- **Case 8** (in sync): Books present everywhere are checked for content changes using SHA256 hashes. If a book has been modified locally, it is re-sent to the device.
- **Case 2** (manually added to device): Books added to the device outside of s2k are left untouched.

### Supplemental Artifacts

When a book is synced, s2k also handles:
- **Page index files** (`.apnx`) - Located either alongside the book or in a `.sdr` subdirectory
- **Sidecar directories** (`.sdr`) - Kindle-generated data directories for books

When a book is removed, its supplemental artifacts are cleaned up as well. Empty parent directories are removed recursively.

### History Database

Each unique combination of protocol, device, and target directory gets its own SQLite history database. History files are stored in the configured `history` directory (default: `~/.s2k/history`). The filename is a SHA256 hash derived from the protocol, device identifier, and target path.

This design allows you to:
- Sync the same local directory to multiple devices
- Sync different target directories on the same device at different intervals
- Maintain independent histories that don't interfere with each other

## History

The `history` command and its subcommands allow you to inspect and diagnose the sync history databases without performing any sync operations.

### Database Identification

Each history database file is named after a SHA256 hash derived from the device identifier, target path, and protocol. Since these names are not human-friendly, the `list` subcommand shows a short 8-character hex prefix for each database. You can use any prefix of any length with the `--db` flag to select a specific database (similar to how git allows abbreviated commit hashes).

### Typical Workflow

1. Run `s2k history list` to see all databases and their short IDs
2. Use `s2k history steps --db <id>` to see the sync timeline for a database
3. Use `s2k history diff --db <id>` to see what changed in the last sync
4. Use `s2k history stats --db <id>` to get a summary of stored content
5. Use `s2k history orphans` to find databases that may no longer be needed

## Configuration

### Configuration File Structure

Configuration files use YAML format. Here is a minimal example:

```yaml
source: /home/user/Books/Kindle
target: documents/fiction
```

And a more complete example:

```yaml
# Local directory with books
source: /home/user/Books/Kindle

# Target path on the device (relative to device root)
target: documents/fiction

# Where to store history databases
history: /home/user/.s2k/history

# Only sync specific device
# device_serial: "G000AB1234567890"

# Book file extensions to synchronize
book_extensions: [.mobi, .azw3, .kfx, .pdf]

# Thumbnail extensions to recognize on device
thumb_extensions: [.jpg]

# Thumbnail dimensions for cover extraction
thumbnails:
  width: 330
  height: 470

logging:
  console:
    level: normal
  file:
    destination: sync2kindle.log
    level: debug
    mode: overwrite
```

### Configuration Options

#### Core Settings

- **`source`** - Local directory containing books to synchronize. Can be relative to the current working directory or absolute. Default: `.` (current directory)
- **`target`** - Either a path on the device for books (always relative, cannot contain `@`) or an email address for Kindle email delivery. Default: `documents/mybooks`
- **`history`** - Directory to store per-device SQLite history databases. Default: `~/.s2k/history`
- **`device_serial`** - Optional serial number to target a specific connected device. When set, only this device will be used with this configuration. Ignored for email delivery. If not set, the first connected supported device is selected automatically.

#### Book and Thumbnail Extensions

- **`book_extensions`** - File extensions recognized as books. Default: `[.mobi, .azw3, .kfx, .pdf]`
- **`thumb_extensions`** - File extensions recognized as thumbnails on the device. Default: `[.jpg]`

#### Thumbnail Settings

```yaml
thumbnails:
  width: 330
  height: 470
```

When an ebook (EBOK type, not personal document/PDOC) is processed, cover thumbnails are extracted from `.mobi`, `.azw3`, and `.kfx` files, resized to the configured dimensions, and copied to the device's `system/thumbnails` directory. Ignored when thumbnails are not accessible on the device or when using email delivery.

#### SMTP Settings (E-mail Delivery)

```yaml
smtp:
  from: "your.email@gmail.com"
  server: smtp.gmail.com
  port: 587
  user: "your.email@gmail.com"
  password: "your-app-password"
```

- **`from`** - Sender email address, must be authorized in your Amazon account settings
- **`server`** - SMTP server hostname. Default: `smtp.gmail.com`
- **`port`** - SMTP server port. Default: `587`
- **`user`** - SMTP authentication username
- **`password`** - SMTP authentication password (redacted in logs and config dumps)

#### Logging Configuration

```yaml
logging:
  console:
    level: normal
  file:
    destination: sync2kindle.log
    level: debug
    mode: overwrite
```

**Console levels:**
- `none` - Suppress all console logging
- `normal` - INFO level and higher
- `debug` - All log messages

**File levels:** Same as console (`none`, `normal`, `debug`)

**File modes:**
- `append` - Keep all old log messages, append new ones
- `overwrite` - Keep only messages from the last run

#### Debug Reporting

```yaml
reporting:
  destination: sync2kindle-report.zip
```

When `--debug` flag is used, a ZIP archive is produced at the specified path containing full debug logs and diagnostic artifacts.

### Configuration Loading

1. **No config file** - Uses built-in defaults (`source: .`, `target: documents/mybooks`)
2. **Custom config** - Specify with `-c` flag: `s2k -c myconfig.yaml mtp`
3. **Merged config** - Your settings override defaults, missing values use defaults

### Getting Default Configuration

```bash
s2k dumpconfig default.yaml
```

This provides a complete template with all available options and their default values.

### Getting Active Configuration

```bash
s2k -c myconfig.yaml dumpconfig --dry-run
```

Shows the actual configuration that would be used (defaults merged with your custom settings).

### Configuration Best Practices

1. **One config per device/target pair.** Keep separate configurations for each device and target directory combination (e.g., `kindle-pw12-fiction.yaml`, `kindle-scribe-nonfiction.yaml`).
2. **Start from defaults.** Dump the default configuration and modify only what you need.
3. **Keep libraries small.** Rather than syncing a huge library all at once, use multiple target directories. Kindle storage is slow, and smaller syncs are faster and more reliable.
4. **Test with `--dry-run`.** Always preview changes before running sync for real, especially with a new configuration.
5. **Use `--debug` when testing new setups.** The debug report contains all the information needed to diagnose issues.

## E-mail Delivery

E-mail delivery uses Amazon's Send-to-Kindle service to deliver books to your device wirelessly.

### Setup

1. **Configure SMTP.** You need a working SMTP server. Gmail with app passwords is a common choice:

```yaml
target: "your-kindle-name@kindle.com"

smtp:
  from: "your.email@gmail.com"
  server: smtp.gmail.com
  port: 587
  user: "your.email@gmail.com"
  password: "your-gmail-app-password"
```

2. **Authorize the sender.** Add the sender email address to your Amazon account's approved Personal Document Email List at [Manage Your Content and Devices](https://www.amazon.com/hz/mycd/myx).

3. **Set the target.** The `target` field must contain your Kindle's email address (contains `@`).

### Limitations

- **No device visibility.** Since there is no access to the device, s2k cannot detect books removed from the device. All decisions are based on local files and history.
- **No thumbnails or page indexes.** Supplemental artifacts are not sent via email.
- **No bidirectional sync.** Device removals cannot be detected, so sync is effectively one-way.
- **Format restrictions.** Only formats accepted by Amazon's service are supported (see [Supported Formats](#supported-formats-e-mail-delivery) above).

### EPUB Metadata

When sending EPUB files via email, s2k extracts the book title from EPUB metadata and uses it as the email subject line. For other formats, the filename is used.

## Thumbnails and Supplemental Artifacts

### Thumbnail Extraction

For ebook-type (EBOK) books in `.mobi`, `.azw3`, and `.kfx` formats, s2k automatically:

1. Extracts the cover image from the book's binary data
2. Resizes it to the configured thumbnail dimensions (default: 330x470)
3. Copies it to the device's `system/thumbnails` directory

This ensures that sideloaded books display proper cover art in the Kindle library.

**Supported thumbnail sources:**
- **MOBI/AZW3** - Cover extracted from EXTH records in the PDB header
- **KFX** - Cover extracted from the Amazon Ion binary container format

Thumbnails are only processed when the `system/thumbnails` directory is accessible on the device (USB and MTP connections only, not email).

### Page Index Files (.apnx)

If `.apnx` page index files exist alongside your books (either in the same directory or in a `.sdr` subdirectory), they are automatically synchronized to the device. This enables page number display on the Kindle.

When a book is removed, its associated `.apnx` files and `.sdr` directories are cleaned up as well.

## Troubleshooting

### Enable Debug Mode

Always use debug mode when investigating issues:

```bash
s2k -d mtp
```

This creates `sync2kindle-report.zip` with complete diagnostic information including full debug logs.

### Common Issues

#### No Device Found

**Problem:** "no supported device found"

**Solutions:**
- Verify the Kindle is connected via USB and recognized by the OS
- Check that the device is a supported model (see [Supported Kindle Devices](#supported-kindle-devices))
- On Linux or macOS, ensure `libmtp` is installed and the device is accessible (Linux may need udev rules)
- Try using the correct subcommand (`mtp` for newer devices, `usb` for older ones)
- If you have multiple devices connected, use `device_serial` in configuration to target a specific one

#### No Books Found

**Problem:** "no books in the source path"

**Solutions:**
- Verify the `source` path points to a directory containing book files
- Check that files have supported extensions (`.mobi`, `.azw3`, `.kfx`, `.pdf`)
- Ensure `book_extensions` in configuration includes the formats you need

#### E-mail Delivery Fails

**Problem:** "unable to send e-mail"

**Solutions:**
- Verify SMTP settings (server, port, user, password)
- For Gmail, use an [App Password](https://support.google.com/accounts/answer/185833) rather than your account password
- Check that the sender address is authorized in your Amazon account settings
- Verify the target email address is correct (should end with `@kindle.com`)
- Test with `--dry-run` first to verify configuration

#### USB Unmount Fails

**Problem:** Device won't unmount with `--unmount` flag

**Solutions:**
- On Windows, ensure no other program has the device open (File Explorer, etc.)
- On Linux, use `eject` or `udisksctl` commands instead
- On macOS, ensure no other application is using the mounted volume
- Wait a moment for write buffers to flush before unmounting

#### Configuration Not Working

**Problem:** Settings seem ignored

**Solutions:**
- Verify YAML syntax (use an online YAML validator)
- Check indentation (spaces, not tabs)
- Dump active config: `s2k -c myconfig.yaml dumpconfig --dry-run`
- Look for error messages in logs about configuration

#### Unexpected File Removals

**Problem:** Books are being removed locally that you want to keep

**Solutions:**
- Use `--ignore-device-removals` flag to prevent case #7 (device removals propagating to local)
- Review the [sync algorithm](#synchronization-algorithm) to understand when removals occur
- Use `--dry-run` to preview what would happen before running sync

### Getting Help

When reporting issues, include:

1. **Version information:** `s2k --version`
2. **Command used:** Full command line with all options
3. **Debug report:** Output from `s2k -d mtp` (or `usb`/`mail`)
4. **Configuration:** Your YAML config file (if used, with SMTP password redacted)
5. **OS and device:** Operating system version and Kindle model

Visit the [GitHub repository](https://github.com/rupor-github/sync2kindle) to:
- Report bugs in Issues
- Request features
- Contribute code
- View source code

### Log Files

Check logs for detailed information:

**Console output:**
- Shows INFO level messages by default
- Change with `logging.console.level` in config

**File log:**
- Default: `sync2kindle.log` in current directory
- Contains DEBUG level by default
- Configure with `logging.file` settings

**Debug report:**
- Created with `--debug` flag
- ZIP archive with complete diagnostic data
- Location: `sync2kindle-report.zip` (configurable)

---

**Version:** For specific version features, see `s2k --version` and release notes.
