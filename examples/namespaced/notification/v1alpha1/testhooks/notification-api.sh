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
CONFIGURED='.configuration.host == "example.org"'
UPDATED=".is_enabled == false and $CONFIGURED"
assert_api "$@"
if [ "${1:-created}" = created ]; then
  ${KUBECTL} -n default patch secret example-notification-configuration --type=merge -p '{"stringData":{"configuration":"{\"url\": \"https://example.org/opencve-webhook\", \"headers\": {\"X-Source\": \"provider-opencve\"}}"}}'
  until object | jq -e "$CONFIGURED"; do
    sleep 5
  done
  object | jq -e --arg id "$(at id)" '.id == $id'
fi
