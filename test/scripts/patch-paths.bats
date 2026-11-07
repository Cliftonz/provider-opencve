#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

load helpers

patch() {
  for path in "$@"; do
    printf 'diff --git a/%s b/%s\n' "$path" "$path"
  done > "$BATS_TEST_TMPDIR/changes.patch"
}

@test "accepts a patch limited to the schema, generated code and CRDs" {
  patch openapi/opencve-v2.yaml internal/clients/opencve/client.gen.go \
    apis/project/v1alpha1/zz_generated.deepcopy.go apis/zz_generated.managed.go \
    package/crds/project.opencve.m.clifton.quest_projects.yaml
  run "$ROOT/hack/patch-paths.sh" "$BATS_TEST_TMPDIR/changes.patch"
  [ "$status" -eq 0 ]
}

@test "refuses a patch touching other paths and names them" {
  patch openapi/opencve-v2.yaml hack/spec-fetch.sh .github/workflows/ci.yml internal/clients/opencve/client.go
  run "$ROOT/hack/patch-paths.sh" "$BATS_TEST_TMPDIR/changes.patch"
  [ "$status" -ne 0 ]
  [ "$output" = "$(printf 'patch touches paths outside the generated set:\n.github/workflows/ci.yml\nhack/spec-fetch.sh\ninternal/clients/opencve/client.go')" ]
}

@test "refuses a rename out of an allowed path" {
  printf 'diff --git a/package/crds/a.yaml b/hack/a.sh\n' > "$BATS_TEST_TMPDIR/changes.patch"
  run "$ROOT/hack/patch-paths.sh" "$BATS_TEST_TMPDIR/changes.patch"
  [ "$status" -ne 0 ]
  [[ "$output" == *hack/a.sh* ]]
}
