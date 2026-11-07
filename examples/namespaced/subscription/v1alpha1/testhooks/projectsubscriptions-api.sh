#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

. "$(dirname "$0")/../../../testhooks/opencve.sh"
MR=projectsubscriptions.subscription.opencve.m.clifton.quest/example-projectsubscriptions
object() {
  get /projects/example-subscriptions-project
}
CREATED='.subscriptions == {"vendors": ["apache"], "products": {"microsoft": ["windows"]}}'
UPDATED='.subscriptions.vendors == ["apache", "linux"]'
DELETED='. == null or .subscriptions == {"vendors": [], "products": {}}'
assert_api "$@"
if [ "${1:-created}" = created ]; then
  ${KUBECTL} -n default patch "$MR" --type=merge -p '{"spec":{"forProvider":{"vendors":["apache","linux"]}}}'
  until object | jq -e "$UPDATED"; do
    sleep 5
  done
fi
