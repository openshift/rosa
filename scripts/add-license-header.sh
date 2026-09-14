#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0

# This script adds Apache 2.0 license headers to source files.
# Usage:
#   ./add-license-header.sh                    # Add headers to all files in repo
#   ./add-license-header.sh -check             # Check for missing headers
#   ./add-license-header.sh file1 file2 ...    # Add headers to specific files
#   ./add-license-header.sh -check file1 ...   # Check specific files

set -euo pipefail

ADDLICENSE="${ADDLICENSE:-addlicense}"

CHECK_MODE=""
FILES=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -check)
            CHECK_MODE="-check"
            shift
            ;;
        -*)
            echo "Unknown option: $1" >&2
            echo "Usage: $0 [-check] [files...]" >&2
            exit 1
            ;;
        *)
            FILES+=("$1")
            shift
            ;;
    esac
done

ADDLICENSE_ARGS=(
    ${CHECK_MODE:+"$CHECK_MODE"}
    -c "Red Hat"
    -l apache
    -y ""
    -s=only
)

if [[ ${#FILES[@]} -gt 0 ]]; then
    "$ADDLICENSE" "${ADDLICENSE_ARGS[@]}" "${FILES[@]}"
else
    "$ADDLICENSE" "${ADDLICENSE_ARGS[@]}" \
        -ignore "**/*.md" \
        -ignore "**/*.yaml" \
        -ignore "**/*.yml" \
        -ignore "**/*.toml" \
        -ignore "**/Dockerfile" \
        -ignore "**/*.Dockerfile" \
        -ignore "**/vendor/**" \
        -ignore "**/mock_*.go" \
        .
fi
