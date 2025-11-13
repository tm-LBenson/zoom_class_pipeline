#!/usr/bin/env bash
set -e

REPO_URL="https://github.com/tm-LBenson/zoom_class_pipeline.git"
INSTALL_DIR="$HOME/zoom-recorder"
BINARY_NAME="zoom-recorder"

if ! command -v git >/dev/null 2>&1; then
  echo "git is required but not installed."
  echo "Install it from https://git-scm.com/downloads or with: brew install git"
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required but not installed."
  echo "Install it from https://go.dev/dl/ or with: brew install go"
  exit 1
fi

WORK_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t zoomrecorder)"
echo "Using temporary directory: $WORK_DIR"

git clone --depth 1 "$REPO_URL" "$WORK_DIR"

cd "$WORK_DIR"
go build -o "$BINARY_NAME"

mkdir -p "$INSTALL_DIR"
mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

cd /
rm -rf "$WORK_DIR"

echo
echo "Installed $BINARY_NAME to $INSTALL_DIR"
echo "Next steps:"
echo "1) Open Terminal"
echo "2) cd \"$INSTALL_DIR\""
echo "3) Run ./zoom-recorder to generate config.json and then edit it."
