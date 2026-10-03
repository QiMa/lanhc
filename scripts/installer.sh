#!/bin/sh
# Copyright (c) Tailscale Inc & contributors
# SPDX-License-Identifier: BSD-3-Clause
#
# This script detects the current operating system, and installs
# 蓝核AI智控台（根据各操作系统惯例）
#
# Environment variables:
#   TRACK: Set to "stable" or "unstable" (default: stable)
#   LANHC_VERSION: Pin to a specific version (e.g., "1.88.4")
#
# Examples:
#   curl -fsSL https://lanhc.com/install.sh | sh
#   curl -fsSL https://lanhc.com/install.sh | LANHC_VERSION=1.88.4 sh
#   curl -fsSL https://lanhc.com/install.sh | TRACK=unstable sh

set -eu

# All the code is wrapped in a main function that gets called at the
# bottom of the file, so that a truncated partial download doesn't end
# up executing half a script.
main() {
	# Step 1: detect the current linux distro, version, and packaging system.
	#
	# We rely on a combination of 'uname' and /etc/os-release to find
	# an OS name and version, and from there work out what
	# installation method we should be using.
	#
	# The end result of this step is that the following three
	# variables are populated, if detection was successful.
	OS=""
	VERSION=""
	PACKAGETYPE=""
	APT_KEY_TYPE="" # Only for apt-based distros
	APT_SYSTEMCTL_START=false # Only needs to be true for Kali
	TRACK="${TRACK:-stable}"
	LANHC_VERSION="${LANHC_VERSION:-}"

	case "$TRACK" in
		stable|unstable)
			;;
		*)
			echo "unsupported track $TRACK"
			exit 1
			;;
	esac

	if [ -f /etc/os-release ]; then
		# /etc/os-release populates a number of shell variables. We care about the following:
		#  - ID: the short name of the OS (e.g. "debian", "freebsd")
		#  - VERSION_ID: the numeric release version for the OS, if any (e.g. "18.04")
		#  - VERSION_CODENAME: the codename of the OS release, if any (e.g. "buster")
		#  - UBUNTU_CODENAME: if it exists, use instead of VERSION_CODENAME
		. /etc/os-release
		VERSION_MAJOR="${VERSION_ID:-}"
		VERSION_MAJOR="${VERSION_MAJOR%%.*}"
		case "$ID" in
			ubuntu|pop|neon|tuxedo)
				OS="ubuntu"
				if [ "${UBUNTU_CODENAME:-}" != "" ]; then
				    VERSION="$UBUNTU_CODENAME"
				else
				    VERSION="$VERSION_CODENAME"
				fi
				PACKAGETYPE="apt"
				# Third-party keyrings became the preferred method of
				# installation in Ubuntu 20.04.
				if [ "$VERSION_MAJOR" -lt 20 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			debian)
				OS="$ID"
				VERSION="$VERSION_CODENAME"
				PACKAGETYPE="apt"
				# Third-party keyrings became the preferred method of
				# installation in Debian 11 (Bullseye).
				if [ -z "${VERSION_ID:-}" ]; then
					# rolling release. If you haven't kept current, that's on you.
					APT_KEY_TYPE="keyring"
				# Parrot Security is a special case that uses ID=debian
				elif [ "$NAME" = "Parrot Security" ]; then
					# All versions new enough to have this behaviour prefer keyring
					# and their VERSION_ID is not consistent with Debian.
					APT_KEY_TYPE="keyring"
					# They don't specify the Debian version they're based off in os-release
					# but Parrot 6 is based on Debian 12 Bookworm.
					VERSION=bookworm
				elif [ "$VERSION_MAJOR" -lt 11 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			linuxmint)
				if [ "${UBUNTU_CODENAME:-}" != "" ]; then
				    OS="ubuntu"
				    VERSION="$UBUNTU_CODENAME"
				elif [ "${DEBIAN_CODENAME:-}" != "" ]; then
				    OS="debian"
				    VERSION="$DEBIAN_CODENAME"
				else
				    OS="ubuntu"
				    VERSION="$VERSION_CODENAME"
				fi
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 5 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			elementary)
				OS="ubuntu"
				VERSION="$UBUNTU_CODENAME"
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 6 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			industrial-os)
				OS="debian"
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 5 ]; then
					VERSION="buster"
					APT_KEY_TYPE="legacy"
				else
					VERSION="bullseye"
					APT_KEY_TYPE="keyring"
				fi
				;;
			parrot|mendel)
				OS="debian"
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 5 ]; then
					VERSION="buster"
					APT_KEY_TYPE="legacy"
				else
					VERSION="bullseye"
					APT_KEY_TYPE="keyring"
				fi
				;;
			galliumos)
				OS="ubuntu"
				PACKAGETYPE="apt"
				VERSION="bionic"
				APT_KEY_TYPE="legacy"
				;;
			pureos|kaisen)
				OS="debian"
				PACKAGETYPE="apt"
				VERSION="bullseye"
				APT_KEY_TYPE="keyring"
				;;
			raspbian)
				OS="$ID"
				VERSION="$VERSION_CODENAME"
				PACKAGETYPE="apt"
				# Third-party keyrings became the preferred method of
				# installation in Raspbian 11 (Bullseye).
				if [ "$VERSION_MAJOR" -lt 11 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			kali)
				OS="debian"
				PACKAGETYPE="apt"
				APT_SYSTEMCTL_START=true
				# Third-party keyrings became the preferred method of
				# installation in Debian 11 (Bullseye), which Kali switched
				# to in roughly 2021.x releases
				if [ "$VERSION_MAJOR" -lt 2021 ]; then
					# Kali VERSION_ID is "kali-rolling", which isn't distinguishing
					VERSION="buster"
					APT_KEY_TYPE="legacy"
				else
					VERSION="bullseye"
					APT_KEY_TYPE="keyring"
				fi
				;;
			Deepin|deepin)  # https://github.com/lanhc/lanhc/issues/7862
				OS="debian"
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 20 ]; then
					APT_KEY_TYPE="legacy"
					VERSION="buster"
				else
					APT_KEY_TYPE="keyring"
					VERSION="bullseye"
				fi
				;;
			pika)
				PACKAGETYPE="apt"
				# All versions of PikaOS are new enough to prefer keyring
				APT_KEY_TYPE="keyring"
				# Older versions of PikaOS are based on Ubuntu rather than Debian
				if [ "$VERSION_MAJOR" -lt 4 ]; then
					OS="ubuntu"
					VERSION="$UBUNTU_CODENAME"
				else
					OS="debian"
					VERSION="$DEBIAN_CODENAME"
				fi
				;;
			sparky)
				OS="debian"
				PACKAGETYPE="apt"
				VERSION="$DEBIAN_CODENAME"
				APT_KEY_TYPE="keyring"
				;;
			centos)
				OS="$ID"
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="dnf"
				if [ "$VERSION" = "7" ]; then
					PACKAGETYPE="yum"
				fi
				;;
			ol)
				OS="oracle"
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="dnf"
				if [ "$VERSION" = "7" ]; then
					PACKAGETYPE="yum"
				fi
				;;
			rhel|miraclelinux)
				OS="$ID"
				if [ "$ID" = "miraclelinux" ]; then
					OS="rhel"
				fi
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="dnf"
				if [ "$VERSION" = "7" ]; then
					PACKAGETYPE="yum"
				fi
				;;
			fedora)
				OS="$ID"
				VERSION=""
				PACKAGETYPE="dnf"
				;;
			rocky|almalinux|nobara|openmandriva|sangoma|risios|cloudlinux|alinux|fedora-asahi-remix|ultramarine)
				OS="fedora"
				VERSION=""
				PACKAGETYPE="dnf"
				;;
			amzn)
				OS="amazon-linux"
				VERSION="$VERSION_ID"
				PACKAGETYPE="yum"
				;;
			xenenterprise)
				OS="centos"
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="yum"
				;;
			opensuse-leap|sles)
				OS="opensuse"
				VERSION="leap/$VERSION_ID"
				PACKAGETYPE="zypper"
				;;
			opensuse-tumbleweed|opensuse-slowroll)
				OS="opensuse"
				VERSION="tumbleweed"
				PACKAGETYPE="zypper"
				;;
			sle-micro-rancher)
				OS="opensuse"
				VERSION="leap/15.4"
				PACKAGETYPE="zypper"
				;;
			arch|archarm|endeavouros|blendos|garuda|archcraft|cachyos)
				OS="arch"
				VERSION="" # rolling release
				PACKAGETYPE="pacman"
				;;
			manjaro|manjaro-arm|biglinux)
				OS="manjaro"
				VERSION="" # rolling release
				PACKAGETYPE="pacman"
				;;
			alpine)
				OS="$ID"
				VERSION="$VERSION_ID"
				PACKAGETYPE="apk"
				;;
			postmarketos)
				OS="alpine"
				VERSION="$VERSION_ID"
				PACKAGETYPE="apk"
				;;
			nixos)
				echo "请直接在 NixOS 配置中添加蓝核AI智控台："
				echo
				echo "services.lanhc.enable = true;"
				exit 1
				;;
			bazzite)
				echo "Bazzite 默认已安装蓝核AI智控台。"
				echo "请以 root 身份运行以下命令启用蓝核AI智控台："
				echo
				echo "ujust enable-lanhc"
				echo "lanhc up"
				exit 1
				;;
			void)
				OS="$ID"
				VERSION="" # rolling release
				PACKAGETYPE="xbps"
				;;
			gentoo)
				OS="$ID"
				VERSION="" # rolling release
				PACKAGETYPE="emerge"
				;;
			freebsd)
				OS="$ID"
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="pkg"
				;;
			osmc)
				OS="debian"
				PACKAGETYPE="apt"
				VERSION="bullseye"
				APT_KEY_TYPE="keyring"
				;;
			photon)
				OS="photon"
				VERSION="$VERSION_MAJOR"
				PACKAGETYPE="tdnf"
				;;
			zorin)
				OS="ubuntu"
				VERSION="$UBUNTU_CODENAME"
				PACKAGETYPE="apt"
				if [ "$VERSION_MAJOR" -lt 16 ]; then
					APT_KEY_TYPE="legacy"
				else
					APT_KEY_TYPE="keyring"
				fi
				;;
			steamos)
				echo "要在 SteamOS 上安装蓝核AI智控台，请按这里的说明操作："
				echo "https://github.com/lanhc-dev/deck-lanhc"
				exit 1
				;;
			kde-linux)
				echo "KDE Linux 维护者提供了多种安装蓝核AI智控台的方式。以下说明不受蓝核AI智控台官方支持："
				echo "https://linux.kde.org/docs/more-software/#lanhc"
				exit 1
				;;

			# TODO: wsl?
			# TODO: synology? qnap?
		esac
	fi

	# If we failed to detect something through os-release, consult
	# uname and try to infer things from that.
	if [ -z "$OS" ]; then
		if type uname >/dev/null 2>&1; then
			case "$(uname)" in
				FreeBSD)
					# FreeBSD before 12.2 doesn't have
					# /etc/os-release, so we wouldn't have found it in
					# the os-release probing above.
					OS="freebsd"
					VERSION="$(freebsd-version | cut -f1 -d.)"
					PACKAGETYPE="pkg"
					;;
				OpenBSD)
					OS="openbsd"
					VERSION="$(uname -r)"
					PACKAGETYPE=""
					;;
				Darwin)
					OS="macos"
					VERSION="$(sw_vers -productVersion | cut -f1-2 -d.)"
					PACKAGETYPE="appstore"
					;;
				Linux)
					OS="other-linux"
					VERSION=""
					PACKAGETYPE=""
					;;
			esac
		fi
	fi

	# Ideally we want to use curl, but on some installs we
	# only have wget. Detect and use what's available.
	CURL=
	if type curl >/dev/null; then
		CURL="curl -fsSL"
	elif type wget >/dev/null; then
		CURL="wget -q -O-"
	fi
	if [ -z "$CURL" ]; then
		echo "The installer needs either curl or wget to download files."
		echo "Please install either curl or wget to proceed."
		exit 1
	fi

	TEST_URL="https://pkgs.lanhc.com/"
	RC=0
	TEST_OUT=$($CURL "$TEST_URL" 2>&1) || RC=$?
	if [ $RC != 0 ]; then
		echo "The installer cannot reach $TEST_URL"
		echo "Please make sure that your machine has internet access."
		echo "Test output:"
		echo $TEST_OUT
		exit 1
	fi

	# Step 2: having detected an OS we support, is it one of the
	# versions we support?
	OS_UNSUPPORTED=
	case "$OS" in
		ubuntu|debian|raspbian|centos|oracle|rhel|amazon-linux|opensuse|photon)
			# Check with the package server whether a given version is supported.
			URL="https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/installer-supported"
			$CURL "$URL" 2> /dev/null | grep -q OK || OS_UNSUPPORTED=1
			;;
		fedora)
			# All versions supported, no version checking required.
			;;
		arch)
			# Rolling release, no version checking needed.
			;;
		manjaro)
			# Rolling release, no version checking needed.
			;;
		alpine)
			# All versions supported, no version checking needed.
			# TODO: is that true? When was lanhc packaged?
			;;
		void)
			# Rolling release, no version checking needed.
			;;
		gentoo)
			# Rolling release, no version checking needed.
			;;
		freebsd)
			if [ "$VERSION" != "12" ] && \
			   [ "$VERSION" != "13" ] && \
			   [ "$VERSION" != "14" ] && \
			   [ "$VERSION" != "15" ]
			then
				OS_UNSUPPORTED=1
			fi
			;;
		openbsd)
			OS_UNSUPPORTED=1
			;;
		macos)
			# We delegate macOS installation to the app store, it will
			# perform version checks for us.
			;;
		other-linux)
			OS_UNSUPPORTED=1
			;;
		*)
			OS_UNSUPPORTED=1
			;;
	esac
	if [ "$OS_UNSUPPORTED" = "1" ]; then
		case "$OS" in
			other-linux)
				echo "Couldn't determine what kind of Linux is running."
				echo "You could try the static binaries at:"
				echo "https://pkgs.lanhc.com/$TRACK/#static"
				;;
			"")
				echo "Couldn't determine what operating system you're running."
				;;
			*)
				echo "$OS $VERSION isn't supported by this script yet."
				;;
		esac
		echo
		echo "If you'd like us to support your system better, please email support@lanhc.com"
		echo "and tell us what OS you're running."
		echo
		echo "Please include the following information we gathered from your system:"
		echo
		echo "OS=$OS"
		echo "VERSION=$VERSION"
		echo "PACKAGETYPE=$PACKAGETYPE"
		if type uname >/dev/null 2>&1; then
			echo "UNAME=$(uname -a)"
		else
			echo "UNAME="
		fi
		echo
		if [ -f /etc/os-release ]; then
			cat /etc/os-release
		else
			echo "No /etc/os-release"
		fi
		exit 1
	fi

	# Step 3: work out if we can run privileged commands, and if so,
	# how.
	CAN_ROOT=
	SUDO=
	if [ "$(id -u)" = 0 ]; then
		CAN_ROOT=1
		SUDO=""
	elif type sudo >/dev/null; then
		CAN_ROOT=1
		SUDO="sudo"
	elif type doas >/dev/null; then
		CAN_ROOT=1
		SUDO="doas"
	fi
	if [ "$CAN_ROOT" != "1" ]; then
		echo "This installer needs to run commands as root."
		echo "We tried looking for 'sudo' and 'doas', but couldn't find them."
		echo "Either re-run this script as root, or set up sudo/doas."
		exit 1
	fi


	# Step 4: run the installation.
	OSVERSION="$OS"
	[ "$VERSION" != "" ] && OSVERSION="$OSVERSION $VERSION"

	# Prepare package name with optional version
	if [ -n "$LANHC_VERSION" ]; then
		echo "正在安装 蓝核AI智控台 $LANHC_VERSION（$OSVERSION，使用 $PACKAGETYPE）"
	else
		echo "正在安装 蓝核AI智控台（$OSVERSION，使用 $PACKAGETYPE）"
	fi
	case "$PACKAGETYPE" in
		apt)
			export DEBIAN_FRONTEND=noninteractive
			if [ "$APT_KEY_TYPE" = "legacy" ] && ! type gpg >/dev/null; then
				$SUDO apt-get update
				$SUDO apt-get install -y gnupg
			fi

			set -x
			$SUDO mkdir -p --mode=0755 /usr/share/keyrings
			case "$APT_KEY_TYPE" in
				legacy)
					$CURL "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION.asc" | $SUDO apt-key add -
					$CURL "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION.list" | $SUDO tee /etc/apt/sources.list.d/lanhc.list
					$SUDO chmod 0644 /etc/apt/sources.list.d/lanhc.list
				;;
				keyring)
					$CURL "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION.noarmor.gpg" | $SUDO tee /usr/share/keyrings/lanhc-archive-keyring.gpg >/dev/null
					$SUDO chmod 0644 /usr/share/keyrings/lanhc-archive-keyring.gpg
					$CURL "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION.lanhc-keyring.list" | $SUDO tee /etc/apt/sources.list.d/lanhc.list
					$SUDO chmod 0644 /etc/apt/sources.list.d/lanhc.list
				;;
			esac
			$SUDO apt-get update
			if [ -n "$LANHC_VERSION" ]; then
				$SUDO apt-get install -y "lanhc=$LANHC_VERSION" lanhc-archive-keyring
			else
				$SUDO apt-get install -y lanhc lanhc-archive-keyring
			fi
			if [ "$APT_SYSTEMCTL_START" = "true" ]; then
				$SUDO systemctl enable --now lanhcd
				$SUDO systemctl start lanhcd
			fi
			set +x
		;;
		yum)
			set -x
			$SUDO yum install yum-utils -y
			$SUDO yum-config-manager -y --add-repo "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/lanhc.repo"
			if [ -n "$LANHC_VERSION" ]; then
				$SUDO yum install "lanhc-$LANHC_VERSION" -y
			else
				$SUDO yum install lanhc -y
			fi
			$SUDO systemctl enable --now lanhcd
			set +x
		;;
		dnf)
			# DNF 5 has a different argument format; determine which one we have.
			DNF_VERSION="3"
			if LANG=C.UTF-8 dnf --version | grep -q '^dnf5 version'; then
				DNF_VERSION="5"
			fi

			# The 'config-manager' plugin wasn't implemented when
			# DNF5 was released; detect that and use the old
			# version if necessary.
			if [ "$DNF_VERSION" = "5" ]; then
				set -x
				$SUDO dnf install -y 'dnf-command(config-manager)' && DNF_HAVE_CONFIG_MANAGER=1 || DNF_HAVE_CONFIG_MANAGER=0
				set +x

				if [ "$DNF_HAVE_CONFIG_MANAGER" != "1" ]; then
					if type dnf-3 >/dev/null; then
						DNF_VERSION="3"
					else
						echo "dnf 5 detected, but 'dnf-command(config-manager)' not available and dnf-3 not found"
						exit 1
					fi
				fi
			fi

			set -x
			if [ "$DNF_VERSION" = "3" ]; then
				$SUDO dnf install -y 'dnf-command(config-manager)'
				$SUDO dnf config-manager --add-repo "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/lanhc.repo"
			elif [ "$DNF_VERSION" = "5" ]; then
				# Already installed config-manager, above.
				$SUDO dnf config-manager addrepo --overwrite --from-repofile="https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/lanhc.repo"
			else
				echo "unexpected: unknown dnf version $DNF_VERSION"
				exit 1
			fi
			if [ -n "$LANHC_VERSION" ]; then
				$SUDO dnf install -y "lanhc-$LANHC_VERSION"
			else
				$SUDO dnf install -y lanhc
			fi
			$SUDO systemctl enable --now lanhcd
			set +x
		;;
		tdnf)
			set -x
			curl -fsSL "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/lanhc.repo" > /etc/yum.repos.d/lanhc.repo
			if [ -n "$LANHC_VERSION" ]; then
				$SUDO tdnf install -y "lanhc-$LANHC_VERSION"
			else
				$SUDO tdnf install -y lanhc
			fi
			$SUDO systemctl enable --now lanhcd
			set +x
		;;
		zypper)
			set -x
			$SUDO rpm --import "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/repo.gpg"
			$SUDO zypper --non-interactive ar -g -r "https://pkgs.lanhc.com/$TRACK/$OS/$VERSION/lanhc.repo"
			$SUDO zypper --non-interactive --gpg-auto-import-keys refresh
			if [ -n "$LANHC_VERSION" ]; then
				$SUDO zypper --non-interactive install "lanhc=$LANHC_VERSION"
			else
				$SUDO zypper --non-interactive install lanhc
			fi
			$SUDO systemctl enable --now lanhcd
			set +x
			;;
		pacman)
			set -x
			if [ -n "$LANHC_VERSION" ]; then
				echo "警告：Arch Linux 维护自己的 蓝核AI智控台 软件包。由于目标版本可能不再可用，版本锁定可能无法按预期工作。"
				$SUDO pacman -S "lanhc=$LANHC_VERSION" --noconfirm
			else
				$SUDO pacman -S lanhc --noconfirm
			fi
			$SUDO systemctl enable --now lanhcd
			set +x
			;;
		pkg)
			set -x
			if [ -n "$LANHC_VERSION" ]; then
				echo "警告：FreeBSD 维护自己的 蓝核AI智控台 软件包。由于目标版本可能不再可用，版本锁定可能无法按预期工作。"
				$SUDO pkg install --yes "lanhc-$LANHC_VERSION"
			else
				$SUDO pkg install --yes lanhc
			fi
			$SUDO service lanhcd enable
			$SUDO service lanhcd start
			set +x
			;;
		apk)
			set -x
			if ! grep -Eq '^http.*/community$' /etc/apk/repositories; then
				if type setup-apkrepos >/dev/null; then
					$SUDO setup-apkrepos -c -1
				else
					echo "installing lanhc requires the community repo to be enabled in /etc/apk/repositories"
					exit 1
				fi
			fi
			if [ -n "$LANHC_VERSION" ]; then
				echo "警告：Alpine Linux 维护自己的 蓝核AI智控台 软件包。由于目标版本可能不再可用，版本锁定可能无法按预期工作。"
				$SUDO apk add "lanhc=$LANHC_VERSION"
			else
				$SUDO apk add lanhc
			fi
			$SUDO rc-update add lanhc
			$SUDO rc-service lanhc start
			set +x
			;;
		xbps)
			set -x
			if [ -n "$LANHC_VERSION" ]; then
				echo "警告：Void Linux 维护自己的 蓝核AI智控台 软件包。由于目标版本可能不再可用，版本锁定可能无法按预期工作。"
				$SUDO xbps-install "lanhc-$LANHC_VERSION" -y
			else
				$SUDO xbps-install lanhc -y
			fi
			set +x
			;;
		emerge)
			set -x
			if [ -n "$LANHC_VERSION" ]; then
				echo "警告：Gentoo 维护自己的 蓝核AI智控台 软件包。由于目标版本可能不再可用，版本锁定可能无法按预期工作。"
				$SUDO emerge --ask=n "=net-vpn/lanhc-$LANHC_VERSION"
			else
				$SUDO emerge --ask=n net-vpn/lanhc
			fi
			set +x
			;;
		appstore)
			set -x
			open "https://apps.apple.com/us/app/lanhc/id1475387142"
			set +x
			;;
		*)
			echo "unexpected: unknown package type $PACKAGETYPE"
			exit 1
			;;
	esac

	echo "安装完成！运行以下命令登录并开始使用蓝核AI智控台："
	echo
	if [ -z "$SUDO" ]; then
		echo "lanhc up"
	else
		echo "$SUDO lanhc up"
	fi
}

main
