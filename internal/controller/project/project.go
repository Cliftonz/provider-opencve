// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"context"
	"fmt"
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}
	resp, err := e.client.OrganizationsProjectsRetrieveWithResponse(ctx, e.org, name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get project %q", name), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	remote := resp.JSON200
	lateInitialized := cr.Spec.ForProvider.Active == nil && remote.Active != nil
	if lateInitialized {
		cr.Spec.ForProvider.Active = remote.Active
	}
	cr.Status.AtProvider = observation(remote)
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        kit.Value(cr.Spec.ForProvider.Description) == kit.Value(cr.Status.AtProvider.Description) && reflect.DeepEqual(cr.Spec.ForProvider.Active, cr.Status.AtProvider.Active),
		ResourceLateInitialized: lateInitialized,
	}, nil
}

func observation(remote *opencve.ProjectDetail) v1alpha1.ProjectObservation {
	o := v1alpha1.ProjectObservation{
		Name:               remote.Name,
		Description:        kit.Optional(remote.Description),
		Active:             remote.Active,
		SubscriptionsCount: remote.SubscriptionsCount,
	}
	if remote.Id != nil {
		o.ID = remote.Id.String()
	}
	return o
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalCreation, error) {
	fp := cr.Spec.ForProvider
	resp, err := e.client.OrganizationsProjectsCreateWithResponse(ctx, e.org, opencve.ProjectWriteRequest{
		Name:        fp.Name,
		Description: kit.NullableOf[string](fp.Description),
		Active:      fp.Active,
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("create project %q", fp.Name), resp); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, fp.Name)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalUpdate, error) {
	name := meta.GetExternalName(cr)
	resp, err := e.client.OrganizationsProjectsPartialUpdateWithResponse(ctx, e.org, name, opencve.PatchedProjectWriteRequest{
		Description: kit.NullableOf[string](cr.Spec.ForProvider.Description),
		Active:      cr.Spec.ForProvider.Active,
	})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update project %q", name), resp); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(resp.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalDelete, error) {
	name := meta.GetExternalName(cr)
	resp, err := e.client.OrganizationsProjectsDestroyWithResponse(ctx, e.org, name)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("delete project %q", name), resp)
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
