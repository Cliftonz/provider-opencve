#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: breaking-changes.sh CRDDIFF
set -euo pipefail

crddiff=$1
release=$(gh release list --json tagName,isLatest --jq '.[] | select(.isLatest) | .tagName')
if [ -z "$release" ]; then
  echo "no release yet; skipping the CRD breaking-change check" >&2
  exit 0
fi

base=$(mktemp -d)
trap 'rm -rf "$base"' EXIT
git archive "$release" package/crds | tar -x -C "$base"
for old in "$base"/package/crds/*.yaml; do
  "$crddiff" revision "$old" "package/crds/${old##*/}"
done
