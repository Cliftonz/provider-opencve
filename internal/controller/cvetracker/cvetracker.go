// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package cvetracker

import (
	"context"
	"errors"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Cliftonz/provider-opencve/apis/cvetracker/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

var errCannotCreate = errors.New("the OpenCVE API cannot create CVE trackers; the CVE must already be in the project")

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.CveTracker] {
	return &external{client: c, org: org}
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.CveTracker) (managed.ExternalObservation, error) {
	if meta.WasDeleted(cr) {
		return managed.ExternalObservation{}, nil
	}
	fp := cr.Spec.ForProvider
	resp, err := e.client.OrganizationsProjectsCvesRetrieveWithResponse(ctx, e.org, fp.ProjectName, fp.CveID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get CVE tracker %q", fp.CveID), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	at := observation(resp.JSON200)
	cr.Status.AtProvider = at
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: kit.Value(fp.Status) == at.Status && kit.Value(fp.Assignee) == at.Assignee,
	}, nil
}

func observation(remote *opencve.ApiProjectCve) v1alpha1.CveTrackerObservation {
	var tracker map[string]any
	if remote.Tracker.IsSpecified() && !remote.Tracker.IsNull() {
		tracker = remote.Tracker.MustGet()
	}
	return v1alpha1.CveTrackerObservation{
		CveID:    remote.CveId,
		Title:    kit.Value(kit.Optional(remote.Title)),
		Status:   field[string](tracker, "status"),
		Assignee: field[string](field[map[string]any](tracker, "assignee"), "email"),
	}
}

func field[T any](m map[string]any, key string) T {
	var zero T
	if v := m[key]; v != nil {
		return v.(T)
	}
	return zero
}

func (e *external) Create(_ context.Context, _ *v1alpha1.CveTracker) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, errCannotCreate
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.CveTracker) (managed.ExternalUpdate, error) {
	fp := cr.Spec.ForProvider
	status, err := kit.NullableFrom(fp.Status, (*opencve.PatchedCveTrackerUpdateRequest_Status).FromCveTrackerUpdateStatusEnum)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	resp, err := e.client.OrganizationsProjectsCvesPartialUpdateWithResponse(ctx, e.org, fp.ProjectName, fp.CveID, opencve.PatchedCveTrackerUpdateRequest{
		Status:   status,
		Assignee: kit.NullableOf[openapi_types.Email](fp.Assignee),
	})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update CVE tracker %q", fp.CveID), resp); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(resp.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(_ context.Context, _ *v1alpha1.CveTracker) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
