// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package projectmember

import (
	"context"
	"fmt"
	"strconv"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/oapi-codegen/nullable"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.ProjectMember] {
	return &external{client: c, org: org}
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.ProjectMember) (managed.ExternalObservation, error) {
	ext := meta.GetExternalName(cr)
	if ext == "" {
		return managed.ExternalObservation{}, nil
	}
	id, err := strconv.Atoi(ext)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	m, found, err := kit.FindPaged(fmt.Sprintf("find project member %q", ext),
		func(page, size int) (*opencve.OrganizationsProjectsMembersListResponse, error) {
			return e.client.OrganizationsProjectsMembersListWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, &opencve.OrganizationsProjectsMembersListParams{Page: &page, PageSize: &size})
		},
		func(r *opencve.OrganizationsProjectsMembersListResponse) ([]opencve.ProjectMembership, nullable.Nullable[string]) {
			return r.JSON200.Results, r.JSON200.Next
		},
		func(m opencve.ProjectMembership) bool { return *m.Id == id },
	)
	if err != nil || !found {
		return managed.ExternalObservation{}, err
	}
	cr.Status.AtProvider = observation(m)
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: m.Role == cr.Spec.ForProvider.Role}, nil
}

func observation(m opencve.ProjectMembership) v1alpha1.ProjectMemberObservation {
	return v1alpha1.ProjectMemberObservation{
		ID:           *m.Id,
		MembershipID: *m.MembershipId,
		Role:         m.Role,
		Email:        string(*m.Email),
		Username:     kit.Value(kit.Optional(m.Username)),
	}
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.ProjectMember) (managed.ExternalCreation, error) {
	membershipID, err := strconv.Atoi(cr.Spec.ForProvider.MembershipID)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	r, err := e.client.OrganizationsProjectsMembersCreateWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, opencve.ProjectMembershipCreateRequest{
		MembershipId: membershipID,
		Role:         opencve.Role34eEnum(cr.Spec.ForProvider.Role),
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("create project member for membership %q", cr.Spec.ForProvider.MembershipID), r); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, strconv.Itoa(*r.JSON201.Id))
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.ProjectMember) (managed.ExternalUpdate, error) {
	ext := meta.GetExternalName(cr)
	id, err := strconv.Atoi(ext)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	role := opencve.Role34eEnum(cr.Spec.ForProvider.Role)
	r, err := e.client.OrganizationsProjectsMembersPartialUpdateWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, id, opencve.PatchedProjectMembershipUpdateRequest{Role: &role})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update project member %q", ext), r); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(*r.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.ProjectMember) (managed.ExternalDelete, error) {
	ext := meta.GetExternalName(cr)
	id, err := strconv.Atoi(ext)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	r, err := e.client.OrganizationsProjectsMembersDestroyWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, id)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(r) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("delete project member %q", ext), r)
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
