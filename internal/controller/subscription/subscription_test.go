// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package subscription

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/subscription/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	projectName       = "prod"
	vendor            = "apple"
	emptySet          = `{"vendors":[],"products":{}}`
	subscriptionsPath = "/organizations/acme/projects/" + projectName + "/subscriptions"
)

type request struct {
	Method string
	Path   string
	Body   string
}

type fakeAPI struct {
	status   int
	response string
	received []request
}

func externalFor(t *testing.T, api *fakeAPI) *external {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		api.received = append(api.received, request{Method: r.Method, Path: r.URL.Path, Body: string(body)})
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

func subscriptions(externalName string, fp v1alpha1.ProjectSubscriptionsParameters) *v1alpha1.ProjectSubscriptions {
	cr := &v1alpha1.ProjectSubscriptions{Spec: v1alpha1.ProjectSubscriptionsSpec{ForProvider: fp}}
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
	got, err := externalFor(t, api).Observe(t.Context(), subscriptions("", v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api)
}

func TestObserveProjectNotFoundDoesNotExist(t *testing.T) {
	api := &fakeAPI{status: http.StatusNotFound, response: `{"detail":"Not found."}`}
	got, err := externalFor(t, api).Observe(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: subscriptionsPath})
}

func TestObserveServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	if _, err := externalFor(t, api).Observe(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName})); err == nil {
		t.Fatal("Observe() error = nil, want error for HTTP 500")
	}
}

func TestObserveExistingRecordsAtProviderAndAvailable(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["microsoft","apple"],"products":{"apple":["safari"]}}`}
	cr := subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{
		ProjectName: projectName,
		Vendors:     []string{vendor, "microsoft"},
		Products:    map[string][]string{vendor: {"safari"}},
	})
	got, err := externalFor(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}) {
		t.Errorf("observation = %+v, want existing and up to date", got)
	}
	wantAt := v1alpha1.ProjectSubscriptionsObservation{
		Vendors:  []string{"microsoft", vendor},
		Products: map[string][]string{vendor: {"safari"}},
	}
	if !reflect.DeepEqual(cr.Status.AtProvider, wantAt) {
		t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, wantAt)
	}
	if !cr.GetCondition(xpv2.TypeReady).Equal(xpv2.Available()) {
		t.Errorf("ready condition = %v, want Available", cr.GetCondition(xpv2.TypeReady))
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: subscriptionsPath})
}

func TestObserveMissingOrNullSetsAreEmpty(t *testing.T) {
	for name, remote := range map[string]string{
		"Missing": `{}`,
		"Null":    `{"vendors":null,"products":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			api := &fakeAPI{status: http.StatusOK, response: remote}
			cr := subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName})
			got, err := externalFor(t, api).Observe(t.Context(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}) {
				t.Errorf("observation = %+v, want existing and up to date", got)
			}
			if !reflect.DeepEqual(cr.Status.AtProvider, v1alpha1.ProjectSubscriptionsObservation{}) {
				t.Errorf("atProvider = %+v, want empty", cr.Status.AtProvider)
			}
		})
	}
}

func TestObserveUpToDateIsSetEquality(t *testing.T) {
	cases := map[string]struct {
		remote string
		fp     v1alpha1.ProjectSubscriptionsParameters
		want   bool
	}{
		"VendorOrderIgnored": {
			remote: `{"vendors":["b","a"],"products":{}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Vendors: []string{"a", "b"}},
			want:   true,
		},
		"ProductOrderIgnored": {
			remote: `{"vendors":[],"products":{"v":["y","x"]}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Products: map[string][]string{"v": {"x", "y"}}},
			want:   true,
		},
		"OmittedEqualsEmpty": {
			remote: emptySet,
			fp:     v1alpha1.ProjectSubscriptionsParameters{},
			want:   true,
		},
		"EmptyProductListEqualsMissingVendor": {
			remote: emptySet,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Products: map[string][]string{"v": {}}},
			want:   true,
		},
		"ExtraRemoteVendor": {
			remote: `{"vendors":["a","b"],"products":{}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Vendors: []string{"a"}},
			want:   false,
		},
		"MissingRemoteVendor": {
			remote: `{"vendors":["a"],"products":{}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Vendors: []string{"a", "b"}},
			want:   false,
		},
		"ExtraRemoteProductVendor": {
			remote: `{"vendors":[],"products":{"v":["x"]}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{},
			want:   false,
		},
		"DifferentProducts": {
			remote: `{"vendors":[],"products":{"v":["x"]}}`,
			fp:     v1alpha1.ProjectSubscriptionsParameters{Products: map[string][]string{"v": {"y"}}},
			want:   false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := &fakeAPI{status: http.StatusOK, response: tc.remote}
			tc.fp.ProjectName = projectName
			got, err := externalFor(t, api).Observe(t.Context(), subscriptions(projectName, tc.fp))
			if err != nil {
				t.Fatal(err)
			}
			if got.ResourceUpToDate != tc.want {
				t.Errorf("ResourceUpToDate = %v, want %v", got.ResourceUpToDate, tc.want)
			}
		})
	}
}

func TestCreatePutsDesiredSetAndSetsExternalName(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["apple"],"products":{"apple":["safari"]}}`}
	cr := subscriptions("", v1alpha1.ProjectSubscriptionsParameters{
		ProjectName: projectName,
		Vendors:     []string{vendor},
		Products:    map[string][]string{vendor: {"safari"}},
	})
	if _, err := externalFor(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPut, Path: subscriptionsPath, Body: `{"products":{"apple":["safari"]},"vendors":["apple"]}`})
	if name := meta.GetExternalName(cr); name != projectName {
		t.Errorf("external-name = %q, want %q", name, projectName)
	}
}

func TestCreateOmittedSetsPutEmpty(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: emptySet}
	if _, err := externalFor(t, api).Create(t.Context(), subscriptions("", v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPut, Path: subscriptionsPath, Body: `{"products":{},"vendors":[]}`})
}

func TestCreateRejectedIsAnErrorAndLeavesExternalNameEmpty(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"vendors":["unknown vendor"]}`}
	cr := subscriptions("", v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{"nope"}})
	if _, err := externalFor(t, api).Create(t.Context(), cr); err == nil {
		t.Fatal("Create() error = nil, want error for HTTP 400")
	}
	if name := meta.GetExternalName(cr); name != "" {
		t.Errorf("external-name = %q, want empty", name)
	}
}

func TestUpdatePutsDesiredSet(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["apple"],"products":{}}`}
	if _, err := externalFor(t, api).Update(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPut, Path: subscriptionsPath, Body: `{"products":{},"vendors":["apple"]}`})
}

func TestUpdateRejectedIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"vendors":["unknown vendor"]}`}
	if _, err := externalFor(t, api).Update(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{"nope"}})); err == nil {
		t.Fatal("Update() error = nil, want error for HTTP 400")
	}
}

func TestDeletePutsEmptySet(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: emptySet}
	if _, err := externalFor(t, api).Delete(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}})); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPut, Path: subscriptionsPath, Body: `{"products":{},"vendors":[]}`})
}

func TestDeleteProjectNotFoundSucceeds(t *testing.T) {
	api := &fakeAPI{status: http.StatusNotFound, response: `{"detail":"Not found."}`}
	if _, err := externalFor(t, api).Delete(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName})); err != nil {
		t.Fatalf("Delete() error = %v, want nil for HTTP 404", err)
	}
}

func TestDeleteRejectedIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusForbidden, response: `{"detail":"forbidden"}`}
	if _, err := externalFor(t, api).Delete(t.Context(), subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName})); err == nil {
		t.Fatal("Delete() error = nil, want error for HTTP 403")
	}
}

func deleted(cr *v1alpha1.ProjectSubscriptions) *v1alpha1.ProjectSubscriptions {
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)
	return cr
}

func TestObserveDeletedWithEmptyRemoteDoesNotExist(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: emptySet}
	cr := deleted(subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}}))
	got, err := externalFor(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
}

func TestObserveDeletedWithRemainingRemoteExists(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["apple"],"products":{}}`}
	cr := deleted(subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}}))
	got, err := externalFor(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ResourceExists {
		t.Errorf("observation = %+v, want resource to exist", got)
	}
}

func TestCreateRecordsAtProviderFromResponse(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["apple"],"products":{"apple":["safari"]}}`}
	cr := subscriptions("", v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}})
	if _, err := externalFor(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.ProjectSubscriptionsObservation{Vendors: []string{vendor}, Products: map[string][]string{vendor: {"safari"}}}
	if !reflect.DeepEqual(cr.Status.AtProvider, want) {
		t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, want)
	}
}

func TestUpdateRecordsAtProviderFromResponse(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"vendors":["apple"],"products":null}`}
	cr := subscriptions(projectName, v1alpha1.ProjectSubscriptionsParameters{ProjectName: projectName, Vendors: []string{vendor}})
	cr.Status.AtProvider = v1alpha1.ProjectSubscriptionsObservation{Vendors: []string{"stale"}}
	if _, err := externalFor(t, api).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.ProjectSubscriptionsObservation{Vendors: []string{vendor}}
	if !reflect.DeepEqual(cr.Status.AtProvider, want) {
		t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, want)
	}
}
