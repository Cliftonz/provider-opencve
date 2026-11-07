// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package kit

import (
	"context"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/oapi-codegen/nullable"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Cliftonz/provider-opencve/internal/clients"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

type NewExternal[T resource.ModernManaged] func(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[T]

func Setup[T resource.ModernManaged](mgr ctrl.Manager, o controller.Options, gvk schema.GroupVersionKind, mg T, list resource.ManagedList, newExternal NewExternal[T]) error {
	name := managed.ControllerName(gvk.GroupKind().String())

	opts := []managed.ReconcilerOption{
		managed.WithTypedExternalConnector[T](connector(mgr.GetClient(), newExternal)),
		managed.WithInitializers(),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))), //nolint:staticcheck // crossplane-runtime has not adopted the new events API
		managed.WithMetricRecorder(o.MetricOptions.MRMetrics),
	}
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if err := mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, list, o.MetricOptions.PollStateMetricInterval)); err != nil {
		return err
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(gvk), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(mg).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

func SetupGated[T resource.ModernManaged](mgr ctrl.Manager, o controller.Options, gvk schema.GroupVersionKind, mg T, list resource.ManagedList, newExternal NewExternal[T]) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o, gvk, mg, list, newExternal); err != nil {
			mgr.GetLogger().Error(err, "unable to setup reconciler", "gvk", gvk.String())
		}
	}, gvk)
	return nil
}

func connector[T resource.ModernManaged](kube client.Client, newExternal NewExternal[T]) managed.TypedExternalConnectorFn[T] {
	return func(ctx context.Context, mg T) (managed.TypedExternalClient[T], error) {
		oc, org, err := clients.NewClient(ctx, kube, mg)
		if err != nil {
			return nil, err
		}
		return newExternal(oc, org), nil
	}
}

type response interface {
	StatusCode() int
	Status() string
	GetBody() []byte
}

func CheckStatus(action string, resp response) error {
	if resp.StatusCode() >= http.StatusOK && resp.StatusCode() < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("cannot %s: %s: %s", action, resp.Status(), resp.GetBody())
}

func NotFound(resp interface{ StatusCode() int }) bool {
	return resp.StatusCode() == http.StatusNotFound
}

const pageSize = 100

func Value[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func NullableOf[T ~string](s *string) nullable.Nullable[T] {
	if Value(s) == "" {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(T(*s))
}

func Optional[T ~string](n nullable.Nullable[T]) *string {
	v, err := n.Get()
	if err != nil || v == "" {
		return nil
	}
	s := string(v)
	return &s
}

func NullableFrom[T any, E ~string](s *string, from func(*T, E) error) (nullable.Nullable[T], error) {
	if Value(s) == "" {
		return nullable.NewNullNullable[T](), nil
	}
	var v T
	if err := from(&v, E(*s)); err != nil {
		return nil, err
	}
	return nullable.NewNullableWithValue(v), nil
}

func OptionalAs[T any, E ~string](n nullable.Nullable[T], as func(T) (E, error)) (*string, error) {
	if !n.IsSpecified() || n.IsNull() {
		return nil, nil
	}
	e, err := as(n.MustGet())
	if err != nil {
		return nil, err
	}
	if e == "" {
		return nil, nil
	}
	s := string(e)
	return &s, nil
}

func FindPaged[T any, R response](action string, fetch func(page, size int) (R, error), page func(R) ([]T, nullable.Nullable[string]), match func(T) bool) (T, bool, error) {
	var none T
	for n := 1; ; n++ {
		r, err := fetch(n, pageSize)
		if err != nil {
			return none, false, err
		}
		if NotFound(r) {
			return none, false, nil
		}
		if err := CheckStatus(action, r); err != nil {
			return none, false, err
		}
		results, next := page(r)
		for _, m := range results {
			if match(m) {
				return m, true, nil
			}
		}
		if Value(Optional(next)) == "" {
			return none, false, nil
		}
	}
}
