#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=cvetracker.cvetracker.opencve.m.clifton.quest/example-cvetracker
object() {
  get /projects/example-cvetracker-project/cves/CVE-2021-44228
}
CREATED='.tracker.status == "analysis_in_progress"'
UPDATED='.tracker.status == "resolved"'
assert_api "$@"
