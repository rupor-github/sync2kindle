<h1>
    <img src="docs/pumping_station.svg" style="vertical-align:middle; width:8%" align="absmiddle"/>
    <span style="vertical-align:middle;">&nbsp;&nbsp;Simple sideloading tool for Kindle devices</span>
</h1>

[![GitHub Release](https://img.shields.io/github/release/rupor-github/sync2kindle.svg)](https://github.com/rupor-github/sync2kindle/releases)

### Purpose
This is CLI tool for day-to-day synchronization of kindle books between local
directory and directory on device over the wire - using either MTP or old USBMS
mount or by using Amazon e-mail delivery.

It was created to support day-to-day side loading usage scenario (based on my
multi-year experience owning various Kindle devices):

I have one or more local directories containing books in Kindle-supported
formats, possibly organized into subdirectories by authors or genres for easier
navigation. I would like to run a single command (not a tool with a UI or
additional complexity) from the terminal or console to send these books to my
device, while preserving the original directory structure. If there are any
additional format specific actions possible (like copying generated page indexes or
extracting and copying thumbnails for books) they should be performed transparenly.

Later, I may add new books to the local directories. At the same time, as I
finish reading books on the device, they may be removed there. When I run the
tool again, I want these changes to be synchronized bidirectionally: new or
updated books should be sent to the device, and completed (and deleted) books
should be removed locally.

The tool should maintain a history of actions performed. If a book is added to
the device outside this process, it should be ignored by the tool and left
untouched. Similarly, any additional directories or files created by the device
(e.g., Kindle-generated files) should not be affected.

The tool should have a minimal number of options and be simple to use. It
should support synchronization from the same local directory to multiple target
devices. The history it maintains should be per device and per target directory
on the device, allowing different target directories on the same device to be
synchronized at different intervals (e.g., syncing "fiction" frequently and
"nonfiction" less often). Simplicity and reliability should take priority over
performance and added flexibility.

### Documentation

[User guide](docs/guide.md)

[Russian discussion forum](https://4pda.to/forum/index.php?showtopic=942250#)

### Installation

Download from the [releases page](https://github.com/rupor-github/sync2kindle/releases) and unpack it in a convenient location.

Release archives are named after the binary they contain:

- `s2k-*` archives contain the full build with USBMS, MTP, e-mail, history and configuration commands.
- `s2km-*` archives contain the minimal build with only e-mail, history and configuration commands.

Use `s2k` on supported Windows, Linux, and macOS systems when direct USB/MTP device synchronization is needed. Use `s2km` when you only need e-mail delivery or want a portable no-CGO build.

### Supported platforms and devices

Kindle devices which mount as USBMS storage (**everything before latest Kindle
Scribe, Paperwhite 12 or Colorsoft**) are supported with **USB** subcommand (tested
with PW2, PW10 and Voyage) and later ones (**Scribe, Colorsoft and latest
Paperwhite**) are supported by **MTP** subcommand (tested with PW12). E-Mail based
delivery should be device agnostic.

Full `s2k` releases are built for Windows x64, Linux x64, and macOS x64/arm64.
Minimal `s2km` releases are built for additional platforms listed on the releases
page. Full builds were tested on fresh Windows 11, KUbuntu 26.04, and macOS 26.6
on Apple Silicon. macOS x64 artifacts are built on GitHub's Intel macOS runners,
but I do not have access to an Intel Mac for hardware testing.

Windows MTP uses native COM/WPD and does not require CGO. Linux and macOS MTP
builds use CGO and libmtp. macOS USBMS support uses native Disk Arbitration and
IOKit APIs.

Synchronization logic code is platform independent; platform-specific code is limited to device drivers.
