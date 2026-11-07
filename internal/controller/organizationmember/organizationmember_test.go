// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package organizationmember

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	membersPath = "/organizations/acme/members"
	listPage1   = "GET " + membersPath + "?page=1&page_size=100"
	listPage2   = "GET " + membersPath + "?page=2&page_size=100"
)

type request struct {
	Method string
	Path   string
	Query  string
	Body   string
}

type reply struct {
	Status int
	Body   string
}

func server(t *testing.T, replies map[string]reply) (*external, *[]request) {
	t.Helper()
	var got []request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)})
		rep, ok := replies[r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery]
		if !ok {
			rep = reply{Status: http.StatusNotFound, Body: `{}`}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rep.Status)
		_, _ = w.Write([]byte(rep.Body))
	}))
	t.Cleanup(srv.Close)
	c, err := opencve.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &external{client: c, org: "acme"}, &got
}

func member(externalName, email, role string) *v1alpha1.OrganizationMember {
	cr := &v1alpha1.OrganizationMember{}
	cr.Spec.ForProvider = v1alpha1.OrganizationMemberParameters{Email: email, Role: role}
	if externalName != "" {
		cr.SetAnnotations(map[string]string{"crossplane.io/external-name": externalName})
	}
	return cr
}

func TestObserveWithoutExternalName(t *testing.T) {
	e, got := server(t, nil)
	obs, err := e.Observe(context.Background(), member("", "a@b.c", "member"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestObserveFoundOnSecondPage(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":2,"next":"http://x/organizations/acme/members?page=2","previous":null,"results":[
			{"id":3,"email":"other@b.c","role":"owner","date_invited":"2026-01-01T00:00:00Z","date_joined":"2026-01-01T00:00:00Z","is_invited":false}]}`},
		listPage2: {Status: http.StatusOK, Body: `{"count":2,"next":null,"previous":"http://x/organizations/acme/members?page=1","results":[
			{"id":7,"email":"a@b.c","role":"member","date_invited":"2026-01-02T03:04:05Z","date_joined":null,"is_invited":true}]}`},
	})
	cr := member("7", "a@b.c", "member")
	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	wantReqs := []request{
		{Method: http.MethodGet, Path: membersPath, Query: "page=1&page_size=100"},
		{Method: http.MethodGet, Path: membersPath, Query: "page=2&page_size=100"},
	}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	wantAt := v1alpha1.OrganizationMemberObservation{
		ID:          7,
		Email:       "a@b.c",
		Role:        "member",
		DateInvited: &metav1.Time{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		IsInvited:   true,
	}
	if diff := cmp.Diff(wantAt, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
	if !cr.GetCondition(xpv2.TypeReady).Equal(xpv2.Available()) {
		t.Errorf("ready condition = %v, want Available", cr.GetCondition(xpv2.TypeReady))
	}
}

func TestObserveJoinedMember(t *testing.T) {
	e, _ := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":1,"next":null,"previous":null,"results":[
			{"id":7,"email":"a@b.c","role":"admin","date_invited":"2026-01-02T03:04:05Z","date_joined":"2026-01-03T00:00:00Z","is_invited":false}]}`},
	})
	cr := member("7", "a@b.c", "admin")
	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	want := &metav1.Time{Time: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}
	if diff := cmp.Diff(want, cr.Status.AtProvider.DateJoined); diff != "" {
		t.Errorf("dateJoined -want +got:\n%s", diff)
	}
}

func TestObserveNotFoundAfterAllPages(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":2,"next":"http://x/organizations/acme/members?page=2","previous":null,"results":[
			{"id":3,"email":"other@b.c","role":"owner","date_invited":"2026-01-01T00:00:00Z","date_joined":null,"is_invited":true}]}`},
		listPage2: {Status: http.StatusOK, Body: `{"count":2,"next":null,"previous":"http://x/organizations/acme/members?page=1","results":[
			{"id":7,"email":"a@b.c","role":"member","date_invited":"2026-01-02T03:04:05Z","date_joined":null,"is_invited":true}]}`},
	})
	obs, err := e.Observe(context.Background(), member("9", "z@b.c", "member"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	wantReqs := []request{
		{Method: http.MethodGet, Path: membersPath, Query: "page=1&page_size=100"},
		{Method: http.MethodGet, Path: membersPath, Query: "page=2&page_size=100"},
	}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestObserveNonNumericExternalName(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":0,"next":null,"previous":null,"results":[]}`},
	})
	if _, err := e.Observe(context.Background(), member("alice", "a@b.c", "member")); err == nil {
		t.Fatal("Observe() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestObserveNotFoundWhenNextMissing(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":1,"results":[
			{"id":3,"email":"other@b.c","role":"owner","date_invited":"2026-01-01T00:00:00Z","date_joined":null,"is_invited":true}]}`},
	})
	obs, err := e.Observe(context.Background(), member("9", "z@b.c", "member"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	wantReqs := []request{{Method: http.MethodGet, Path: membersPath, Query: "page=1&page_size=100"}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestObserveListNotFound(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusNotFound, Body: `{"detail":"Not found."}`},
	})
	obs, err := e.Observe(context.Background(), member("7", "a@b.c", "member"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	wantReqs := []request{{Method: http.MethodGet, Path: membersPath, Query: "page=1&page_size=100"}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestObserveListError(t *testing.T) {
	e, _ := server(t, map[string]reply{
		listPage1: {Status: http.StatusInternalServerError, Body: `{"detail":"boom"}`},
	})
	if _, err := e.Observe(context.Background(), member("7", "a@b.c", "member")); err == nil || !strings.Contains(err.Error(), `cannot list organization members of "acme"`) {
		t.Fatalf("Observe() error = %v, want cannot list organization members of \"acme\"", err)
	}
}

func TestObserveRoleDiffers(t *testing.T) {
	e, _ := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":1,"next":null,"previous":null,"results":[
			{"id":7,"email":"a@b.c","role":"member","date_invited":"2026-01-02T03:04:05Z","date_joined":null,"is_invited":true}]}`},
	})
	obs, err := e.Observe(context.Background(), member("7", "a@b.c", "admin"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
}

func TestCreate(t *testing.T) {
	e, got := server(t, map[string]reply{
		"POST /organizations/acme/members?": {Status: http.StatusCreated, Body: `{"id":7,"email":"a@b.c","role":"member","date_invited":"2026-01-02T03:04:05Z","date_joined":null,"is_invited":true}`},
	})
	cr := member("", "a@b.c", "member")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	wantReqs := []request{{Method: http.MethodPost, Path: membersPath, Body: `{"email":"a@b.c","role":"member"}`}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	if name := meta.GetExternalName(cr); name != "7" {
		t.Errorf("external-name = %q, want %q", name, "7")
	}
}

func TestCreateRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"POST /organizations/acme/members?": {Status: http.StatusBadRequest, Body: `{"role":["\"boss\" is not a valid choice."]}`},
	})
	cr := member("", "a@b.c", "boss")
	if _, err := e.Create(context.Background(), cr); err == nil || !strings.Contains(err.Error(), `cannot create organization member "a@b.c"`) {
		t.Fatalf("Create() error = %v, want cannot create organization member \"a@b.c\"", err)
	}
	if name := meta.GetExternalName(cr); name != "" {
		t.Errorf("external-name = %q, want empty", name)
	}
}

func TestUpdate(t *testing.T) {
	e, got := server(t, map[string]reply{
		"PATCH /organizations/acme/members/7?": {Status: http.StatusOK, Body: `{"id":7,"email":"a@b.c","role":"admin","date_invited":"2026-01-02T03:04:05Z","date_joined":null,"is_invited":true}`},
	})
	cr := member("7", "a@b.c", "admin")
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	wantReqs := []request{{Method: http.MethodPatch, Path: "/organizations/acme/members/7", Body: `{"role":"admin"}`}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	wantAt := v1alpha1.OrganizationMemberObservation{
		ID:          7,
		Email:       "a@b.c",
		Role:        "admin",
		DateInvited: &metav1.Time{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		IsInvited:   true,
	}
	if diff := cmp.Diff(wantAt, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
}

func TestUpdateRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"PATCH /organizations/acme/members/7?": {Status: http.StatusBadRequest, Body: `{"role":["\"boss\" is not a valid choice."]}`},
	})
	if _, err := e.Update(context.Background(), member("7", "a@b.c", "boss")); err == nil || !strings.Contains(err.Error(), `cannot update organization member "7"`) {
		t.Fatalf("Update() error = %v, want cannot update organization member \"7\"", err)
	}
}

func TestDelete(t *testing.T) {
	e, got := server(t, map[string]reply{
		"DELETE /organizations/acme/members/7?": {Status: http.StatusNoContent},
	})
	if _, err := e.Delete(context.Background(), member("7", "a@b.c", "member")); err != nil {
		t.Fatal(err)
	}
	wantReqs := []request{{Method: http.MethodDelete, Path: "/organizations/acme/members/7"}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestDeleteRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"DELETE /organizations/acme/members/7?": {Status: http.StatusForbidden, Body: `{"detail":"forbidden"}`},
	})
	if _, err := e.Delete(context.Background(), member("7", "a@b.c", "member")); err == nil || !strings.Contains(err.Error(), `cannot delete organization member "7"`) {
		t.Fatalf("Delete() error = %v, want cannot delete organization member \"7\"", err)
	}
}

func TestDeleteAlreadyGone(t *testing.T) {
	e, got := server(t, map[string]reply{
		"DELETE /organizations/acme/members/7?": {Status: http.StatusNotFound, Body: `{"detail":"Not found."}`},
	})
	if _, err := e.Delete(context.Background(), member("7", "a@b.c", "member")); err != nil {
		t.Fatalf("Delete() error = %v, want nil for HTTP 404", err)
	}
	wantReqs := []request{{Method: http.MethodDelete, Path: "/organizations/acme/members/7"}}
	if diff := cmp.Diff(wantReqs, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestUpdateNonNumericExternalName(t *testing.T) {
	e, got := server(t, nil)
	if _, err := e.Update(context.Background(), member("alice", "a@b.c", "admin")); err == nil {
		t.Fatal("Update() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestDeleteNonNumericExternalName(t *testing.T) {
	e, got := server(t, nil)
	if _, err := e.Delete(context.Background(), member("alice", "a@b.c", "member")); err == nil {
		t.Fatal("Delete() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}
