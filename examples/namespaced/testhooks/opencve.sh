#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

state="${TMPDIR:-/tmp}/opencve-e2e-$(basename "$0" .sh).json"
CREATED='. != null'
DELETED='. == null'

at() {
  jq -r ".$1" "$state"
}

get() {
  local response status
  response="$(curl -sS -w '\n%{http_code}' -H "Authorization: Bearer $OPENCVE_TOKEN" "$OPENCVE_URL/api/v2/organizations/$OPENCVE_ORGANIZATION$1")"
  status="${response##*$'\n'}"
  if [ "$status" = 404 ]; then
    echo null
  else
    test "$status" = 200
    printf '%s\n' "${response%$'\n'*}"
  fi
}

assert_api() {
  case "${1:-created}" in
    created)
      ${KUBECTL} -n default get "$MR" -o json | jq .status.atProvider > "$state"
      object | jq -e "$CREATED"
      ;;
    updated)
      object | jq -e "$UPDATED"
      ;;
    deleted)
      ${KUBECTL} -n default wait --for=delete "$MR" --timeout=5m
      until object | jq -e "$DELETED"; do
        sleep 5
      done
      ;;
  esac
}
