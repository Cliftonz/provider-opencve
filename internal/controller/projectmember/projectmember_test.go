// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package projectmember

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	membersPath = "/organizations/acme/projects/prod/members"
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
			rep = reply{Status: http.StatusTeapot, Body: `{}`}
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

func member(externalName, membershipID, role string) *v1alpha1.ProjectMember {
	cr := &v1alpha1.ProjectMember{}
	cr.Spec.ForProvider = v1alpha1.ProjectMemberParameters{ProjectName: "prod", MembershipID: membershipID, Role: role}
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func listRequests(pages int) []request {
	reqs := []request{{Method: http.MethodGet, Path: membersPath, Query: "page=1&page_size=100"}}
	if pages == 2 {
		reqs = append(reqs, request{Method: http.MethodGet, Path: membersPath, Query: "page=2&page_size=100"})
	}
	return reqs
}

func TestObserveWithoutExternalName(t *testing.T) {
	e, got := server(t, nil)
	obs, err := e.Observe(t.Context(), member("", "42", "viewer"))
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

func TestObserveNonNumericExternalName(t *testing.T) {
	e, got := server(t, nil)
	if _, err := e.Observe(t.Context(), member("abc", "42", "viewer")); err == nil {
		t.Fatal("Observe() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestObserveFoundOnSecondPage(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":2,"next":"http://x/next","previous":null,"results":[
			{"id":3,"membership_id":40,"role":"project_admin","email":"bob@b.c","username":"bob"}]}`},
		listPage2: {Status: http.StatusOK, Body: `{"count":2,"next":null,"previous":"http://x/prev","results":[
			{"id":7,"membership_id":42,"role":"viewer","email":"alice@b.c","username":"alice"}]}`},
	})
	cr := member("7", "42", "viewer")
	obs, err := e.Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	if diff := cmp.Diff(listRequests(2), *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	want := v1alpha1.ProjectMemberObservation{ID: 7, MembershipID: 42, Role: "viewer", Email: "alice@b.c", Username: "alice"}
	if diff := cmp.Diff(want, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
	if !cr.GetCondition(xpv2.TypeReady).Equal(xpv2.Available()) {
		t.Errorf("ready condition = %v, want Available", cr.GetCondition(xpv2.TypeReady))
	}
}

func TestObserveUsernameNullOrMissingIsEmpty(t *testing.T) {
	for name, entry := range map[string]string{
		"null":    `{"id":7,"membership_id":42,"role":"viewer","email":"alice@b.c","username":null}`,
		"missing": `{"id":7,"membership_id":42,"role":"viewer","email":"alice@b.c"}`,
	} {
		t.Run(name, func(t *testing.T) {
			e, _ := server(t, map[string]reply{
				listPage1: {Status: http.StatusOK, Body: `{"count":1,"next":null,"previous":null,"results":[` + entry + `]}`},
			})
			cr := member("7", "42", "viewer")
			if _, err := e.Observe(t.Context(), cr); err != nil {
				t.Fatal(err)
			}
			if cr.Status.AtProvider.Username != "" {
				t.Errorf("username = %q, want empty", cr.Status.AtProvider.Username)
			}
		})
	}
}

func TestObserveResultsNullOrMissingIsEmpty(t *testing.T) {
	for name, body := range map[string]string{
		"null":    `{"count":0,"next":null,"previous":null,"results":null}`,
		"missing": `{"count":0,"next":null,"previous":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			e, got := server(t, map[string]reply{listPage1: {Status: http.StatusOK, Body: body}})
			obs, err := e.Observe(t.Context(), member("7", "42", "viewer"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
				t.Errorf("observation -want +got:\n%s", diff)
			}
			if diff := cmp.Diff(listRequests(1), *got); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

func TestObserveNextEmptyOrMissingEndsPaging(t *testing.T) {
	for name, next := range map[string]string{"empty": `"next":"",`, "missing": ``} {
		t.Run(name, func(t *testing.T) {
			e, got := server(t, map[string]reply{
				listPage1: {Status: http.StatusOK, Body: `{"count":1,` + next + `"results":[{"id":3,"membership_id":40,"role":"viewer","email":"bob@b.c","username":"bob"}]}`},
			})
			obs, err := e.Observe(t.Context(), member("7", "42", "viewer"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
				t.Errorf("observation -want +got:\n%s", diff)
			}
			if diff := cmp.Diff(listRequests(1), *got); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

func TestObserveNotFoundAfterAllPages(t *testing.T) {
	e, got := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":2,"next":"http://x/next","previous":null,"results":[
			{"id":3,"membership_id":40,"role":"viewer","email":"bob@b.c","username":"bob"}]}`},
		listPage2: {Status: http.StatusOK, Body: `{"count":2,"next":null,"previous":"http://x/prev","results":[
			{"id":4,"membership_id":41,"role":"viewer","email":"carol@b.c","username":"carol"}]}`},
	})
	obs, err := e.Observe(t.Context(), member("7", "42", "viewer"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	if diff := cmp.Diff(listRequests(2), *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestObserveListError(t *testing.T) {
	e, _ := server(t, map[string]reply{listPage1: {Status: http.StatusInternalServerError, Body: `{"detail":"boom"}`}})
	if _, err := e.Observe(t.Context(), member("7", "42", "viewer")); err == nil {
		t.Fatal("Observe() error = nil, want error for HTTP 500")
	}
}

func TestObserveProjectGoneIsAbsent(t *testing.T) {
	e, got := server(t, map[string]reply{listPage1: {Status: http.StatusNotFound, Body: `{"detail":"Not found."}`}})
	obs, err := e.Observe(t.Context(), member("7", "42", "viewer"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	if diff := cmp.Diff(listRequests(1), *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
}

func TestObserveRoleDiffers(t *testing.T) {
	e, _ := server(t, map[string]reply{
		listPage1: {Status: http.StatusOK, Body: `{"count":1,"next":null,"previous":null,"results":[
			{"id":7,"membership_id":42,"role":"viewer","email":"alice@b.c","username":"alice"}]}`},
	})
	obs, err := e.Observe(t.Context(), member("7", "42", "contributor"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
}

func TestCreate(t *testing.T) {
	e, got := server(t, map[string]reply{
		"POST " + membersPath + "?": {Status: http.StatusCreated, Body: `{"id":7,"membership_id":42,"role":"viewer","email":"alice@b.c","username":"alice"}`},
	})
	cr := member("", "42", "viewer")
	if _, err := e.Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := []request{{Method: http.MethodPost, Path: membersPath, Body: `{"membership_id":42,"role":"viewer"}`}}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	if name := meta.GetExternalName(cr); name != "7" {
		t.Errorf("external-name = %q, want %q", name, "7")
	}
}

func TestCreateNonNumericMembershipID(t *testing.T) {
	e, got := server(t, nil)
	cr := member("", "abc", "viewer")
	if _, err := e.Create(t.Context(), cr); err == nil {
		t.Fatal("Create() error = nil, want error for non-numeric membershipId")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
	if name := meta.GetExternalName(cr); name != "" {
		t.Errorf("external-name = %q, want empty", name)
	}
}

func TestCreateRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"POST " + membersPath + "?": {Status: http.StatusBadRequest, Body: `{"membership_id":["already a member"]}`},
	})
	cr := member("", "42", "viewer")
	if _, err := e.Create(t.Context(), cr); err == nil {
		t.Fatal("Create() error = nil, want error for HTTP 400")
	}
	if name := meta.GetExternalName(cr); name != "" {
		t.Errorf("external-name = %q, want empty", name)
	}
}

func TestUpdate(t *testing.T) {
	e, got := server(t, map[string]reply{
		"PATCH " + membersPath + "/7?": {Status: http.StatusOK, Body: `{"id":7,"membership_id":42,"role":"contributor","email":"alice@b.c","username":"alice"}`},
	})
	cr := member("7", "42", "contributor")
	if _, err := e.Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := []request{{Method: http.MethodPatch, Path: membersPath + "/7", Body: `{"role":"contributor"}`}}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	wantAt := v1alpha1.ProjectMemberObservation{ID: 7, MembershipID: 42, Role: "contributor", Email: "alice@b.c", Username: "alice"}
	if diff := cmp.Diff(wantAt, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
}

func TestUpdateRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"PATCH " + membersPath + "/7?": {Status: http.StatusForbidden, Body: `{"detail":"forbidden"}`},
	})
	if _, err := e.Update(t.Context(), member("7", "42", "contributor")); err == nil {
		t.Fatal("Update() error = nil, want error for HTTP 403")
	}
}

func TestUpdateNonNumericExternalName(t *testing.T) {
	e, got := server(t, nil)
	if _, err := e.Update(t.Context(), member("abc", "42", "contributor")); err == nil {
		t.Fatal("Update() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestDelete(t *testing.T) {
	for name, status := range map[string]int{"deleted": http.StatusNoContent, "already gone": http.StatusNotFound} {
		t.Run(name, func(t *testing.T) {
			e, got := server(t, map[string]reply{"DELETE " + membersPath + "/7?": {Status: status}})
			if _, err := e.Delete(t.Context(), member("7", "42", "viewer")); err != nil {
				t.Fatal(err)
			}
			want := []request{{Method: http.MethodDelete, Path: membersPath + "/7"}}
			if diff := cmp.Diff(want, *got); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

func TestDeleteRejected(t *testing.T) {
	e, _ := server(t, map[string]reply{
		"DELETE " + membersPath + "/7?": {Status: http.StatusForbidden, Body: `{"detail":"forbidden"}`},
	})
	if _, err := e.Delete(t.Context(), member("7", "42", "viewer")); err == nil {
		t.Fatal("Delete() error = nil, want error for HTTP 403")
	}
}

func TestDeleteNonNumericExternalName(t *testing.T) {
	e, got := server(t, nil)
	if _, err := e.Delete(t.Context(), member("abc", "42", "viewer")); err == nil {
		t.Fatal("Delete() error = nil, want error for non-numeric external-name")
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}
