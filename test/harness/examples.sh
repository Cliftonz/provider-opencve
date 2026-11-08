#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

find examples/namespaced -path '*/v1alpha1/*.yaml' ! -exec grep -q upjet.upbound.io/manual-intervention {} \; -print | sort |
  if [ "${E2E_TARGET:-kind}" = saas ]; then cat; else sed '\#/projectmember/#d'; fi
