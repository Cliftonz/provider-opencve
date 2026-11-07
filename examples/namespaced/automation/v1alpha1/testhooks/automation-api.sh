#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=automation.automation.opencve.m.clifton.quest/example-automation
object() {
  get /projects/example-automation-project/automations/example-automation
}
CREATED='.trigger_type == "report" and .frequency == "weekly" and .schedule_weekday == "monday" and .configuration.actions == [{"type": "generate_report", "value": true}]'
UPDATED='.schedule_weekday == "friday"'
assert_api "$@"
