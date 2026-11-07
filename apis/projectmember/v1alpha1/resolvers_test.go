// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	organizationmemberv1alpha1 "github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
	projectv1alpha1 "github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	"github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
)

const (
	prodSlug          = "prod"
	aliceMembershipID = "42"
	viewer            = "viewer"
)

func TestResolveReferencesFromExternalNames(t *testing.T) {
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		v1alpha1.SchemeBuilder.AddToScheme,
		projectv1alpha1.SchemeBuilder.AddToScheme,
		organizationmemberv1alpha1.SchemeBuilder.AddToScheme,
	} {
		if err := add(s); err != nil {
			t.Fatal(err)
		}
	}
	p := &projectv1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "production", Namespace: "team-a"}}
	meta.SetExternalName(p, prodSlug)
	om := &organizationmemberv1alpha1.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Name: "alice", Namespace: "team-a"}}
	meta.SetExternalName(om, aliceMembershipID)
	kube := kfake.NewClientBuilder().WithScheme(s).WithObjects(p, om).Build()

	pm := &v1alpha1.ProjectMember{
		ObjectMeta: metav1.ObjectMeta{Name: "alice-prod", Namespace: "team-a"},
		Spec: v1alpha1.ProjectMemberSpec{ForProvider: v1alpha1.ProjectMemberParameters{
			ProjectNameRef:  &xpv2.NamespacedReference{Name: "production"},
			MembershipIDRef: &xpv2.NamespacedReference{Name: "alice"},
			Role:            viewer,
		}},
	}
	if err := pm.ResolveReferences(t.Context(), kube); err != nil {
		t.Fatal(err)
	}
	if pm.Spec.ForProvider.ProjectName != prodSlug {
		t.Errorf("projectName = %q, want %q", pm.Spec.ForProvider.ProjectName, prodSlug)
	}
	if pm.Spec.ForProvider.MembershipID != aliceMembershipID {
		t.Errorf("membershipId = %q, want %q", pm.Spec.ForProvider.MembershipID, aliceMembershipID)
	}
}
