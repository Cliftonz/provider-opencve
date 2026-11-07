// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1alpha1_test

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
)

func TestEmailIsImmutable(t *testing.T) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../package/crds/organizationmember.opencve.m.clifton.quest_organizationmembers.yaml"},
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

	ctx := context.Background()
	cr := &v1alpha1.OrganizationMember{
		ObjectMeta: metav1.ObjectMeta{Name: "alice", Namespace: "default"},
		Spec:       v1alpha1.OrganizationMemberSpec{ForProvider: v1alpha1.OrganizationMemberParameters{Email: "alice@example.com", Role: "member"}},
	}
	if err := kube.Create(ctx, cr); err != nil {
		t.Fatal(err)
	}

	cr.Spec.ForProvider.Email = "bob@example.com"
	err = kube.Update(ctx, cr)
	if err == nil || !strings.Contains(err.Error(), "email is immutable") {
		t.Fatalf("Update() error = %v, want email is immutable", err)
	}
}

func TestRoleMustBeKnown(t *testing.T) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../package/crds/organizationmember.opencve.m.clifton.quest_organizationmembers.yaml"},
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

	cr := &v1alpha1.OrganizationMember{
		ObjectMeta: metav1.ObjectMeta{Name: "alice", Namespace: "default"},
		Spec:       v1alpha1.OrganizationMemberSpec{ForProvider: v1alpha1.OrganizationMemberParameters{Email: "alice@example.com", Role: "boss"}},
	}
	err = kube.Create(context.Background(), cr)
	if err == nil || !strings.Contains(err.Error(), "spec.forProvider.role: Unsupported value") {
		t.Fatalf("Create() error = %v, want spec.forProvider.role: Unsupported value", err)
	}
}
