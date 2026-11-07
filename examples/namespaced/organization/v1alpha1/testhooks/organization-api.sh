#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=organization.organization.opencve.m.clifton.quest/example-organization
object() {
  get ""
}
CREATED='.name == env.OPENCVE_ORGANIZATION'
DELETED='.name == env.OPENCVE_ORGANIZATION'
assert_api "$@"
