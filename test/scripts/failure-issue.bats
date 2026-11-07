#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

load helpers

setup() {
  setup_stubs
  export GITHUB_SERVER_URL=https://github.com GITHUB_REPOSITORY=owner/repo GITHUB_RUN_ID=42
}

@test "opens an issue when none with the title is open" {
  stub gh
  run "$ROOT/hack/failure-issue.sh" "OpenCVE schema 2026-11-01: spec tracking"
  [ "$status" -eq 0 ]
  grep -qF "gh issue create --title OpenCVE schema 2026-11-01: spec tracking --body Workflow run: https://github.com/owner/repo/actions/runs/42" "$CALLS"
}

@test "comments on the open issue with the same title" {
  stub gh '[ "$1 $2" != "issue list" ] || echo 7'
  run "$ROOT/hack/failure-issue.sh" "OpenCVE schema 2026-11-01: spec tracking"
  [ "$status" -eq 0 ]
  grep -qF "gh issue comment 7 --body Workflow run: https://github.com/owner/repo/actions/runs/42" "$CALLS"
  ! grep -q "issue create" "$CALLS"
}
