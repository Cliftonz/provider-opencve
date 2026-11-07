#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: push-xpkg.sh REGISTRY/ORG/REPO:TAG
set -euo pipefail

ref=$1
crossplane_version=$(sed -n 's/^CROSSPLANE_CLI_VERSION = //p' Makefile)
bin=$(mktemp -d)
curl -fsSLo "$bin/crossplane" "https://releases.crossplane.io/stable/${crossplane_version}/bin/linux_amd64/crank"
chmod +x "$bin/crossplane"

if out=$(docker buildx imagetools inspect "$ref" 2>&1); then
  echo "$ref exists"
  exit 0
fi
if ! grep -q 'not found' <<< "$out"; then
  echo "$out" >&2
  exit 1
fi
files=()
for f in xpkg/*/provider-opencve-"${ref##*:}".xpkg; do
  files+=(--package-files "$f")
done
"$bin/crossplane" xpkg push "${files[@]}" "$ref"
