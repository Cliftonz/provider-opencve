// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/apis/notification/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	return kit.Setup(mgr, o, v1alpha1.NotificationGroupVersionKind, &v1alpha1.Notification{}, &v1alpha1.NotificationList{}, newExternal(mgr))
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return kit.SetupGated(mgr, o, v1alpha1.NotificationGroupVersionKind, &v1alpha1.Notification{}, &v1alpha1.NotificationList{}, newExternal(mgr))
}

func newExternal(mgr ctrl.Manager) kit.NewExternal[*v1alpha1.Notification] {
	return func(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.Notification] {
		return &external{kube: mgr.GetClient(), client: c, org: org}
	}
}
