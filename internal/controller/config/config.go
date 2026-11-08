// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	if err := setupNamespacedProviderConfig(mgr, o); err != nil {
		return err
	}
	return setupClusterProviderConfig(mgr, o)
}

func setupNamespacedProviderConfig(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName(v1beta1.ProviderConfigGroupKind)

	of := resource.ProviderConfigKinds{
		Config:    v1beta1.ProviderConfigGroupVersionKind,
		Usage:     v1beta1.ProviderConfigUsageGroupVersionKind,
		UsageList: v1beta1.ProviderConfigUsageListGroupVersionKind,
	}

	r := providerconfig.NewReconciler(mgr, of,
		providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))) //nolint:staticcheck // crossplane-runtime has not adopted the new events API

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.ProviderConfig{}).
		Watches(&v1beta1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{Kind: v1beta1.ProviderConfigKind}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

func setupClusterProviderConfig(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName(v1beta1.ClusterProviderConfigGroupKind)

	of := resource.ProviderConfigKinds{
		Config:    v1beta1.ClusterProviderConfigGroupVersionKind,
		Usage:     v1beta1.ProviderConfigUsageGroupVersionKind,
		UsageList: v1beta1.ProviderConfigUsageListGroupVersionKind,
	}

	r := providerconfig.NewReconciler(mgr, of,
		providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))) //nolint:staticcheck // crossplane-runtime has not adopted the new events API

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.ClusterProviderConfig{}).
		Watches(&v1beta1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{Kind: v1beta1.ClusterProviderConfigKind}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			mgr.GetLogger().Error(err, "unable to setup reconcilers", "clusterProviderConfigGVK", v1beta1.ClusterProviderConfigGroupVersionKind.String(), "providerConfigGVK", v1beta1.ProviderConfigGroupVersionKind.String(), "providerConfigUsageGVK", v1beta1.ProviderConfigUsageGroupVersionKind.String())
		}
	}, v1beta1.ClusterProviderConfigGroupVersionKind, v1beta1.ProviderConfigGroupVersionKind, v1beta1.ProviderConfigUsageGroupVersionKind)
	return nil
}
