#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

load helpers

setup() {
  setup_stubs
  cd "$BATS_TEST_TMPDIR"
  git init -q
  mkdir -p package/crds
  crd 'name: {type: string}' > package/crds/widgets.yaml
  git add package
  git -c user.name=test -c user.email=test@example.org commit -qm release
  git tag v0.1.0
}

crd() {
  printf '%s\n' \
    'apiVersion: apiextensions.k8s.io/v1' \
    'kind: CustomResourceDefinition' \
    'metadata:' \
    '  name: widgets.example.org' \
    'spec:' \
    '  group: example.org' \
    '  names: {kind: Widget, plural: widgets}' \
    '  scope: Namespaced' \
    '  versions:' \
    '  - name: v1alpha1' \
    '    served: true' \
    '    storage: true' \
    '    schema:' \
    '      openAPIV3Schema:' \
    '        type: object' \
    '        properties:' \
    '          spec:' \
    '            type: object' \
    '            properties:'
  for field in "$@"; do
    echo "              $field"
  done
}

@test "skips the check when there is no release yet" {
  stub gh
  run "$ROOT/hack/breaking-changes.sh" "$CRDDIFF"
  [ "$status" -eq 0 ]
  [ "$output" = "no release yet; skipping the CRD breaking-change check" ]
}

@test "passes when a CRD only gains an optional field" {
  stub gh 'echo v0.1.0'
  crd 'name: {type: string}' 'description: {type: string}' > package/crds/widgets.yaml
  run "$ROOT/hack/breaking-changes.sh" "$CRDDIFF"
  [ "$status" -eq 0 ]
}

@test "fails when a field is removed since the latest release" {
  stub gh 'echo v0.1.0'
  crd 'description: {type: string}' > package/crds/widgets.yaml
  run "$ROOT/hack/breaking-changes.sh" "$CRDDIFF"
  [ "$status" -ne 0 ]
  [[ "$output" == *name* ]]
}

@test "fails when a CRD is removed since the latest release" {
  stub gh 'echo v0.1.0'
  rm package/crds/widgets.yaml
  run "$ROOT/hack/breaking-changes.sh" "$CRDDIFF"
  [ "$status" -ne 0 ]
  [[ "$output" == *widgets.yaml* ]]
}
