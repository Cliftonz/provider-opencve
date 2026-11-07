// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package opencve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestOrganizationsProjectsRetrieve(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/organizations/acme/projects/prod" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "e9f53830-1a88-4292-9995-0748d2d7bcfb",
			"created_at": "2026-06-29T10:31:05.417496Z",
			"updated_at": "2026-06-29T10:36:38.084220Z",
			"name": "prod",
			"description": "Production",
			"active": true,
			"subscriptions_count": 1,
			"subscriptions": {"vendors": ["fortinet"], "products": {}}
		}`))
	}))
	defer srv.Close()

	c, err := NewClientWithResponses(srv.URL+"/api/v2", WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer secret")
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.OrganizationsProjectsRetrieveWithResponse(context.Background(), "acme", "prod")
	if err != nil {
		t.Fatal(err)
	}
	if resp.JSON200 == nil {
		t.Fatalf("status %d, body %s", resp.StatusCode(), resp.Body)
	}
	if resp.JSON200.Name != "prod" {
		t.Errorf("name = %q, want prod", resp.JSON200.Name)
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want Bearer secret", gotAuth)
	}
}

func TestGeneratedClientStartsWithBoilerplate(t *testing.T) {
	boilerplate, err := os.ReadFile("../../../hack/boilerplate.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile("client.gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(generated), string(boilerplate)) {
		t.Errorf("client.gen.go does not start with hack/boilerplate.go.txt")
	}
}

func TestPatchedProjectWriteRequestSendsExplicitNullDescription(t *testing.T) {
	var req PatchedProjectWriteRequest
	if err := json.Unmarshal([]byte(`{"description":null}`), &req); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"description":null`) {
		t.Errorf("marshaled %s, want \"description\":null", b)
	}
}
