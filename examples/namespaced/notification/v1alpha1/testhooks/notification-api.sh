#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=notification.notification.opencve.m.clifton.quest/example-notification
object() {
  get /projects/example-notification-project/notifications/example-notification
}
CREATED='.type == "webhook" and .is_enabled == true and .configuration.host == "example.com" and .configuration.header_names == ["X-Source"]'
UPDATED='.is_enabled == false'
assert_api "$@"
