// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1alpha1_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/cvetracker/v1alpha1"
)

func kubeClient(t *testing.T) client.Client {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "package", "crds", "cvetracker.opencve.m.clifton.quest_cvetrackers.yaml")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	s := runtime.NewScheme()
	if err := v1alpha1.SchemeBuilder.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	kube, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		t.Fatal(err)
	}
	return kube
}

func tracker(name string, fp v1alpha1.CveTrackerParameters) *v1alpha1.CveTracker {
	return &v1alpha1.CveTracker{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       v1alpha1.CveTrackerSpec{ForProvider: fp},
	}
}

const (
	cve     = "CVE-2026-1"
	project = "prod"
)

func str(s string) *string { return &s }

func TestValidation(t *testing.T) {
	kube := kubeClient(t)

	t.Run("ProjectNameIsImmutable", func(t *testing.T) {
		cr := tracker("project", v1alpha1.CveTrackerParameters{ProjectName: project, CveID: cve})
		if err := kube.Create(t.Context(), cr); err != nil {
			t.Fatal(err)
		}
		cr.Spec.ForProvider.ProjectName = "staging"
		err := kube.Update(t.Context(), cr)
		if err == nil || !strings.Contains(err.Error(), "projectName is immutable") {
			t.Errorf("update error = %v, want projectName is immutable", err)
		}
	})

	t.Run("ProjectSourceIsRequired", func(t *testing.T) {
		cr := tracker("no-project", v1alpha1.CveTrackerParameters{CveID: cve})
		err := kube.Create(t.Context(), cr)
		if err == nil || !strings.Contains(err.Error(), ": projectName, projectNameRef or projectNameSelector is required") {
			t.Errorf("create error = %v, want project source rejection", err)
		}
	})

	t.Run("ProjectNameCanBeResolvedLater", func(t *testing.T) {
		cr := tracker("resolved", v1alpha1.CveTrackerParameters{CveID: cve, ProjectNameRef: &xpv2.NamespacedReference{Name: "prod"}})
		if err := kube.Create(t.Context(), cr); err != nil {
			t.Fatal(err)
		}
		cr.Spec.ForProvider.ProjectName = project
		if err := kube.Update(t.Context(), cr); err != nil {
			t.Errorf("update error = %v, want none", err)
		}
	})

	t.Run("CveIDIsImmutable", func(t *testing.T) {
		cr := tracker("cve", v1alpha1.CveTrackerParameters{ProjectName: project, CveID: cve})
		if err := kube.Create(t.Context(), cr); err != nil {
			t.Fatal(err)
		}
		cr.Spec.ForProvider.CveID = "CVE-2026-2"
		err := kube.Update(t.Context(), cr)
		if err == nil || !strings.Contains(err.Error(), "cveId is immutable") {
			t.Errorf("update error = %v, want cveId is immutable", err)
		}
	})

	t.Run("CveIDMustBeACveIdentifier", func(t *testing.T) {
		for i, v := range []string{"cve-2026-1", "CVE-26-1", "CVE-2026-", "GHSA-2026-1"} {
			err := kube.Create(t.Context(), tracker(fmt.Sprintf("cve-rejected-%d", i), v1alpha1.CveTrackerParameters{ProjectName: project, CveID: v}))
			if err == nil || !strings.Contains(err.Error(), "cveId must match CVE-YYYY-N") {
				t.Errorf("%q: create error = %v, want cveId must match CVE-YYYY-N", v, err)
			}
		}
		if err := kube.Create(t.Context(), tracker("cve-accepted", v1alpha1.CveTrackerParameters{ProjectName: project, CveID: "CVE-2026-12345"})); err != nil {
			t.Error(err)
		}
	})

	t.Run("StatusOutsideEnumIsRejected", func(t *testing.T) {
		cr := tracker("bad-status", v1alpha1.CveTrackerParameters{ProjectName: project, CveID: cve, Status: str("done")})
		err := kube.Create(t.Context(), cr)
		if err == nil || !strings.Contains(err.Error(), "spec.forProvider.status") {
			t.Errorf("create error = %v, want rejection of spec.forProvider.status", err)
		}
	})

	t.Run("StatusInEnumIsAccepted", func(t *testing.T) {
		cr := tracker("good-status", v1alpha1.CveTrackerParameters{ProjectName: project, CveID: cve, Status: str("risk_accepted")})
		if err := kube.Create(t.Context(), cr); err != nil {
			t.Errorf("create error = %v, want none", err)
		}
	})
}
