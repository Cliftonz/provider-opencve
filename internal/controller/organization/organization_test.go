// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Cliftonz/provider-opencve/apis/organization/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

type recorded struct {
	Method string
	Path   string
}

func serve(t *testing.T, status int, body string) (*external, *[]recorded) {
	t.Helper()
	var requests []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, recorded{Method: r.Method, Path: r.URL.Path})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, err := opencve.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &external{client: c, organization: "acme"}, &requests
}

func TestObserveNotFound(t *testing.T) {
	e, requests := serve(t, http.StatusNotFound, `{"detail":"Not found."}`)

	got, err := e.Observe(context.Background(), &v1alpha1.Organization{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, got); diff != "" {
		t.Errorf("observation (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]recorded{{Method: http.MethodGet, Path: "/organizations/acme"}}, *requests); diff != "" {
		t.Errorf("requests (-want +got):\n%s", diff)
	}
}

func TestObserveExists(t *testing.T) {
	e, requests := serve(t, http.StatusOK, `{
		"id": "6f1c1a2e-1b2c-4d3e-8f90-123456789abc",
		"name": "acme",
		"members_count": 3,
		"projects_count": 5,
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-02-03T04:05:06Z"
	}`)
	cr := &v1alpha1.Organization{}

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, got); diff != "" {
		t.Errorf("observation (-want +got):\n%s", diff)
	}
	members, projects := 3, 5
	want := v1alpha1.OrganizationObservation{
		ID:            "6f1c1a2e-1b2c-4d3e-8f90-123456789abc",
		Name:          "acme",
		MembersCount:  &members,
		ProjectsCount: &projects,
		CreatedAt:     &metav1.Time{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		UpdatedAt:     &metav1.Time{Time: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)},
	}
	if diff := cmp.Diff(want, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(xpv2.Available(), cr.GetCondition(xpv2.TypeReady), cmpopts.IgnoreFields(xpv2.Condition{}, "LastTransitionTime")); diff != "" {
		t.Errorf("ready condition (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]recorded{{Method: http.MethodGet, Path: "/organizations/acme"}}, *requests); diff != "" {
		t.Errorf("requests (-want +got):\n%s", diff)
	}
}

func TestObserveExistsWithoutIDOrTimestamps(t *testing.T) {
	e, _ := serve(t, http.StatusOK, `{"name": "acme"}`)
	cr := &v1alpha1.Organization{}

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, got); diff != "" {
		t.Errorf("observation (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(v1alpha1.OrganizationObservation{Name: "acme"}, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider (-want +got):\n%s", diff)
	}
}

func TestObserveServerError(t *testing.T) {
	e, requests := serve(t, http.StatusInternalServerError, `{"detail":"boom"}`)

	_, err := e.Observe(context.Background(), &v1alpha1.Organization{})
	if err == nil || !strings.HasPrefix(err.Error(), `cannot get organization "acme": `) {
		t.Fatalf("expected a quoted-identifier error for HTTP 500, got %v", err)
	}
	if diff := cmp.Diff([]recorded{{Method: http.MethodGet, Path: "/organizations/acme"}}, *requests); diff != "" {
		t.Errorf("requests (-want +got):\n%s", diff)
	}
}

func TestObserveDeletedReportsAbsent(t *testing.T) {
	e, requests := serve(t, http.StatusNotFound, `{"detail":"Not found."}`)
	cr := &v1alpha1.Organization{}
	cr.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, got); diff != "" {
		t.Errorf("observation (-want +got):\n%s", diff)
	}
	if len(*requests) != 0 {
		t.Errorf("Observe sent requests: %v", *requests)
	}
}

func TestCreateCannotCreate(t *testing.T) {
	e, requests := serve(t, http.StatusOK, `{}`)

	_, err := e.Create(context.Background(), &v1alpha1.Organization{})
	if !errors.Is(err, errCannotCreate) {
		t.Errorf("Create error: want %v, got %v", errCannotCreate, err)
	}
	if len(*requests) != 0 {
		t.Errorf("Create sent requests: %v", *requests)
	}
}

func TestUpdateAndDeleteLeaveServerUntouched(t *testing.T) {
	e, requests := serve(t, http.StatusOK, `{}`)
	cr := &v1alpha1.Organization{}

	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Delete(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	if len(*requests) != 0 {
		t.Errorf("Update or Delete sent requests: %v", *requests)
	}
}
