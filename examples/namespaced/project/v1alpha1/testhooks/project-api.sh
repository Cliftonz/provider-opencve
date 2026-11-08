#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=project.project.opencve.m.clifton.quest/example-project
object() {
  get /projects/example-project
}
CREATED='.description == "managed by provider-opencve"'
UPDATED='.description == "updated by uptest"'
assert_api "$@"
