<!--
SPDX-FileCopyrightText: 2026 The provider-opencve authors

SPDX-License-Identifier: Apache-2.0
-->

# Decisions

- OpenCVE is pinned to tag `v3.1.0`. The webserver image is built from `web/Dockerfile` at that tag; its `FROM python:3.12-slim-trixie` is replaced through a buildx named context with the digest in `images.env`.
- Only the webserver and Postgres run. Airflow, nginx, Redis and the KB import are omitted; the v3.1.0 webserver does not use Redis.
- Subscriptions are validated against the vendor and product tables and a CveTracker needs a CVE matching the project subscriptions, so `bootstrap.py` seeds the vendors, product and CVE the examples use instead of importing the KB.
- The webserver uses `DJANGO_SETTINGS_MODULE=opencve.conf.base` and `OPENCVE_*` environment variables, so no `settings.py` or `.env` file is mounted.
- Migrations run in an init container. Static files are not collected; the API does not serve them.
- The superuser comes from `createsuperuser --noinput` with `DJANGO_SUPERUSER_*`, run only when no superuser exists. `bootstrap.py` runs in `manage.py shell`: it creates the `opencve-e2e` organization with the superuser as owner and replaces its write-access token through `OrganizationAPIToken.create_token`, an internal API.
- The webserver is a NodePort on 30900, mapped to the host by `kind.yaml`. The provider uses `http://<cluster>-control-plane:30900`, which pods in any kind cluster on the Docker `kind` network resolve.
- Postgres uses the Alpine image; the `cves` migrations create the `pg_trgm` and `uuid-ossp` extensions themselves, so the compose init script is not needed.
- Fixed, non-secret `OPENCVE_SECRET_KEY` and superuser password in the manifest; the instance is local and throwaway.
- Storage is `emptyDir`; the harness is disposable.
- `e2e.sh` runs uptest once per example file; dependencies in a file carry the root's example-id.
- ProjectMember runs only in SaaS mode; the project members endpoint is not in v3.1.0.
- `E2E_PACKAGE` makes `e2e.sh` install a published package with `runtimeconfig.yaml` instead of building and deploying a local xpkg.
