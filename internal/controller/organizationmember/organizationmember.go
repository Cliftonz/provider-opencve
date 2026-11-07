// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package organizationmember

import (
	"context"
	"fmt"
	"strconv"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.OrganizationMember] {
	return &external{client: c, org: org}
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.OrganizationMember) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}
	id, err := strconv.Atoi(name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	m, found, err := kit.FindPaged(fmt.Sprintf("list organization members of %q", e.org),
		func(page, size int) (*opencve.OrganizationsMembersListResponse, error) {
			return e.client.OrganizationsMembersListWithResponse(ctx, e.org, &opencve.OrganizationsMembersListParams{Page: &page, PageSize: &size})
		},
		func(r *opencve.OrganizationsMembersListResponse) ([]opencve.Membership, nullable.Nullable[string]) {
			return r.JSON200.Results, r.JSON200.Next
		},
		func(m opencve.Membership) bool { return *m.Id == id },
	)
	if err != nil || !found {
		return managed.ExternalObservation{}, err
	}
	cr.Status.AtProvider = observation(m)
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: *m.Role == cr.Spec.ForProvider.Role}, nil
}

func observation(m opencve.Membership) v1alpha1.OrganizationMemberObservation {
	o := v1alpha1.OrganizationMemberObservation{
		ID:          *m.Id,
		Email:       string(*m.Email),
		Role:        *m.Role,
		DateInvited: &metav1.Time{Time: *m.DateInvited},
		IsInvited:   *m.IsInvited,
	}
	if m.DateJoined.IsSpecified() && !m.DateJoined.IsNull() {
		o.DateJoined = &metav1.Time{Time: m.DateJoined.MustGet()}
	}
	return o
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.OrganizationMember) (managed.ExternalCreation, error) {
	r, err := e.client.OrganizationsMembersCreateWithResponse(ctx, e.org, opencve.MembershipCreateRequest{
		Email: openapi_types.Email(cr.Spec.ForProvider.Email),
		Role:  opencve.Role5a7Enum(cr.Spec.ForProvider.Role),
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("create organization member %q", cr.Spec.ForProvider.Email), r); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, strconv.Itoa(*r.JSON201.Id))
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.OrganizationMember) (managed.ExternalUpdate, error) {
	id, err := strconv.Atoi(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	role := opencve.Role5a7Enum(cr.Spec.ForProvider.Role)
	r, err := e.client.OrganizationsMembersPartialUpdateWithResponse(ctx, e.org, id, opencve.PatchedMembershipUpdateRequest{Role: &role})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update organization member %q", meta.GetExternalName(cr)), r); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(*r.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.OrganizationMember) (managed.ExternalDelete, error) {
	id, err := strconv.Atoi(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	r, err := e.client.OrganizationsMembersDestroyWithResponse(ctx, e.org, id)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(r) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("delete organization member %q", meta.GetExternalName(cr)), r)
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
