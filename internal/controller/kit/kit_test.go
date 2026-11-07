// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package kit

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/oapi-codegen/nullable"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

func respond(code int, status, body string) opencve.OrganizationsProjectsRetrieveResponse {
	return opencve.OrganizationsProjectsRetrieveResponse{
		Body:         []byte(body),
		HTTPResponse: &http.Response{StatusCode: code, Status: status},
	}
}

func TestCheckStatusAcceptsSuccess(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent} {
		if err := CheckStatus("get project", respond(code, http.StatusText(code), "")); err != nil {
			t.Errorf("CheckStatus(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckStatusReportsActionStatusAndBody(t *testing.T) {
	for _, code := range []int{http.StatusMultipleChoices, http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		status := http.StatusText(code)
		err := CheckStatus("get project", respond(code, status, `{"detail":"nope"}`))
		if err == nil {
			t.Fatalf("CheckStatus(%d) = nil, want error", code)
		}
		for _, want := range []string{"get project", status, `{"detail":"nope"}`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("CheckStatus(%d) = %q, want it to contain %q", code, err, want)
			}
		}
	}
}

func TestNotFound(t *testing.T) {
	for code, want := range map[int]bool{http.StatusNotFound: true, http.StatusOK: false, http.StatusGone: false, http.StatusInternalServerError: false} {
		if got := NotFound(respond(code, http.StatusText(code), "")); got != want {
			t.Errorf("NotFound(%d) = %v, want %v", code, got, want)
		}
	}
}

func TestConnectHandsClientAndOrganizationToExternal(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, v1beta1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme} {
		if err := add(s); err != nil {
			t.Fatal(err)
		}
	}
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
	mapper.Add(v1beta1.ProviderConfigGroupVersionKind, meta.RESTScopeNamespace)
	mapper.Add(v1beta1.ProviderConfigUsageGroupVersionKind, meta.RESTScopeNamespace)
	mapper.Add(v1alpha1.ProjectGroupVersionKind, meta.RESTScopeNamespace)
	pc := &v1beta1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "team-a"},
		Spec: v1beta1.ProviderConfigSpec{Credentials: v1beta1.ProviderCredentials{
			Source: xpv2.CredentialsSourceSecret,
			CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: &xpv2.SecretKeySelector{
				SecretReference: xpv2.SecretReference{Name: "opencve-creds"},
				Key:             "credentials",
			}},
		}},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "opencve-creds", Namespace: "team-a"},
		Data:       map[string][]byte{"credentials": []byte(`{"token":"t","organization":"acme","url":"` + srv.URL + `"}`)},
	}
	kube := kfake.NewClientBuilder().WithScheme(s).WithRESTMapper(mapper).WithObjects(pc, secret).Build()

	cr := &v1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "team-a", UID: "project-uid"}}
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: v1beta1.ProviderConfigKind, Name: "default"})

	var gotClient *opencve.ClientWithResponses
	var gotOrg string
	newExternal := func(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.Project] {
		gotClient, gotOrg = c, org
		return nil
	}

	if _, err := connector(kube, newExternal).Connect(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	if gotOrg != "acme" {
		t.Errorf("org = %q, want %q", gotOrg, "acme")
	}
	if _, err := gotClient.OrganizationsProjectsRetrieveWithResponse(t.Context(), gotOrg, "prod"); err != nil {
		t.Fatal(err)
	}
	if want := "/api/v2/organizations/acme/projects/prod"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
	if want := "Bearer t"; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
}

func ptr[T any](v T) *T { return &v }

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestValueOfNilIsZero(t *testing.T) {
	if got := Value[string](nil); got != "" {
		t.Errorf("Value(nil) = %q, want empty", got)
	}
	if got := Value[[]string](nil); got != nil {
		t.Errorf("Value(nil) = %v, want nil", got)
	}
}

func TestValueOfPointerIsPointee(t *testing.T) {
	if got := Value(ptr("x")); got != "x" {
		t.Errorf("Value(&x) = %q, want x", got)
	}
}

func TestNullableOfNilOrBlankIsNull(t *testing.T) {
	for _, s := range []*string{nil, ptr("")} {
		if got := NullableOf[string](s); !got.IsNull() {
			t.Errorf("NullableOf(%v) = %v, want explicit null", s, got)
		}
	}
}

func TestNullableOfValueIsValue(t *testing.T) {
	if got := encode(t, NullableOf[opencve.PeriodFrequencyEnum](ptr("weekly"))); got != `"weekly"` {
		t.Errorf("NullableOf(weekly) = %s, want \"weekly\"", got)
	}
}

func TestOptionalOfNullOrMissingIsNil(t *testing.T) {
	for _, n := range []nullable.Nullable[string]{nullable.NewNullNullable[string](), {}} {
		if got := Optional(n); got != nil {
			t.Errorf("Optional(%v) = %q, want nil", n, *got)
		}
	}
}

func TestOptionalOfValueIsValue(t *testing.T) {
	if got := Optional(nullable.NewNullableWithValue(opencve.Weekly)); got == nil || *got != "weekly" {
		t.Errorf("Optional(weekly) = %v, want weekly", got)
	}
}

func TestNullableFromNilOrBlankIsNull(t *testing.T) {
	for _, s := range []*string{nil, ptr("")} {
		got, err := NullableFrom(s, (*opencve.AutomationWriteRequest_Frequency).FromPeriodFrequencyEnum)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsNull() {
			t.Errorf("NullableFrom(%v) = %v, want explicit null", s, got)
		}
	}
}

func TestNullableFromValueIsUnionValue(t *testing.T) {
	got, err := NullableFrom(ptr("monday"), (*opencve.AutomationWriteRequest_ScheduleWeekday).FromScheduleWeekdayEnum)
	if err != nil {
		t.Fatal(err)
	}
	if body := encode(t, opencve.AutomationWriteRequest{ScheduleWeekday: got}); body != `{"name":"","schedule_weekday":"monday"}` {
		t.Errorf("request = %s, want schedule_weekday monday", body)
	}
}

func TestOptionalAsNullOrMissingIsNil(t *testing.T) {
	for _, body := range []string{`{"frequency":null}`, `{}`} {
		var a opencve.Automation
		if err := json.Unmarshal([]byte(body), &a); err != nil {
			t.Fatal(err)
		}
		got, err := OptionalAs(a.Frequency, opencve.Automation_Frequency.AsPeriodFrequencyEnum)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("OptionalAs(%s) = %q, want nil", body, *got)
		}
	}
}

func TestOptionalOfBlankIsNil(t *testing.T) {
	if got := Optional(nullable.NewNullableWithValue("")); got != nil {
		t.Errorf("Optional(\"\") = %q, want nil", *got)
	}
}

func TestOptionalAsBlankIsNil(t *testing.T) {
	var a opencve.Automation
	if err := json.Unmarshal([]byte(`{"frequency":""}`), &a); err != nil {
		t.Fatal(err)
	}
	got, err := OptionalAs(a.Frequency, opencve.Automation_Frequency.AsPeriodFrequencyEnum)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("OptionalAs(\"\") = %q, want nil", *got)
	}
}

func TestOptionalAsValueIsValue(t *testing.T) {
	var a opencve.Automation
	if err := json.Unmarshal([]byte(`{"frequency":"daily"}`), &a); err != nil {
		t.Fatal(err)
	}
	got, err := OptionalAs(a.Frequency, opencve.Automation_Frequency.AsPeriodFrequencyEnum)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != "daily" {
		t.Errorf("OptionalAs(daily) = %v, want daily", got)
	}
}

func TestOptionalAsReportsUnionError(t *testing.T) {
	var a opencve.Automation
	if err := json.Unmarshal([]byte(`{"frequency":7}`), &a); err != nil {
		t.Fatal(err)
	}
	if _, err := OptionalAs(a.Frequency, opencve.Automation_Frequency.AsPeriodFrequencyEnum); err == nil {
		t.Error("OptionalAs(7) = nil error, want error")
	}
}

type pageOf struct {
	code  int
	ids   []int
	next  nullable.Nullable[string]
	raise error
}

func pages(t *testing.T, ps ...pageOf) (func(page, size int) (opencve.OrganizationsMembersListResponse, error), *[]int) {
	var fetched []int
	return func(page, size int) (opencve.OrganizationsMembersListResponse, error) {
		fetched = append(fetched, page)
		if size != 100 {
			t.Errorf("page size = %d, want 100", size)
		}
		p := ps[page-1]
		r := opencve.OrganizationsMembersListResponse{HTTPResponse: &http.Response{StatusCode: p.code, Status: http.StatusText(p.code)}}
		if p.code == http.StatusOK {
			r.JSON200 = &opencve.PaginatedMembershipList{Next: p.next}
			for _, id := range p.ids {
				r.JSON200.Results = append(r.JSON200.Results, opencve.Membership{Id: ptr(id)})
			}
		}
		return r, p.raise
	}, &fetched
}

func members(r opencve.OrganizationsMembersListResponse) ([]opencve.Membership, nullable.Nullable[string]) {
	return r.JSON200.Results, r.JSON200.Next
}

func id(want int) func(opencve.Membership) bool {
	return func(m opencve.Membership) bool { return *m.Id == want }
}

func more() nullable.Nullable[string] { return nullable.NewNullableWithValue("https://x/?page=next") }

func TestFindPagedFindsOnLaterPage(t *testing.T) {
	fetch, fetched := pages(t, pageOf{code: http.StatusOK, ids: []int{1, 2}, next: more()}, pageOf{code: http.StatusOK, ids: []int{3, 4}})
	m, found, err := FindPaged("list members", fetch, members, id(4))
	if err != nil {
		t.Fatal(err)
	}
	if !found || *m.Id != 4 {
		t.Errorf("FindPaged = %v, %v, want member 4", m, found)
	}
	if len(*fetched) != 2 {
		t.Errorf("fetched pages %v, want [1 2]", *fetched)
	}
}

func TestFindPagedStopsWhenNoNext(t *testing.T) {
	for name, next := range map[string]nullable.Nullable[string]{"null": nullable.NewNullNullable[string](), "missing": {}, "blank": nullable.NewNullableWithValue("")} {
		fetch, fetched := pages(t, pageOf{code: http.StatusOK, ids: []int{1}, next: next}, pageOf{code: http.StatusOK, ids: []int{2}})
		_, found, err := FindPaged("list members", fetch, members, id(2))
		if err != nil {
			t.Fatal(err)
		}
		if found || len(*fetched) != 1 {
			t.Errorf("next %s: found = %v after pages %v, want absent after [1]", name, found, *fetched)
		}
	}
}

func TestFindPagedListNotFoundIsAbsent(t *testing.T) {
	fetch, _ := pages(t, pageOf{code: http.StatusNotFound})
	_, found, err := FindPaged("list members", fetch, members, id(1))
	if err != nil || found {
		t.Errorf("FindPaged on 404 = %v, %v, want absent without error", found, err)
	}
}

func TestFindPagedServerErrorIsError(t *testing.T) {
	fetch, _ := pages(t, pageOf{code: http.StatusInternalServerError})
	if _, _, err := FindPaged("list members", fetch, members, id(1)); err == nil {
		t.Error("FindPaged on 500 = nil error, want error")
	}
}

func TestFindPagedReturnsFetchError(t *testing.T) {
	boom := errors.New("boom")
	fetch, _ := pages(t, pageOf{code: http.StatusOK, raise: boom})
	if _, _, err := FindPaged("list members", fetch, members, id(1)); !errors.Is(err, boom) {
		t.Errorf("FindPaged error = %v, want boom", err)
	}
}
