// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"errors"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Cliftonz/provider-opencve/apis/organization/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

var errCannotCreate = errors.New("the OpenCVE API cannot create organizations; an Organization only observes the organization named in its ProviderConfig credentials")

type external struct {
	client       *opencve.ClientWithResponses
	organization string
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.Organization) (managed.ExternalObservation, error) {
	if meta.WasDeleted(cr) {
		return managed.ExternalObservation{}, nil
	}
	resp, err := e.client.OrganizationsRetrieveWithResponse(ctx, e.organization)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get organization %q", e.organization), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	o := resp.JSON200
	cr.Status.AtProvider = v1alpha1.OrganizationObservation{
		Name:          o.Name,
		MembersCount:  o.MembersCount,
		ProjectsCount: o.ProjectsCount,
	}
	if o.Id != nil {
		cr.Status.AtProvider.ID = o.Id.String()
	}
	if o.CreatedAt != nil {
		cr.Status.AtProvider.CreatedAt = &metav1.Time{Time: *o.CreatedAt}
	}
	if o.UpdatedAt != nil {
		cr.Status.AtProvider.UpdatedAt = &metav1.Time{Time: *o.UpdatedAt}
	}
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *external) Create(_ context.Context, _ *v1alpha1.Organization) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, errCannotCreate
}

func (e *external) Update(_ context.Context, _ *v1alpha1.Organization) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(_ context.Context, _ *v1alpha1.Organization) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
