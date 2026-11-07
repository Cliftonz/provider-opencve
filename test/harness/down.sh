#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

cd "$(dirname "$0")"
kind delete cluster --name "${E2E_CLUSTER:-opencve-e2e}"
rm -f .out/credentials.json
