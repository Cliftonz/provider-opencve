// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package organizationmember

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	return kit.Setup(mgr, o, v1alpha1.OrganizationMemberGroupVersionKind, &v1alpha1.OrganizationMember{}, &v1alpha1.OrganizationMemberList{}, newExternal)
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return kit.SetupGated(mgr, o, v1alpha1.OrganizationMemberGroupVersionKind, &v1alpha1.OrganizationMember{}, &v1alpha1.OrganizationMemberList{}, newExternal)
}
