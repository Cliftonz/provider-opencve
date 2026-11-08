#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

load helpers

setup() {
  setup_stubs
  printf '#!/usr/bin/env bash\necho "$(basename "$0") $*" >> "%s"\n' "$CALLS" > "$BATS_TEST_TMPDIR/recorder"
  stub curl "cp '$BATS_TEST_TMPDIR/recorder' \"\$2\""
  cd "$BATS_TEST_TMPDIR"
  echo 'CROSSPLANE_CLI_VERSION = v2.2.1' > Makefile
  mkdir -p xpkg/linux_amd64 xpkg/linux_arm64
  touch xpkg/linux_amd64/provider-opencve-v0.1.0.xpkg xpkg/linux_arm64/provider-opencve-v0.1.0.xpkg
}

@test "skips the push when the tag exists" {
  stub docker
  run "$ROOT/hack/push-xpkg.sh" registry.example/org/repo:v0.1.0
  [ "$status" -eq 0 ]
  [ "$output" = "registry.example/org/repo:v0.1.0 exists" ]
  ! grep -q '^crossplane ' "$CALLS"
}

@test "pushes both platforms when the tag is not found" {
  stub docker 'echo "ERROR: registry.example/org/repo:v0.1.0: not found"; exit 1'
  run "$ROOT/hack/push-xpkg.sh" registry.example/org/repo:v0.1.0
  [ "$status" -eq 0 ]
  grep -qF "curl -fsSLo" "$CALLS"
  grep -qF "https://releases.crossplane.io/stable/v2.2.1/bin/linux_amd64/crank" "$CALLS"
  grep -qxF "crossplane xpkg push --package-files xpkg/linux_amd64/provider-opencve-v0.1.0.xpkg --package-files xpkg/linux_arm64/provider-opencve-v0.1.0.xpkg registry.example/org/repo:v0.1.0" "$CALLS"
}

@test "fails without pushing when the lookup fails for another reason" {
  stub docker 'echo "ERROR: unauthorized"; exit 1'
  run "$ROOT/hack/push-xpkg.sh" registry.example/org/repo:v0.1.0
  [ "$status" -eq 1 ]
  [ "$output" = "ERROR: unauthorized" ]
  ! grep -q '^crossplane ' "$CALLS"
}
