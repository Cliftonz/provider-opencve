// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	projectName = "prod"
	projectPath = "/organizations/acme/projects/prod"
)

type request struct {
	Method string
	Path   string
	Body   map[string]any
}

type fakeAPI struct {
	status   int
	response string
	received []request
}

func connectTo(t *testing.T, api *fakeAPI) *external {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		api.received = append(api.received, request{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(api.status)
		_, _ = io.WriteString(w, api.response)
	}))
	t.Cleanup(srv.Close)
	c, err := opencve.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &external{client: c, org: "acme"}
}

func project(externalName string, fp v1alpha1.ProjectParameters) *v1alpha1.Project {
	cr := &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{ForProvider: fp}}
	meta.SetExternalName(cr, externalName)
	return cr
}

func assertReceived(t *testing.T, api *fakeAPI, want ...request) {
	t.Helper()
	if !reflect.DeepEqual(api.received, want) {
		t.Errorf("received requests\n got: %#v\nwant: %#v", api.received, want)
	}
}

func TestObserveWithoutExternalNameDoesNotExist(t *testing.T) {
	api := &fakeAPI{}
	got, err := connectTo(t, api).Observe(t.Context(), project("", v1alpha1.ProjectParameters{Name: projectName}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api)
}

func TestObserveNotFoundDoesNotExist(t *testing.T) {
	api := &fakeAPI{status: http.StatusNotFound, response: `{"detail":"Not found."}`}
	got, err := connectTo(t, api).Observe(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: projectPath})
}

func TestObserveServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	_, err := connectTo(t, api).Observe(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName}))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want an error reporting HTTP 500", err)
	}
}

func TestObserveExistingProjectLateInitializesActive(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":false,"subscriptions_count":2}`}
	cr := project(projectName, v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production")})
	got, err := connectTo(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	want := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	if wantSpec := (v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production"), Active: ptr(false)}); !reflect.DeepEqual(cr.Spec.ForProvider, wantSpec) {
		t.Errorf("forProvider = %+v, want %+v", cr.Spec.ForProvider, wantSpec)
	}
	wantAt := v1alpha1.ProjectObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: projectName, Description: ptr("Production"), Active: ptr(false), SubscriptionsCount: ptr(2)}
	if !reflect.DeepEqual(cr.Status.AtProvider, wantAt) {
		t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, wantAt)
	}
	if c := cr.GetCondition(xpv2.TypeReady); c.Reason != xpv2.ReasonAvailable {
		t.Errorf("ready condition = %+v, want Available", c)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: projectPath})
}

func ptr[T any](v T) *T {
	return &v
}

func TestObserveUpToDateComparesDescriptionAndActive(t *testing.T) {
	cases := map[string]struct {
		remote string
		spec   v1alpha1.ProjectParameters
		want   bool
	}{
		"Matching": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production"), Active: ptr(true)},
			want:   true,
		},
		"BothDescriptionsNull": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":null,"active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Active: ptr(true)},
			want:   true,
		},
		"DescriptionOmittedRemoteEmpty": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"","active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Active: ptr(true)},
			want:   true,
		},
		"DescriptionOmittedRemoteMissing": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Active: ptr(true)},
			want:   true,
		},
		"DescriptionEmptyRemoteNull": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":null,"active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Description: ptr(""), Active: ptr(true)},
			want:   true,
		},
		"DescriptionOmittedButSetRemotely": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Active: ptr(true)},
			want:   false,
		},
		"DescriptionDiffers": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":true}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Staging"), Active: ptr(true)},
			want:   false,
		},
		"ActiveDiffers": {
			remote: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":false}`,
			spec:   v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production"), Active: ptr(true)},
			want:   false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := &fakeAPI{status: http.StatusOK, response: tc.remote}
			got, err := connectTo(t, api).Observe(t.Context(), project(projectName, tc.spec))
			if err != nil {
				t.Fatal(err)
			}
			if got.ResourceUpToDate != tc.want {
				t.Errorf("ResourceUpToDate = %v, want %v", got.ResourceUpToDate, tc.want)
			}
		})
	}
}

func TestObserveRemoteWithoutActiveDoesNotLateInitialize(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production"}`}
	cr := project(projectName, v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production")})
	got, err := connectTo(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	want := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	if cr.Spec.ForProvider.Active != nil {
		t.Errorf("forProvider.active = %v, want nil", *cr.Spec.ForProvider.Active)
	}
}

func TestCreatePostsProjectAndSetsExternalName(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Production","active":true}`}
	cr := project("", v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Production"), Active: ptr(true)})
	if _, err := connectTo(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	if got := meta.GetExternalName(cr); got != projectName {
		t.Errorf("external-name = %q, want %q", got, projectName)
	}
	assertReceived(t, api, request{
		Method: http.MethodPost,
		Path:   "/organizations/acme/projects",
		Body:   map[string]any{"name": projectName, "description": "Production", "active": true},
	})
}

func TestCreateRejectedIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"name":["invalid"]}`}
	_, err := connectTo(t, api).Create(t.Context(), project("", v1alpha1.ProjectParameters{Name: "bad name"}))
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %v, want an error reporting HTTP 400", err)
	}
}

func TestUpdatePatchesDescriptionAndActiveWithExplicitNull(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":null,"active":false}`}
	if _, err := connectTo(t, api).Update(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName, Active: ptr(false)})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{
		Method: http.MethodPatch,
		Path:   projectPath,
		Body:   map[string]any{"description": nil, "active": false},
	})
}

func TestUpdateSetsAtProviderFromResponse(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":"Staging","active":false,"subscriptions_count":3}`}
	cr := project(projectName, v1alpha1.ProjectParameters{Name: projectName, Description: ptr("Staging"), Active: ptr(false)})
	cr.Status.AtProvider = v1alpha1.ProjectObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: projectName, Description: ptr("Production"), Active: ptr(true), SubscriptionsCount: ptr(3)}
	if _, err := connectTo(t, api).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.ProjectObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: projectName, Description: ptr("Staging"), Active: ptr(false), SubscriptionsCount: ptr(3)}
	if !reflect.DeepEqual(cr.Status.AtProvider, want) {
		t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, want)
	}
}

func TestCreateSendsEmptyDescriptionAsNull(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":null,"active":true}`}
	if _, err := connectTo(t, api).Create(t.Context(), project("", v1alpha1.ProjectParameters{Name: projectName, Description: ptr("")})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{
		Method: http.MethodPost,
		Path:   "/organizations/acme/projects",
		Body:   map[string]any{"name": projectName, "description": nil},
	})
}

func TestUpdateSendsEmptyDescriptionAsNull(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"prod","description":null,"active":true}`}
	if _, err := connectTo(t, api).Update(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName, Description: ptr(""), Active: ptr(true)})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{
		Method: http.MethodPatch,
		Path:   projectPath,
		Body:   map[string]any{"description": nil, "active": true},
	})
}

func TestUpdateRejectedIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"active":["invalid"]}`}
	_, err := connectTo(t, api).Update(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName, Active: ptr(true)}))
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %v, want an error reporting HTTP 400", err)
	}
}

func TestDeleteSendsDelete(t *testing.T) {
	api := &fakeAPI{status: http.StatusNoContent}
	if _, err := connectTo(t, api).Delete(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodDelete, Path: projectPath})
}

func TestDeleteNotFoundSucceeds(t *testing.T) {
	api := &fakeAPI{status: http.StatusNotFound, response: `{"detail":"Not found."}`}
	if _, err := connectTo(t, api).Delete(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodDelete, Path: projectPath})
}

func TestDeleteRejectedIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusForbidden, response: `{"detail":"Forbidden."}`}
	_, err := connectTo(t, api).Delete(t.Context(), project(projectName, v1alpha1.ProjectParameters{Name: projectName}))
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %v, want an error reporting HTTP 403", err)
	}
}
