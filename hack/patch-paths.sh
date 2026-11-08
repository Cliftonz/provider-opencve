#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: patch-paths.sh PATCH
set -euo pipefail

allowed='^(openapi/|package/crds/|internal/clients/opencve/client\.gen\.go$|(.*/)?zz_generated[^/]*$)'
if outside=$(sed -n 's|^diff --git a/\(.*\) b/\(.*\)$|\1\n\2|p' "$1" | sort -u | grep -Ev "$allowed"); then
  echo "patch touches paths outside the generated set:"
  echo "$outside"
  exit 1
fi
