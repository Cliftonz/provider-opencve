#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# usage: failure-issue.sh TITLE
set -euo pipefail

export TITLE=$1
body="Workflow run: ${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"

number=$(gh issue list --state open --search "\"${TITLE}\" in:title" --json number,title \
  --jq '.[] | select(.title == env.TITLE) | .number' | head -n1)
if [ -n "$number" ]; then
  gh issue comment "$number" --body "$body"
else
  gh issue create --title "$TITLE" --body "$body"
fi
