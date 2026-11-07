#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

set -aeuo pipefail

for ns in crossplane-system default; do
  ${KUBECTL} -n "$ns" create secret generic provider-secret --from-literal=credentials="${UPTEST_CLOUD_CREDENTIALS}" --dry-run=client -o yaml | ${KUBECTL} apply -f -
done

${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m
${KUBECTL} -n crossplane-system wait --for=condition=Available deployment --all --timeout=5m

cat <<EOF | ${KUBECTL} apply -f -
apiVersion: opencve.m.clifton.quest/v1beta1
kind: ClusterProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: provider-secret
      namespace: crossplane-system
      key: credentials
---
apiVersion: opencve.m.clifton.quest/v1beta1
kind: ProviderConfig
metadata:
  name: default
  namespace: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: provider-secret
      namespace: default
      key: credentials
EOF
