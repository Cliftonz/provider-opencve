#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: open-pr.sh SCHEMA_VERSION PATCH BODY
# Commits PATCH through the index only, so the checked-out scripts stay those of main.
set -euo pipefail

export BRANCH="spec/$1"
title="feat: track OpenCVE schema $1"

git switch -C "$BRANCH"
git apply --cached "$2"
git commit -m "$title"
git push --force "https://x-access-token:${GH_TOKEN}@github.com/${GITHUB_REPOSITORY}.git" "$BRANCH"

prs=$(gh pr list --state open --json number,headRefName,url)
url=$(jq -r '.[] | select(.headRefName == env.BRANCH) | .url' <<< "$prs")
if [ -z "$url" ]; then
  url=$(gh pr create --base main --head "$BRANCH" --title "$title" --body "$3")
fi
jq -r '.[] | select(.headRefName | startswith("spec/")) | select(.headRefName != env.BRANCH) | .number' <<< "$prs" |
  xargs -r -I{} gh pr close {} --comment "Superseded by ${url}"
