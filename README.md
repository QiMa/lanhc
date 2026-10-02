# Lanhc

https://lanhc.com

Private WireGuard® networks made easy

## Overview

This repository contains the majority of Lanhc's open source code.
Notably, it includes the `lanhcd` daemon and
the `lanhc` CLI tool. The `lanhcd` daemon runs on Linux, Windows,
[macOS](https://lanhc.com/kb/1065/macos-variants/), and to varying degrees
on FreeBSD and OpenBSD. The Lanhc iOS and Android apps use this repo's
code, but this repo doesn't contain the mobile GUI code.

Other [Lanhc repos](https://github.com/orgs/lanhc/repositories) of note:

* the Android app is at https://github.com/lanhc/lanhc-android
* the Synology package is at https://github.com/lanhc/lanhc-synology
* the QNAP package is at https://github.com/lanhc/lanhc-qpkg
* the Chocolatey packaging is at https://github.com/lanhc/lanhc-chocolatey

For background on which parts of Lanhc are open source and why,
see [https://lanhc.com/opensource/](https://lanhc.com/opensource/).

## Using

We serve packages for a variety of distros and platforms at
[https://pkgs.lanhc.com](https://pkgs.lanhc.com/).

## Other clients

The [macOS, iOS, and Windows clients](https://lanhc.com/download)
use the code in this repository but additionally include small GUI
wrappers. The GUI wrappers on non-open source platforms are themselves
not open source.

## Building

We always require the latest Go release, currently Go 1.26. (While we build
releases with our [Go fork](https://github.com/tailscale/go/), its use is not
required.)

```
go install lanhc.com/cmd/lanhc{,d}
```

If you're packaging Lanhc for distribution, use `build_dist.sh`
instead, to burn commit IDs and version info into the binaries:

```
./build_dist.sh lanhc.com/cmd/lanhc
./build_dist.sh lanhc.com/cmd/lanhcd
```

If your distro has conventions that preclude the use of
`build_dist.sh`, please do the equivalent of what it does in your
distro's way, so that bug reports contain useful version information.

## Bugs

Please file any issues about this code or the hosted service on
[the issue tracker](https://github.com/lanhc/lanhc/issues).

## Contributing

PRs welcome! But please file bugs. Commit messages should [reference
bugs](https://docs.github.com/en/github/writing-on-github/autolinked-references-and-urls).

We require [Developer Certificate of
Origin](https://en.wikipedia.org/wiki/Developer_Certificate_of_Origin)
`Signed-off-by` lines in commits.

See [commit-messages.md](docs/commit-messages.md) (or skim `git log`) for our commit message style.

## About Us

[Lanhc](https://lanhc.com/) is primarily developed by the
people at https://github.com/orgs/lanhc/people. For other contributors,
see:

* https://github.com/lanhc/lanhc/graphs/contributors
* https://github.com/lanhc/lanhc-android/graphs/contributors

## Legal

WireGuard is a registered trademark of Jason A. Donenfeld.
