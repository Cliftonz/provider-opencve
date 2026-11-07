#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

load helpers

setup() {
  setup_stubs
  export GH_TOKEN=token GITHUB_REPOSITORY=owner/repo
  export GIT_CONFIG_GLOBAL=$BATS_TEST_TMPDIR/gitconfig
  git config --global user.name test
  git config --global user.email test@example.org
  git config --global url."$BATS_TEST_TMPDIR/remote.git".insteadOf https://x-access-token:token@github.com/owner/repo.git
  git init -q --bare "$BATS_TEST_TMPDIR/remote.git"
  git init -q -b main "$BATS_TEST_TMPDIR/repo"
  cd "$BATS_TEST_TMPDIR/repo"
  mkdir openapi
  echo old > openapi/opencve-v2.yaml
  git add openapi
  git commit -qm init
  echo new > openapi/opencve-v2.yaml
  git diff --binary > "$BATS_TEST_TMPDIR/changes.patch"
  git checkout -q openapi/opencve-v2.yaml
}

open_prs() {
  echo "$1" > "$BATS_TEST_TMPDIR/prs.json"
  stub gh "case \"\$1 \$2\" in
    'pr list') cat '$BATS_TEST_TMPDIR/prs.json' ;;
    'pr create') echo https://github.com/owner/repo/pull/9 ;;
  esac"
}

@test "pushes the spec branch and opens a PR when none is open" {
  open_prs '[]'
  run "$ROOT/hack/open-pr.sh" 2026-11-01 "$BATS_TEST_TMPDIR/changes.patch" "schema body"
  [ "$status" -eq 0 ]
  [ "$(git -C "$BATS_TEST_TMPDIR/remote.git" show spec/2026-11-01:openapi/opencve-v2.yaml)" = new ]
  grep -qxF "gh pr create --base main --head spec/2026-11-01 --title feat: track OpenCVE schema 2026-11-01 --body schema body" "$CALLS"
  ! grep -q "pr merge\|pr close" "$CALLS"
}

@test "force-updates the spec branch without opening a PR when one is open" {
  git switch -qc spec/2026-11-01
  echo stale > openapi/opencve-v2.yaml
  git commit -qam stale
  git push -q https://x-access-token:token@github.com/owner/repo.git spec/2026-11-01
  git switch -q main
  open_prs '[{"number":5,"headRefName":"spec/2026-11-01","url":"https://github.com/owner/repo/pull/5"}]'
  run "$ROOT/hack/open-pr.sh" 2026-11-01 "$BATS_TEST_TMPDIR/changes.patch" "schema body"
  [ "$status" -eq 0 ]
  [ "$(git -C "$BATS_TEST_TMPDIR/remote.git" show spec/2026-11-01:openapi/opencve-v2.yaml)" = new ]
  ! grep -q "pr create\|pr close" "$CALLS"
}

@test "closes superseded spec PRs with a link to the current one, leaving other PRs open" {
  open_prs '[{"number":3,"headRefName":"spec/2026-06-24","url":"https://github.com/owner/repo/pull/3"},
    {"number":4,"headRefName":"feature/x","url":"https://github.com/owner/repo/pull/4"}]'
  run "$ROOT/hack/open-pr.sh" 2026-11-01 "$BATS_TEST_TMPDIR/changes.patch" "schema body"
  [ "$status" -eq 0 ]
  grep -qxF "gh pr close 3 --comment Superseded by https://github.com/owner/repo/pull/9" "$CALLS"
  ! grep -q "pr close 4" "$CALLS"
}
