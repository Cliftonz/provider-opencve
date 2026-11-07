// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1alpha1_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/Cliftonz/provider-opencve/apis/subscription/v1alpha1"
)

var kube client.Client

func TestMain(m *testing.M) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "package", "crds", "subscription.opencve.m.clifton.quest_projectsubscriptions.yaml")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		panic(err)
	}
	s := runtime.NewScheme()
	if err := v1alpha1.SchemeBuilder.AddToScheme(s); err != nil {
		panic(err)
	}
	kube, err = client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		panic(err)
	}
	code := m.Run()
	_ = env.Stop()
	os.Exit(code)
}

func subscriptions(name string, fp v1alpha1.ProjectSubscriptionsParameters) *v1alpha1.ProjectSubscriptions {
	return &v1alpha1.ProjectSubscriptions{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       v1alpha1.ProjectSubscriptionsSpec{ForProvider: fp},
	}
}

func TestProjectNameIsImmutable(t *testing.T) {
	ps := subscriptions("immutable", v1alpha1.ProjectSubscriptionsParameters{ProjectName: "prod"})
	if err := kube.Create(context.Background(), ps); err != nil {
		t.Fatal(err)
	}
	ps.Spec.ForProvider.ProjectName = "staging"
	err := kube.Update(context.Background(), ps)
	if err == nil || !strings.Contains(err.Error(), "projectName is immutable") {
		t.Errorf("update error = %v, want projectName is immutable", err)
	}
}

func TestProjectNameSourceIsRequired(t *testing.T) {
	err := kube.Create(context.Background(), subscriptions("no-source", v1alpha1.ProjectSubscriptionsParameters{Vendors: []string{"apple"}}))
	if err == nil || !strings.Contains(err.Error(), ": projectName, projectNameRef or projectNameSelector is required") {
		t.Errorf("create error = %v, want projectName, projectNameRef or projectNameSelector is required", err)
	}
}

func TestProjectNameRefAloneIsAccepted(t *testing.T) {
	ps := subscriptions("ref-only", v1alpha1.ProjectSubscriptionsParameters{ProjectNameRef: &xpv2.NamespacedReference{Name: "prod"}})
	if err := kube.Create(context.Background(), ps); err != nil {
		t.Errorf("create error = %v, want accepted", err)
	}
}
