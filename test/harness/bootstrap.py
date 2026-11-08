# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

from django.utils import timezone

from cves.models import Cve, Product, Vendor
from organizations.models import Membership, Organization, OrganizationAPIToken
from users.models import User

admin = User.objects.get(is_superuser=True)
org, _ = Organization.objects.get_or_create(name="opencve-e2e")
Membership.objects.get_or_create(
    user=admin, organization=org, defaults={"role": Membership.OWNER, "date_joined": timezone.now()}
)
OrganizationAPIToken.objects.filter(organization=org, name="opencve-e2e").delete()
token = OrganizationAPIToken.create_token(org, "opencve-e2e", None, admin, OrganizationAPIToken.AccessMode.WRITE)

for name in ("apache", "linux"):
    Vendor.objects.get_or_create(name=name)
Product.objects.get_or_create(name="windows", vendor=Vendor.objects.get_or_create(name="microsoft")[0])
Cve.objects.get_or_create(
    cve_id="CVE-2021-44228", defaults={"title": "Apache Log4j2 JNDI features", "vendors": ["apache"]}
)

print(f"OPENCVE_TOKEN={token}")
print(f"OPENCVE_ORGANIZATION={org.name}")
