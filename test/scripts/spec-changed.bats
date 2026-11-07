#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

setup() {
  ROOT=$(cd "$BATS_TEST_DIRNAME/../.." && pwd)
  cd "$BATS_TEST_TMPDIR"
}

schema() {
  printf "openapi: 3.0.3\ninfo:\n  title: OpenCVE API v2\n  version: '%s'\npaths: {%s}\n" "$1" "$2"
}

@test "reports unchanged when the fresh schema matches the pinned one" {
  schema 2026-06-24 > pinned.yaml
  schema 2026-06-24 > fresh.yaml
  run "$ROOT/hack/spec-changed.sh" pinned.yaml fresh.yaml
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf 'changed=false\nold_version=2026-06-24\nnew_version=2026-06-24')" ]
}

@test "reports changed when the content differs under the same version" {
  schema 2026-06-24 > pinned.yaml
  schema 2026-06-24 '/api/v2/cves: {}' > fresh.yaml
  run "$ROOT/hack/spec-changed.sh" pinned.yaml fresh.yaml
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf 'changed=true\nold_version=2026-06-24\nnew_version=2026-06-24')" ]
}

@test "reports changed with both versions when the version moves" {
  schema 2026-06-24 > pinned.yaml
  schema 2026-11-01 > fresh.yaml
  run "$ROOT/hack/spec-changed.sh" pinned.yaml fresh.yaml
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf 'changed=true\nold_version=2026-06-24\nnew_version=2026-11-01')" ]
}

@test "fails without reporting when an input is missing" {
  schema 2026-06-24 > pinned.yaml
  run "$ROOT/hack/spec-changed.sh" pinned.yaml missing.yaml
  [ "$status" -ne 0 ]
  [[ "$output" != *changed=* ]]
}

@test "fails when a version cannot be read" {
  schema 2026-06-24 > pinned.yaml
  printf 'info: [\n' > fresh.yaml
  run "$ROOT/hack/spec-changed.sh" pinned.yaml fresh.yaml
  [ "$status" -ne 0 ]
}
