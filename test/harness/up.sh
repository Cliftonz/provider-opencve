#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

cd "$(dirname "$0")"
set -a
# shellcheck source=images.env
. images.env
set +a

CLUSTER="${E2E_CLUSTER:-opencve-e2e}"
NAMESPACE=opencve
kc() { kubectl --context "kind-$CLUSTER" --namespace "$NAMESPACE" "$@"; }
credentials() {
  jq -n --arg token "$1" --arg organization "$2" --arg url "$3" '{$token, $organization, $url}' > .out/credentials.json
  printf 'OPENCVE_URL=%s\nOPENCVE_TOKEN=%s\nOPENCVE_ORGANIZATION=%s\n' "$4" "$1" "$2" > .out/opencve.env
}

mkdir -p .out
if ! kind get clusters | grep -qx "$CLUSTER"; then
  kind create cluster --name "$CLUSTER" --image "$KIND_NODE_IMAGE" --config kind.yaml
fi

if [ "${E2E_TARGET:-kind}" = saas ]; then
  url="${OPENCVE_URL:-https://app.opencve.io}"
  credentials "$OPENCVE_TOKEN" "$OPENCVE_ORGANIZATION" "$url" "$url"
  exit
fi

docker build --build-context "${PYTHON_IMAGE%@*}=docker-image://$PYTHON_IMAGE" --build-arg "OPENCVE_VERSION=$OPENCVE_VERSION" \
  -t "opencve-web:$OPENCVE_VERSION" "https://github.com/opencve/opencve.git#$OPENCVE_VERSION:web"
kind load docker-image --name "$CLUSTER" "opencve-web:$OPENCVE_VERSION"

kubectl --context "kind-$CLUSTER" create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl --context "kind-$CLUSTER" apply -f -
envsubst "$(sed -n 's/^\([A-Z_]*\)=.*/${\1}/p' images.env | tr '\n' ' ')" < manifests/opencve.yaml | kc apply -f -
kc wait --for=condition=Available deployment --all --timeout=15m

if ! kc exec deploy/webserver -- python manage.py shell -c 'from users.models import User; exit(not User.objects.filter(is_superuser=True).exists())'; then
  kc exec deploy/webserver -- python manage.py createsuperuser --noinput
fi
BOOTSTRAP="$(kc exec -i deploy/webserver -- python manage.py shell < bootstrap.py)"
credentials "$(sed -n 's/^OPENCVE_TOKEN=//p' <<< "$BOOTSTRAP")" "$(sed -n 's/^OPENCVE_ORGANIZATION=//p' <<< "$BOOTSTRAP")" \
  "http://$CLUSTER-control-plane:30900" http://localhost:30900
echo "wrote $(pwd)/.out/credentials.json and $(pwd)/.out/opencve.env"
