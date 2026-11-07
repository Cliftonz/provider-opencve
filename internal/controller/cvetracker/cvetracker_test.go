// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package cvetracker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/cvetracker/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	cveID   = "CVE-2026-41084"
	cvePath = "/organizations/acme/projects/prod/cves/" + cveID
)

const assignedTracker = `{"status":"to_evaluate","assignee":{"email":"user@example.com"},"assigned_at":"2026-07-03T09:14:59Z","updated_at":"2026-07-03T09:15:01Z"}`

type request struct {
	Method string
	Path   string
	Body   string
}

type reply struct {
	Status int
	Body   string
}

func server(t *testing.T, rep reply) (*external, *[]request) {
	t.Helper()
	var got []request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, request{Method: r.Method, Path: r.URL.Path, Body: string(body)})
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

func str(s string) *string { return &s }

func tracker(status, assignee *string) *v1alpha1.CveTracker {
	cr := &v1alpha1.CveTracker{}
	cr.Spec.ForProvider = v1alpha1.CveTrackerParameters{ProjectName: "prod", CveID: cveID, Status: status, Assignee: assignee}
	return cr
}

func projectCve(tracker string) reply {
	return reply{Status: http.StatusOK, Body: `{"created_at":"2026-06-01T07:51:56Z","updated_at":"2026-06-02T18:49:13Z","cve_id":"CVE-2026-41084",
		"description":"d","title":"Airflow bypass","metrics":{},"tracker":` + tracker + `}`}
}

func TestObserveUpToDate(t *testing.T) {
	e, got := server(t, projectCve(assignedTracker))
	cr := tracker(str("to_evaluate"), str("user@example.com"))
	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
	if diff := cmp.Diff([]request{{Method: http.MethodGet, Path: cvePath}}, *got); diff != "" {
		t.Errorf("requests -want +got:\n%s", diff)
	}
	wantAt := v1alpha1.CveTrackerObservation{CveID: cveID, Title: "Airflow bypass", Status: "to_evaluate", Assignee: "user@example.com"}
	if diff := cmp.Diff(wantAt, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
	if !cr.GetCondition(xpv2.TypeReady).Equal(xpv2.Available()) {
		t.Errorf("ready condition = %v, want Available", cr.GetCondition(xpv2.TypeReady))
	}
}

func TestObserveDrift(t *testing.T) {
	cases := map[string]struct {
		tracker  string
		status   *string
		assignee *string
		want     bool
	}{
		"StatusDiffers":                 {tracker: assignedTracker, status: str("resolved"), assignee: str("user@example.com")},
		"AssigneeDiffers":               {tracker: assignedTracker, status: str("to_evaluate"), assignee: str("other@example.com")},
		"OmittedStatusWhileRemoteSet":   {tracker: assignedTracker, assignee: str("user@example.com")},
		"OmittedAssigneeWhileRemoteSet": {tracker: assignedTracker, status: str("to_evaluate")},
		"NullRemoteAndOmitted":          {tracker: `{"status":null,"assignee":null}`, want: true},
		"MissingKeysAndEmptyStrings":    {tracker: `{}`, status: str(""), assignee: str(""), want: true},
		"NullRemoteWhileSpecSet":        {tracker: `{"status":null,"assignee":null}`, status: str("resolved"), assignee: str("user@example.com")},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e, _ := server(t, projectCve(tc.tracker))
			obs, err := e.Observe(context.Background(), tracker(tc.status, tc.assignee))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: tc.want}, obs); diff != "" {
				t.Errorf("observation -want +got:\n%s", diff)
			}
		})
	}
}

func TestObserveNullOrMissingIsEmpty(t *testing.T) {
	cases := map[string]string{
		"NullTrackerAndTitle":    `{"created_at":"2026-06-01T07:51:56Z","updated_at":"2026-06-02T18:49:13Z","cve_id":"CVE-2026-41084","title":null,"tracker":null}`,
		"MissingTrackerAndTitle": `{"created_at":"2026-06-01T07:51:56Z","updated_at":"2026-06-02T18:49:13Z","cve_id":"CVE-2026-41084"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			e, _ := server(t, reply{Status: http.StatusOK, Body: body})
			cr := tracker(nil, nil)
			obs, err := e.Observe(context.Background(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, obs); diff != "" {
				t.Errorf("observation -want +got:\n%s", diff)
			}
			if diff := cmp.Diff(v1alpha1.CveTrackerObservation{CveID: cveID}, cr.Status.AtProvider); diff != "" {
				t.Errorf("atProvider -want +got:\n%s", diff)
			}
		})
	}
}

func TestObserveNotFound(t *testing.T) {
	e, _ := server(t, reply{Status: http.StatusNotFound, Body: `{"detail":"Not found."}`})
	obs, err := e.Observe(context.Background(), tracker(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(managed.ExternalObservation{}, obs); diff != "" {
		t.Errorf("observation -want +got:\n%s", diff)
	}
}

func TestObserveError(t *testing.T) {
	e, _ := server(t, reply{Status: http.StatusInternalServerError, Body: `{"detail":"boom"}`})
	_, err := e.Observe(context.Background(), tracker(nil, nil))
	if err == nil || !strings.Contains(err.Error(), `cannot get CVE tracker "CVE-2026-41084"`) {
		t.Fatalf("Observe() error = %v, want cannot get CVE tracker %q", err, cveID)
	}
}

func TestObserveDeletedIsAbsent(t *testing.T) {
	e, got := server(t, projectCve(assignedTracker))
	cr := tracker(str("to_evaluate"), str("user@example.com"))
	cr.SetDeletionTimestamp(&metav1.Time{Time: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)})
	obs, err := e.Observe(context.Background(), cr)
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

func TestObserveWrongTrackerShapePanics(t *testing.T) {
	e, _ := server(t, projectCve(`{"status":7,"assignee":null}`))
	defer func() {
		if recover() == nil {
			t.Error("Observe() did not panic for a non-string status")
		}
	}()
	_, _ = e.Observe(context.Background(), tracker(nil, nil))
}

func TestCreateFails(t *testing.T) {
	e, got := server(t, reply{Status: http.StatusOK, Body: `{}`})
	if _, err := e.Create(context.Background(), tracker(nil, nil)); !errors.Is(err, errCannotCreate) {
		t.Fatalf("Create() error = %v, want %v", err, errCannotCreate)
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}

func TestUpdate(t *testing.T) {
	cases := map[string]struct {
		status   *string
		assignee *string
		body     string
	}{
		"Values":       {status: str("resolved"), assignee: str("user@example.com"), body: `{"assignee":"user@example.com","status":"resolved"}`},
		"Omitted":      {body: `{"assignee":null,"status":null}`},
		"EmptyStrings": {status: str(""), assignee: str(""), body: `{"assignee":null,"status":null}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e, got := server(t, projectCve(assignedTracker))
			if _, err := e.Update(context.Background(), tracker(tc.status, tc.assignee)); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff([]request{{Method: http.MethodPatch, Path: cvePath, Body: tc.body}}, *got); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

func TestUpdateRefreshesAtProvider(t *testing.T) {
	e, _ := server(t, projectCve(`{"status":"resolved","assignee":{"email":"user@example.com"}}`))
	cr := tracker(str("resolved"), str("user@example.com"))
	cr.Status.AtProvider = v1alpha1.CveTrackerObservation{CveID: cveID, Title: "Airflow bypass", Status: "to_evaluate"}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.CveTrackerObservation{CveID: cveID, Title: "Airflow bypass", Status: "resolved", Assignee: "user@example.com"}
	if diff := cmp.Diff(want, cr.Status.AtProvider); diff != "" {
		t.Errorf("atProvider -want +got:\n%s", diff)
	}
}

func TestUpdateRejected(t *testing.T) {
	e, _ := server(t, reply{Status: http.StatusBadRequest, Body: `{"assignee":["not a member"]}`})
	_, err := e.Update(context.Background(), tracker(nil, str("x@example.com")))
	if err == nil || !strings.Contains(err.Error(), `cannot update CVE tracker "CVE-2026-41084"`) {
		t.Fatalf("Update() error = %v, want cannot update CVE tracker %q", err, cveID)
	}
}

func TestDeleteLeavesRemoteUntouched(t *testing.T) {
	e, got := server(t, reply{Status: http.StatusOK, Body: `{}`})
	if _, err := e.Delete(context.Background(), tracker(nil, nil)); err != nil {
		t.Fatal(err)
	}
	if len(*got) != 0 {
		t.Errorf("requests = %v, want none", *got)
	}
}
