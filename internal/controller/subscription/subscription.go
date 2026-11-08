// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package subscription

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/Cliftonz/provider-opencve/apis/subscription/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.ProjectSubscriptions] {
	return &external{client: c, org: org}
}

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.ProjectSubscriptions) (managed.ExternalObservation, error) {
	project := meta.GetExternalName(cr)
	if project == "" {
		return managed.ExternalObservation{}, nil
	}
	resp, err := e.client.OrganizationsProjectsSubscriptionsListWithResponse(ctx, e.org, project)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get subscriptions of project %q", project), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	cr.Status.AtProvider = observation(resp.JSON200)
	if meta.WasDeleted(cr) && matches(nil, nil, cr.Status.AtProvider) {
		return managed.ExternalObservation{}, nil
	}
	cr.SetConditions(xpv2.Available())
	fp := cr.Spec.ForProvider
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: matches(fp.Vendors, fp.Products, cr.Status.AtProvider),
	}, nil
}

func observation(l *opencve.SubscriptionList) v1alpha1.ProjectSubscriptionsObservation {
	return v1alpha1.ProjectSubscriptionsObservation{
		Vendors:  kit.Value(l.Vendors),
		Products: kit.Value(l.Products),
	}
}

func matches(vendors []string, products map[string][]string, o v1alpha1.ProjectSubscriptionsObservation) bool {
	return sameSet(vendors, o.Vendors) && sameProducts(products, o.Products)
}

func sameSet(a, b []string) bool {
	return sets.New(a...).Equal(sets.New(b...))
}

func sameProducts(a, b map[string][]string) bool {
	for vendor, products := range a {
		if !sameSet(products, b[vendor]) {
			return false
		}
	}
	for vendor, products := range b {
		if !sameSet(products, a[vendor]) {
			return false
		}
	}
	return true
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.ProjectSubscriptions) (managed.ExternalCreation, error) {
	project := cr.Spec.ForProvider.ProjectName
	resp, err := e.put(ctx, project, cr.Spec.ForProvider.Vendors, cr.Spec.ForProvider.Products)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("set subscriptions of project %q", project), resp); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, project)
	cr.Status.AtProvider = observation(resp.JSON200)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.ProjectSubscriptions) (managed.ExternalUpdate, error) {
	project := meta.GetExternalName(cr)
	resp, err := e.put(ctx, project, cr.Spec.ForProvider.Vendors, cr.Spec.ForProvider.Products)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("set subscriptions of project %q", project), resp); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(resp.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.ProjectSubscriptions) (managed.ExternalDelete, error) {
	project := meta.GetExternalName(cr)
	resp, err := e.put(ctx, project, nil, nil)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("clear subscriptions of project %q", project), resp)
}

func (e *external) put(ctx context.Context, project string, vendors []string, products map[string][]string) (*opencve.OrganizationsProjectsSubscriptionsUpdateResponse, error) {
	if vendors == nil {
		vendors = []string{}
	}
	if products == nil {
		products = map[string][]string{}
	}
	return e.client.OrganizationsProjectsSubscriptionsUpdateWithResponse(ctx, e.org, project, opencve.SubscriptionListRequest{Vendors: &vendors, Products: &products})
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
