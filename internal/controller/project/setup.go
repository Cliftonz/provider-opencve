// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	return kit.Setup(mgr, o, v1alpha1.ProjectGroupVersionKind, &v1alpha1.Project{}, &v1alpha1.ProjectList{}, newExternal)
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return kit.SetupGated(mgr, o, v1alpha1.ProjectGroupVersionKind, &v1alpha1.Project{}, &v1alpha1.ProjectList{}, newExternal)
}

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.Project] {
	return &external{client: c, org: org}
}
