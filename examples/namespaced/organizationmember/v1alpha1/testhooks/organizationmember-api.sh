#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=organizationmember.organizationmember.opencve.m.clifton.quest/example-organizationmember
object() {
  get "/members?page_size=100" | jq --argjson id "$(at id)" 'first((.results // [])[] | select(.id == $id)) // null'
}
CREATED='.email == "example-organizationmember@example.com" and .role == "member"'
assert_api "$@"
