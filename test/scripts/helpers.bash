# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

ROOT=$(cd "$BATS_TEST_DIRNAME/../.." && pwd)

setup_stubs() {
  mkdir -p "$BATS_TEST_TMPDIR/bin"
  PATH=$BATS_TEST_TMPDIR/bin:$PATH
  CALLS=$BATS_TEST_TMPDIR/calls
  : > "$CALLS"
}

# usage: stub NAME [BODY]
stub() {
  printf '#!/usr/bin/env bash\necho "%s $*" >> "%s"\n%s\n' "$1" "$CALLS" "${2:-}" > "$BATS_TEST_TMPDIR/bin/$1"
  chmod +x "$BATS_TEST_TMPDIR/bin/$1"
}
