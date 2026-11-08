<!--
SPDX-FileCopyrightText: 2026 The provider-opencve authors

SPDX-License-Identifier: Apache-2.0
-->

# OpenCVE test harness

A kind cluster named `opencve-e2e` that end to end tests run against. By default it runs a self-hosted OpenCVE webserver and Postgres, bootstrapped headlessly so the tests get a real organization token. Crossplane and the provider are installed into the same cluster by `e2e.sh`.

The webserver image is built from the OpenCVE source at `OPENCVE_VERSION`, a master commit because the project members endpoint is not in any release yet; it moves back to a release tag once one ships with it. The image is built with the Python base image pinned by digest; every other image is pinned by digest in `images.env`.

## Run

Requires Docker with buildx, kind, kubectl, curl, jq and envsubst; run under bash (WSL Ubuntu or Linux).

```sh
test/harness/up.sh     # create cluster if missing, deploy OpenCVE, bootstrap
test/harness/e2e.sh    # install Crossplane and the locally built provider, run uptest on every example
E2E_PACKAGE=xpkg.upbound.io/cliftonz/provider-opencve:v0.1.0 test/harness/e2e.sh   # same, with a published package
test/harness/down.sh   # delete the cluster
```

`up.sh` writes `test/harness/.out/credentials.json`, the ProviderConfig secret:

| Key | Meaning |
|---|---|
| `token` | write-access organization API token |
| `organization` | `opencve-e2e` |
| `url` | `http://opencve-e2e-control-plane:30900`, reachable from pods in kind; the provider appends `/api/v2` |

It also writes `test/harness/.out/opencve.env` with `OPENCVE_URL` (`http://localhost:30900`, the host side of the same API), `OPENCVE_TOKEN`, `OPENCVE_ORGANIZATION` and `OPENCVE_MEMBERSHIP_ID`; `e2e.sh` exports them for the API test hooks and substitutes `${OPENCVE_MEMBERSHIP_ID}` in the examples.

`bootstrap.py` seeds the vendors apache, linux and microsoft, the product microsoft windows and the CVE CVE-2021-44228 (vendor apache), which the subscription and CveTracker examples need. It also seeds the user `e2e-member` with a joined `member` membership in `opencve-e2e`, whose id is `OPENCVE_MEMBERSHIP_ID`, for the ProjectMember example.

## SaaS mode

```sh
E2E_TARGET=saas OPENCVE_TOKEN=opc_org.... OPENCVE_ORGANIZATION=my-org OPENCVE_MEMBERSHIP_ID=123 test/harness/up.sh
E2E_TARGET=saas test/harness/e2e.sh
```

`OPENCVE_MEMBERSHIP_ID` is required: the id of a joined member of the organization who is not the token's author, since a pending invite cannot join a project and the token's author cannot change its own project role. `OPENCVE_URL` defaults to `https://app.opencve.io`. `up.sh` then only creates the kind cluster and writes the given credentials; no OpenCVE is deployed.

## Examples per target

| Example | kind | saas |
|---|---|---|
| projectmember | run, against the seeded `e2e-member` | run, against `OPENCVE_MEMBERSHIP_ID` |
| every other kind | run | run |

## API test hooks

Each example root runs `<group>/v1alpha1/testhooks/<kind>-api.sh` against the OpenCVE API with the e2e token: as post-assert hook it checks the created object, as pre-delete hook with `updated` it checks the update-parameter value, and as post-delete hook with `deleted` it waits for the MR to go and the object to be gone. `examples/namespaced/testhooks/opencve.sh` holds the shared phases. Organization deletion leaves the organization, so its delete hook asserts it still exists; CveTracker deletion leaves the tracker, so it has no delete hook.
