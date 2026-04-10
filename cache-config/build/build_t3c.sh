#!/bin/bash

# Script to compile t3c packages using go build
# This script assumes it's run from the Traffic Control project root directory
# Usage: ./cache-config/build/build_t3c.sh [package_name | all]
# If no argument or 'all', builds all t3c packages
# Otherwise, builds the specified package (e.g., t3c-apply)

set -e  # Exit on any error

# Set environment variables
TC_DIR=$(pwd)
T3C_DIR="$TC_DIR/cache-config"
DIST_DIR="$TC_DIR/dist"
VERSION=$(cat "$TC_DIR/VERSION")
GIT_REV=$(git rev-parse HEAD)
BUILD_TIME=$(date +'%Y-%M-%dT%H:%M:%s')

# Default packages to build (all t3c-* except t3cutil which is a library)
ALL_PACKAGES=("t3c" "t3c-apply" "t3c-check" "t3c-check-refs" "t3c-check-reload" "t3c-diff" "t3c-generate" "t3c-preprocess" "t3c-request" "t3c-update")

# Determine which packages to build
if [ $# -eq 0 ] || [ "$1" = "all" ]; then
    PACKAGES=("${ALL_PACKAGES[@]}")
else
    PACKAGES=("$1")
fi

echo "Building t3c packages: ${PACKAGES[*]}"
echo "Version: $VERSION"
echo "Git Revision: $GIT_REV"
echo "Build Time: $BUILD_TIME"

# Create dist directory if it doesn't exist
mkdir -p "$DIST_DIR"

# Vendor dependencies once (from project root)
echo "Vendoring dependencies..."
go mod vendor -v

# Build each package
for pkg in "${PACKAGES[@]}"; do
    pkg_dir="$T3C_DIR/$pkg"
    if [ ! -d "$pkg_dir" ]; then
        echo "Warning: Directory $pkg_dir does not exist. Skipping $pkg."
        continue
    fi

    echo "Building $pkg..."

    # Change to the package directory
    cd "$pkg_dir"

    # Build the binary
    go build -v \
      -gcflags "all=-N -l" \
      -ldflags "-s -w \
        -X main.GitRevision=$GIT_REV \
        -X main.BuildTimestamp=$BUILD_TIME \
        -X main.Version=$VERSION" \
      -tags "osusergo netgo"

    # Move binary to dist directory
    if [ -f "$pkg" ]; then
        mv "$pkg" "$DIST_DIR/"
        echo "Binary $pkg moved to $DIST_DIR/"
    else
        echo "Warning: Binary $pkg not found after build."
    fi
done

echo "Build completed successfully. Binaries in: $DIST_DIR/"