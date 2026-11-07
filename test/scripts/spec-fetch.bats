#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

setup() {
  ROOT=$(cd "$BATS_TEST_DIRNAME/../.." && pwd)
  mkdir "$BATS_TEST_TMPDIR/bin"
  PATH=$BATS_TEST_TMPDIR/bin:$PATH
}

stub_curl() {
  printf '#!/usr/bin/env bash\necho "$*" > "%s/curl-args"\n%s\n' "$BATS_TEST_TMPDIR" "$1" > "$BATS_TEST_TMPDIR/bin/curl"
  chmod +x "$BATS_TEST_TMPDIR/bin/curl"
}

@test "writes the OpenCVE v2 schema to the given path" {
  stub_curl "printf 'openapi: 3.0.3\n'"
  run "$ROOT/hack/spec-fetch.sh" "$BATS_TEST_TMPDIR/schema.yaml"
  [ "$status" -eq 0 ]
  [ "$(cat "$BATS_TEST_TMPDIR/schema.yaml")" = "openapi: 3.0.3" ]
  grep -qF https://app.opencve.io/api/v2/schema/ "$BATS_TEST_TMPDIR/curl-args"
}

@test "fails when the download fails, leaving the existing file untouched and no temp file" {
  printf 'openapi: 3.0.3\n' > "$BATS_TEST_TMPDIR/schema.yaml"
  stub_curl "printf partial; exit 22"
  run "$ROOT/hack/spec-fetch.sh" "$BATS_TEST_TMPDIR/schema.yaml"
  [ "$status" -eq 22 ]
  [ "$(cat "$BATS_TEST_TMPDIR/schema.yaml")" = "openapi: 3.0.3" ]
  [ -z "$(compgen -G "$BATS_TEST_TMPDIR/schema.yaml.*")" ]
}
