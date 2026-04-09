#!/usr/bin/env bash
# Build a single-file AppImage for the Fyne GUI (linuxdeploy + appimagetool).
# Usage: build-appimage.sh <amd64|arm64> <repo_root> <out_dir> [app_version]
#
# Requires: Linux host arch matching target, Go + CGO deps, curl or wget.
# Optional: librsvg2-bin (rsvg-convert) if icons/v2ray-subscription-monitor.png is missing.

set -euo pipefail

ARCH="${1:?first arg: amd64 or arm64}"
REPO_ROOT="$(cd "${2:?repo root}" && pwd)"
OUT_DIR_ARG="${3:?out directory}"
mkdir -p "$OUT_DIR_ARG"
OUT_DIR="$(cd "$OUT_DIR_ARG" && pwd)"
APP_VERSION="${4:-dev}"

APP="v2ray-subscription-monitor"
DESKTOP_FILE="io.github.v2ray-subscription-data-usage-monitor.desktop"
HOST="$(uname -m)"

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "missing command: $1" >&2
		exit 1
	}
}

if [[ "$ARCH" == "amd64" ]]; then
	if [[ "$HOST" != "x86_64" ]]; then
		echo "linux/amd64 AppImage must be built on an x86_64 Linux host (found: $HOST)." >&2
		exit 1
	fi
	APPIMAGE_ARCH="x86_64"
	LINUXDEPLOY_NAME="linuxdeploy-x86_64.AppImage"
	APPIMAGETOOL_NAME="appimagetool-x86_64.AppImage"
elif [[ "$ARCH" == "arm64" ]]; then
	if [[ "$HOST" != "aarch64" ]]; then
		echo "linux/arm64 AppImage must be built on an aarch64 Linux host (found: $HOST)." >&2
		exit 1
	fi
	APPIMAGE_ARCH="aarch64"
	LINUXDEPLOY_NAME="linuxdeploy-aarch64.AppImage"
	APPIMAGETOOL_NAME="appimagetool-aarch64.AppImage"
else
	echo "unsupported arch: $ARCH (use amd64 or arm64)" >&2
	exit 1
fi

need_cmd go

CACHE="${REPO_ROOT}/.cache/appimage-tools"
mkdir -p "$CACHE"

fetch() {
	local url="$1" dest="$2"
	if [[ -f "$dest" ]]; then
		return 0
	fi
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$dest" "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$dest" "$url"
	else
		echo "need curl or wget to download $url" >&2
		exit 1
	fi
	chmod +x "$dest"
}

LINUXDEPLOY_URL="https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/${LINUXDEPLOY_NAME}"
APPIMAGETOOL_URL="https://github.com/AppImage/AppImageKit/releases/download/continuous/${APPIMAGETOOL_NAME}"

LINUXDEPLOY_BIN="${CACHE}/${LINUXDEPLOY_NAME}"
APPIMAGETOOL_BIN="${CACHE}/${APPIMAGETOOL_NAME}"

fetch "$LINUXDEPLOY_URL" "$LINUXDEPLOY_BIN"
fetch "$APPIMAGETOOL_URL" "$APPIMAGETOOL_BIN"

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/appimage-build.XXXXXX")"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

APPDIR="${WORKDIR}/AppDir"
mkdir -p "${APPDIR}/usr/bin"
mkdir -p "${APPDIR}/usr/share/applications"

ICON_SRC="${WORKDIR}/v2ray-subscription-monitor.png"
if [[ -f "${REPO_ROOT}/assets/icons/v2ray-subscription-monitor.png" ]]; then
	cp "${REPO_ROOT}/assets/icons/v2ray-subscription-monitor.png" "$ICON_SRC"
elif command -v rsvg-convert >/dev/null 2>&1; then
	rsvg-convert -w 512 -h 512 "${REPO_ROOT}/assets/icons/v2ray.svg" -o "$ICON_SRC"
else
	echo "Add assets/icons/v2ray-subscription-monitor.png or install rsvg-convert (librsvg2-bin)." >&2
	exit 1
fi

cp "${REPO_ROOT}/installer/linux/${DESKTOP_FILE}" "${APPDIR}/usr/share/applications/${DESKTOP_FILE}"

cd "$REPO_ROOT"
export CGO_ENABLED=1
export CGO_CFLAGS=-Os
export GOOS=linux
export GOARCH="$ARCH"
go build -buildvcs=false -trimpath -ldflags="-s -w -X main.appVersion=${APP_VERSION}" -o "${APPDIR}/usr/bin/${APP}" .

export APPIMAGE_EXTRACT_AND_RUN=1

# Bundle dynamically linked deps (GLFW, X11, libGL, etc.) into the AppDir.
"$LINUXDEPLOY_BIN" \
	--appdir "$APPDIR" \
	--executable "${APPDIR}/usr/bin/${APP}" \
	--desktop-file "${APPDIR}/usr/share/applications/${DESKTOP_FILE}" \
	--icon-file "$ICON_SRC"

# Optional gnuTLS / openssl for HTTPS - the Go binary is mostly static but may dlopen; linuxdeploy traces ELF NEEDED.
export ARCH="$APPIMAGE_ARCH"
export VERSION="$APP_VERSION"

OUT_FILE="${OUT_DIR}/${APP}-linux-${ARCH}.AppImage"
rm -f "$OUT_FILE"

"$APPIMAGETOOL_BIN" "$APPDIR" "$OUT_FILE"

chmod +x "$OUT_FILE"
echo "Wrote $OUT_FILE"
