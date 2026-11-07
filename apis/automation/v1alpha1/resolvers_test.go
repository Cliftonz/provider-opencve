// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/Cliftonz/provider-opencve/apis/automation/v1alpha1"
	projectv1alpha1 "github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
)

const projectName = "prod"

func TestResolveReferencesSetsProjectNameFromProjectExternalName(t *testing.T) {
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{v1alpha1.SchemeBuilder.AddToScheme, projectv1alpha1.SchemeBuilder.AddToScheme} {
		if err := add(s); err != nil {
			t.Fatal(err)
		}
	}
	p := &projectv1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "production", Namespace: "team-a"}}
	meta.SetExternalName(p, projectName)
	kube := kfake.NewClientBuilder().WithScheme(s).WithObjects(p).Build()

	a := &v1alpha1.Automation{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "team-a"},
		Spec: v1alpha1.AutomationSpec{ForProvider: v1alpha1.AutomationParameters{
			Name:           "nightly",
			ProjectNameRef: &xpv2.NamespacedReference{Name: "production"},
		}},
	}
	if err := a.ResolveReferences(t.Context(), kube); err != nil {
		t.Fatal(err)
	}
	if a.Spec.ForProvider.ProjectName != projectName {
		t.Errorf("projectName = %q, want %q", a.Spec.ForProvider.ProjectName, projectName)
	}
}
