#!/usr/bin/env bash
set -euo pipefail

# Installs Blender for the STP renderer worker.
#
# Usage:
#   sudo bash install_blender.sh
#   BLENDER_SERIES=4.5 sudo bash install_blender.sh
#   BLENDER_VERSION=4.5.0 sudo bash install_blender.sh
#   BLENDER_URL=https://download.blender.org/release/Blender4.5/blender-4.5.0-linux-x64.tar.xz sudo bash install_blender.sh
#
# After installation, configure the worker with:
#   BLENDER_BIN=/opt/blender/blender

BLENDER_SERIES="${BLENDER_SERIES:-4.5}"
INSTALL_DIR="${INSTALL_DIR:-/opt/blender}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required. Install it first, for example: apt-get update && apt-get install -y curl"
  exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
  echo "tar is required. Install it first, for example: apt-get update && apt-get install -y tar xz-utils"
  exit 1
fi

TMP_DIR="$(mktemp -d)"
ARCHIVE_PATH="$TMP_DIR/blender.tar.xz"
EXTRACT_DIR="$TMP_DIR/extract"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if [ -z "${BLENDER_URL:-}" ]; then
  if [ -n "${BLENDER_VERSION:-}" ]; then
    BLENDER_URL="https://download.blender.org/release/Blender${BLENDER_SERIES}/blender-${BLENDER_VERSION}-linux-x64.tar.xz"
  else
    RELEASE_DIR="https://download.blender.org/release/Blender${BLENDER_SERIES}/"
    echo "Finding latest Linux x64 Blender archive in:"
    echo "  $RELEASE_DIR"
    ARCHIVE_NAME="$(
      curl -fsL "$RELEASE_DIR" |
        grep -oE "blender-[0-9]+\\.[0-9]+\\.[0-9]+-linux-x64\\.tar\\.xz" |
        sort -V |
        tail -n 1
    )"
    if [ -z "$ARCHIVE_NAME" ]; then
      echo "Could not auto-detect Blender archive. Set BLENDER_VERSION or BLENDER_URL explicitly."
      echo "Example:"
      echo "  BLENDER_VERSION=4.5.0 sudo bash install_blender.sh"
      exit 1
    fi
    BLENDER_URL="${RELEASE_DIR}${ARCHIVE_NAME}"
  fi
fi

echo "Downloading Blender from:"
echo "  $BLENDER_URL"
curl -fL --retry 3 --retry-delay 3 -o "$ARCHIVE_PATH" "$BLENDER_URL"

mkdir -p "$EXTRACT_DIR"
tar -xJf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

FOUND_DIR="$(find "$EXTRACT_DIR" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$FOUND_DIR" ] || [ ! -x "$FOUND_DIR/blender" ]; then
  echo "Blender executable was not found in the downloaded archive."
  exit 1
fi

TARGET_TMP="${INSTALL_DIR}.tmp"
TARGET_OLD="${INSTALL_DIR}.old"

rm -rf "$TARGET_TMP"
mkdir -p "$TARGET_TMP"
cp -a "$FOUND_DIR"/. "$TARGET_TMP"/

if [ -d "$INSTALL_DIR" ]; then
  rm -rf "$TARGET_OLD"
  mv "$INSTALL_DIR" "$TARGET_OLD"
fi
mv "$TARGET_TMP" "$INSTALL_DIR"

echo "Blender installed:"
if ! "$INSTALL_DIR/blender" --background --version; then
  echo
  echo "Blender was installed, but it cannot start because system libraries are missing."
  echo "Install the desktop/runtime libraries below, then run:"
  echo "  $INSTALL_DIR/blender --background --version"
  echo
  echo "RHEL/CentOS/Alibaba Cloud Linux:"
  echo "  yum install -y libXi libXrender libXfixes libXcursor libXinerama libXrandr libXxf86vm mesa-libGL mesa-libEGL mesa-dri-drivers libglvnd-glx libglvnd-egl libSM fontconfig freetype"
  echo
  echo "Ubuntu/Debian:"
  echo "  apt-get update && apt-get install -y libxi6 libxrender1 libxfixes3 libxcursor1 libxinerama1 libxrandr2 libxxf86vm1 libgl1 libegl1 libglx-mesa0 libgl1-mesa-dri libsm6 libfontconfig1 libfreetype6"
  exit 1
fi

echo
echo "Set this environment variable for the renderer worker:"
echo "  BLENDER_BIN=$INSTALL_DIR/blender"
