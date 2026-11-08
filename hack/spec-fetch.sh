#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: spec-fetch.sh OUT
set -euo pipefail

tmp=$(mktemp "$1.XXXXXX")
trap 'rm -f "$tmp"' EXIT
curl -fsSL https://app.opencve.io/api/v2/schema/ > "$tmp"
mv "$tmp" "$1"
