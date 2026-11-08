// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1alpha1_test

import (
	"path/filepath"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
)

func apiServer(t *testing.T) client.Client {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "package", "crds", "projectmember.opencve.m.clifton.quest_projectmembers.yaml")},
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

func projectMember(name string, p v1alpha1.ProjectMemberParameters) *v1alpha1.ProjectMember {
	return &v1alpha1.ProjectMember{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       v1alpha1.ProjectMemberSpec{ForProvider: p},
	}
}

func TestProjectMemberValidation(t *testing.T) {
	kube := apiServer(t)

	t.Run("projectName is immutable", func(t *testing.T) {
		pm := projectMember("immutable-project", v1alpha1.ProjectMemberParameters{ProjectName: prodSlug, MembershipID: aliceMembershipID, Role: viewer})
		if err := kube.Create(t.Context(), pm); err != nil {
			t.Fatal(err)
		}
		pm.Spec.ForProvider.ProjectName = "staging"
		err := kube.Update(t.Context(), pm)
		if err == nil || !strings.Contains(err.Error(), "projectName is immutable") {
			t.Errorf("update error = %v, want projectName is immutable", err)
		}
	})

	t.Run("membershipId is immutable", func(t *testing.T) {
		pm := projectMember("immutable-membership", v1alpha1.ProjectMemberParameters{ProjectName: prodSlug, MembershipID: aliceMembershipID, Role: viewer})
		if err := kube.Create(t.Context(), pm); err != nil {
			t.Fatal(err)
		}
		pm.Spec.ForProvider.MembershipID = "43"
		err := kube.Update(t.Context(), pm)
		if err == nil || !strings.Contains(err.Error(), "membershipId is immutable") {
			t.Errorf("update error = %v, want membershipId is immutable", err)
		}
	})

	t.Run("references may fill projectName and membershipId later", func(t *testing.T) {
		pm := projectMember("by-reference", v1alpha1.ProjectMemberParameters{
			ProjectNameRef:  &xpv2.NamespacedReference{Name: "prod"},
			MembershipIDRef: &xpv2.NamespacedReference{Name: "alice"},
			Role:            viewer,
		})
		if err := kube.Create(t.Context(), pm); err != nil {
			t.Fatal(err)
		}
		pm.Spec.ForProvider.ProjectName = prodSlug
		pm.Spec.ForProvider.MembershipID = aliceMembershipID
		if err := kube.Update(t.Context(), pm); err != nil {
			t.Errorf("update error = %v, want none", err)
		}
	})

	t.Run("selectors satisfy the projectName and membershipId rules", func(t *testing.T) {
		pm := projectMember("by-selector", v1alpha1.ProjectMemberParameters{
			ProjectNameSelector:  &xpv2.NamespacedSelector{MatchLabels: map[string]string{"app": "prod"}},
			MembershipIDSelector: &xpv2.NamespacedSelector{MatchLabels: map[string]string{"user": "alice"}},
			Role:                 viewer,
		})
		if err := kube.Create(t.Context(), pm); err != nil {
			t.Errorf("create error = %v, want none", err)
		}
	})

	t.Run("projectName or a reference to it is required", func(t *testing.T) {
		err := kube.Create(t.Context(), projectMember("no-project", v1alpha1.ProjectMemberParameters{MembershipID: aliceMembershipID, Role: viewer}))
		if err == nil || !strings.Contains(err.Error(), ": projectName, projectNameRef or projectNameSelector is required") {
			t.Errorf("create error = %v, want projectName, projectNameRef or projectNameSelector is required", err)
		}
	})

	t.Run("membershipId or a reference to it is required", func(t *testing.T) {
		err := kube.Create(t.Context(), projectMember("no-membership", v1alpha1.ProjectMemberParameters{ProjectName: prodSlug, Role: viewer}))
		if err == nil || !strings.Contains(err.Error(), ": membershipId, membershipIdRef or membershipIdSelector is required") {
			t.Errorf("create error = %v, want membershipId, membershipIdRef or membershipIdSelector is required", err)
		}
	})

	t.Run("role must be a project role", func(t *testing.T) {
		err := kube.Create(t.Context(), projectMember("bad-role", v1alpha1.ProjectMemberParameters{ProjectName: prodSlug, MembershipID: aliceMembershipID, Role: "owner"}))
		if err == nil || !strings.Contains(err.Error(), `Unsupported value: "owner"`) {
			t.Errorf("create error = %v, want unsupported role", err)
		}
	})

	t.Run("each project role is accepted", func(t *testing.T) {
		for _, role := range []string{"project_admin", "contributor", "viewer"} {
			if err := kube.Create(t.Context(), projectMember("role-"+strings.ReplaceAll(role, "_", "-"), v1alpha1.ProjectMemberParameters{ProjectName: prodSlug, MembershipID: aliceMembershipID, Role: role})); err != nil {
				t.Errorf("create with role %q error = %v, want none", role, err)
			}
		}
	})
}
