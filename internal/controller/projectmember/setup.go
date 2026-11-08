// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package projectmember

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	return kit.Setup(mgr, o, v1alpha1.ProjectMemberGroupVersionKind, &v1alpha1.ProjectMember{}, &v1alpha1.ProjectMemberList{}, newExternal)
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return kit.SetupGated(mgr, o, v1alpha1.ProjectMemberGroupVersionKind, &v1alpha1.ProjectMember{}, &v1alpha1.ProjectMemberList{}, newExternal)
}
