#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=projectmember.projectmember.opencve.m.clifton.quest/example-projectmember
object() {
  get "/projects/example-projectmember-project/members?page_size=100" | jq --argjson id "$(at id)" 'first((.results // [])[] | select(.id == $id)) // null'
}
CREATED='.role == "viewer"'
UPDATED='.role == "contributor"'
assert_api "$@"
