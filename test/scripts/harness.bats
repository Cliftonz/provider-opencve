#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

setup() {
  ROOT=$(cd "$BATS_TEST_DIRNAME/../.." && pwd)
  cd "$BATS_TEST_TMPDIR"
}

stub() {
  mkdir -p bin
  printf '#!/usr/bin/env bash\n%s\n' "$2" > "bin/$1"
  chmod +x "bin/$1"
}

example() {
  mkdir -p "examples/namespaced/$1/v1alpha1"
  printf 'kind: X\nmetadata:\n  annotations:\n%s\n' "${3:-    a: b}" > "examples/namespaced/$1/v1alpha1/$2.yaml"
}

@test "saas mode writes the provider credentials from the environment without deploying OpenCVE" {
  cp -r "$ROOT/test/harness" harness
  stub kind 'exit 0'
  stub docker 'exit 1'
  stub kubectl 'exit 1'
  PATH="$PWD/bin:$PATH" E2E_TARGET=saas OPENCVE_TOKEN=opc_org.abc.def OPENCVE_ORGANIZATION=acme OPENCVE_URL=https://app.opencve.io \
    OPENCVE_MEMBERSHIP_ID=42 run harness/up.sh
  [ "$status" -eq 0 ]
  [ "$(jq -c . harness/.out/credentials.json)" = '{"token":"opc_org.abc.def","organization":"acme","url":"https://app.opencve.io"}' ]
  [ "$(cat harness/.out/opencve.env)" = "$(printf 'OPENCVE_URL=https://app.opencve.io\nOPENCVE_TOKEN=opc_org.abc.def\nOPENCVE_ORGANIZATION=acme\nOPENCVE_MEMBERSHIP_ID=42')" ]
}

@test "saas mode defaults the url to app.opencve.io" {
  cp -r "$ROOT/test/harness" harness
  stub kind 'exit 0'
  PATH="$PWD/bin:$PATH" E2E_TARGET=saas OPENCVE_TOKEN=t OPENCVE_ORGANIZATION=o OPENCVE_MEMBERSHIP_ID=1 run env -u OPENCVE_URL harness/up.sh
  [ "$status" -eq 0 ]
  [ "$(jq -r .url harness/.out/credentials.json)" = https://app.opencve.io ]
}

kind_up() {
  cp -r "$ROOT/test/harness" harness
  printf 'FROM base AS builder\n%s\nRUN python3 -m venv /app/venv\n' "$1" > upstream.Dockerfile
  stub kind 'exit 0'
  stub curl "echo \"\$*\" >> $PWD/curl.log; cat $PWD/upstream.Dockerfile"
  stub docker "if [ \"\$1\" = build ]; then echo \"\$*\" >> $PWD/docker.log; cat > $PWD/built.Dockerfile; fi"
  stub kubectl 'if [ "${*: -3}" = "apply -f -" ]; then cat > /dev/null; fi
if [ "${*: -1}" = shell ]; then printf "OPENCVE_TOKEN=t\nOPENCVE_ORGANIZATION=o\nOPENCVE_MEMBERSHIP_ID=7\n"; fi'
  PATH="$PWD/bin:$PATH" run harness/up.sh
}

@test "kind mode writes the membership id the bootstrap seeded" {
  kind_up 'RUN git clone --depth 1 -b "${OPENCVE_VERSION}" --single-branch "${OPENCVE_REPOSITORY}"'
  [ "$status" -eq 0 ]
  [ "$(cat harness/.out/opencve.env)" = "$(printf 'OPENCVE_URL=http://localhost:30900\nOPENCVE_TOKEN=t\nOPENCVE_ORGANIZATION=o\nOPENCVE_MEMBERSHIP_ID=7')" ]
}

@test "kind mode builds the upstream Dockerfile at the pinned commit with the clone checking out that commit" {
  kind_up 'RUN git clone --depth 1 -b "${OPENCVE_VERSION}" --single-branch "${OPENCVE_REPOSITORY}"'
  [ "$status" -eq 0 ]
  version=$(sed -n 's/^OPENCVE_VERSION=//p' harness/images.env)
  grep -qx "https://raw.githubusercontent.com/opencve/opencve/$version/web/Dockerfile" <(grep -o 'https://[^ ]*' curl.log)
  [ "$(cat built.Dockerfile)" = "$(printf '%s\n' 'FROM base AS builder' 'RUN git clone "${OPENCVE_REPOSITORY}" && git -C opencve checkout "${OPENCVE_VERSION}"' 'RUN python3 -m venv /app/venv')" ]
  grep -q -- "-f - https://github.com/opencve/opencve.git#$version:web" docker.log
}

@test "kind mode fails before building when the upstream Dockerfile has no clone line to patch" {
  kind_up 'RUN git clone https://example.com/other.git'
  [ "$status" -ne 0 ]
  [ ! -e docker.log ]
}

@test "lists every example except manual intervention" {
  example project project
  example projectmember projectmember
  example cvetracker cvetracker '    upjet.upbound.io/manual-intervention: needs a CVE'
  run "$ROOT/test/harness/examples.sh"
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf 'examples/namespaced/project/v1alpha1/project.yaml\nexamples/namespaced/projectmember/v1alpha1/projectmember.yaml')" ]
}

@test "package mode installs the given package instead of building and runs uptest on every example" {
  mkdir -p test/harness/.out
  cp "$ROOT/test/harness/e2e.sh" "$ROOT/test/harness/examples.sh" "$ROOT/test/harness/runtimeconfig.yaml" test/harness/
  echo '{"token":"t","organization":"o","url":"u"}' > test/harness/.out/credentials.json
  printf 'OPENCVE_URL=u\nOPENCVE_TOKEN=t\nOPENCVE_ORGANIZATION=o\nOPENCVE_MEMBERSHIP_ID=42\n' > test/harness/.out/opencve.env
  example project project
  example automation automation '    a: $b ${OPENCVE_MEMBERSHIP_ID}'
  stub kind 'exit 0'
  stub make 'echo "$*" >> make.log; if [ "$1" = uptest ]; then cat "${2#*=}" >> uptest.yaml; fi'
  stub kubectl 'echo "$*" >> kubectl.log; if [ "$1" = apply ]; then cat >> applied.yaml; fi'
  PATH="$PWD/bin:$PATH" E2E_PACKAGE=xpkg.upbound.io/cliftonz/provider-opencve:v9.9.9 run test/harness/e2e.sh
  [ "$status" -eq 0 ]
  [ "$(grep -v '^controlplane.up' make.log)" = "$(printf 'uptest UPTEST_INPUT_MANIFESTS=_output/examples/namespaced/automation/v1alpha1/automation.yaml\nuptest UPTEST_INPUT_MANIFESTS=_output/examples/namespaced/project/v1alpha1/project.yaml')" ]
  grep -qx '    a: $b 42' uptest.yaml
  grep -q 'package: xpkg.upbound.io/cliftonz/provider-opencve:v9.9.9' applied.yaml
  grep -q 'kind: DeploymentRuntimeConfig' applied.yaml
  grep -q 'wait provider.pkg provider-opencve --for condition=Healthy' kubectl.log
}

hook() {
  mkdir -p hooks
  cat > hooks/x-api.sh <<HOOK
. "$ROOT/examples/namespaced/testhooks/opencve.sh"
MR=project.project.opencve.m.clifton.quest/x
object() { get /projects/x; }
CREATED='.description == "a"'
UPDATED='.description == "b"'
assert_api "\$@"
HOOK
  stub kubectl 'echo "{\"status\":{\"atProvider\":{\"name\":\"x\"}}}"'
}

api_returns() {
  stub curl "echo \"\$*\" >> curl.log; printf '%s\n%s' '$2' $1"
}

@test "hook asserts the created object through the API with the e2e token" {
  hook
  api_returns 200 '{"description":"a"}'
  PATH="$PWD/bin:$PATH" KUBECTL=kubectl OPENCVE_URL=http://h OPENCVE_TOKEN=tok OPENCVE_ORGANIZATION=org run bash hooks/x-api.sh
  [ "$status" -eq 0 ]
  grep -q 'Authorization: Bearer tok http://h/api/v2/organizations/org/projects/x' curl.log
}

@test "hook fails when the API object does not match" {
  hook
  api_returns 200 '{"description":"a"}'
  PATH="$PWD/bin:$PATH" KUBECTL=kubectl OPENCVE_URL=http://h OPENCVE_TOKEN=tok OPENCVE_ORGANIZATION=org run bash hooks/x-api.sh updated
  [ "$status" -ne 0 ]
}

@test "hook treats a 404 as deleted" {
  hook
  api_returns 404 '{"detail":"Not found."}'
  PATH="$PWD/bin:$PATH" KUBECTL=kubectl OPENCVE_URL=http://h OPENCVE_TOKEN=tok OPENCVE_ORGANIZATION=org run bash hooks/x-api.sh deleted
  [ "$status" -eq 0 ]
}

@test "subscriptions hook updates the vendors itself after create and waits for the API to show them" {
  stub kubectl 'echo "$*" >> kubectl.log; echo "{\"status\":{\"atProvider\":{}}}"'
  stub curl 'n=$(cat n 2>/dev/null || echo 0); echo $((n + 1)) > n
if [ "$n" = 0 ]; then printf "%s\n200" "{\"subscriptions\":{\"vendors\":[\"apache\"],\"products\":{\"microsoft\":[\"windows\"]}}}"
else printf "%s\n200" "{\"subscriptions\":{\"vendors\":[\"apache\",\"linux\"],\"products\":{\"microsoft\":[\"windows\"]}}}"; fi'
  PATH="$PWD/bin:$PATH" KUBECTL=kubectl OPENCVE_URL=http://h OPENCVE_TOKEN=tok OPENCVE_ORGANIZATION=org \
    run bash "$ROOT/examples/namespaced/subscription/v1alpha1/testhooks/projectsubscriptions-api.sh"
  [ "$status" -eq 0 ]
  grep -q 'patch projectsubscriptions.subscription.opencve.m.clifton.quest/example-projectsubscriptions --type=merge -p {"spec":{"forProvider":{"vendors":\["apache","linux"\]}}}' kubectl.log
}
