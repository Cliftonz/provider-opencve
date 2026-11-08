<!--
SPDX-FileCopyrightText: 2026 The provider-opencve authors

SPDX-License-Identifier: Apache-2.0
-->

# Decisions

- OpenCVE is pinned to master commit `c1237101ea3979b63e498368bdc3cefc50d305e5`, because the project members endpoint is not in any release yet; the pin moves back to a release tag once one ships with it. The webserver image is built from `web/Dockerfile` at that commit; its `FROM python:3.12-slim-trixie` is replaced through a buildx named context with the digest in `images.env`.
- The upstream `web/Dockerfile` clones with `git clone -b`, which rejects commit SHAs, so `up.sh` fetches it at the pinned commit and patches that line at build time into a full clone plus `git checkout`; the build fails if the line is not found. No copy of the Dockerfile is committed, since OpenCVE's license differs from this repo's and a copy would drift.
- Only the webserver and Postgres run. Airflow, nginx, Redis and the KB import are omitted; the pinned webserver does not use Redis.
- Subscriptions are validated against the vendor and product tables and a CveTracker needs a CVE matching the project subscriptions, so `bootstrap.py` seeds the vendors, product and CVE the examples use instead of importing the KB.
- The webserver uses `DJANGO_SETTINGS_MODULE=opencve.conf.base` and `OPENCVE_*` environment variables, so no `settings.py` or `.env` file is mounted.
- Migrations run in an init container. Static files are not collected; the API does not serve them.
- The superuser comes from `createsuperuser --noinput` with `DJANGO_SUPERUSER_*`, run only when no superuser exists. `bootstrap.py` runs in `manage.py shell`: it creates the `opencve-e2e` organization with the superuser as owner and replaces its write-access token through `OrganizationAPIToken.create_token`, an internal API.
- The webserver is a NodePort on 30900, mapped to the host by `kind.yaml`. The provider uses `http://<cluster>-control-plane:30900`, which pods in any kind cluster on the Docker `kind` network resolve.
- Postgres uses the Alpine image; the `cves` migrations create the `pg_trgm` and `uuid-ossp` extensions themselves, so the compose init script is not needed.
- Fixed, non-secret `OPENCVE_SECRET_KEY` and superuser password in the manifest; the instance is local and throwaway.
- Storage is `emptyDir`; the harness is disposable.
- `e2e.sh` runs uptest once per example file; dependencies in a file carry the root's example-id.
- A project member needs a joined organization member other than the token's author, so `bootstrap.py` seeds the user `e2e-member` with a joined membership and prints its id as `OPENCVE_MEMBERSHIP_ID`. `e2e.sh` substitutes it into the examples under `_output/examples`, where uptest reads them; in SaaS mode the user provides it.
- `E2E_PACKAGE` makes `e2e.sh` install a published package with `runtimeconfig.yaml` instead of building and deploying a local xpkg.
