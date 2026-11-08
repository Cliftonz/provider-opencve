// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1beta1_test

import (
	"context"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
)

func TestSecretRefIsRequired(t *testing.T) {
	env := &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../package/crds/opencve.m.clifton.quest_providerconfigs.yaml",
			"../../package/crds/opencve.m.clifton.quest_clusterproviderconfigs.yaml",
		},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })

	s := runtime.NewScheme()
	if err := v1beta1.SchemeBuilder.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	kube, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		t.Fatal(err)
	}

	spec := v1beta1.ProviderConfigSpec{Credentials: v1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceSecret}}
	cases := map[string]client.Object{
		"ProviderConfig":        &v1beta1.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"}, Spec: spec},
		"ClusterProviderConfig": &v1beta1.ClusterProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "default"}, Spec: spec},
	}
	for name, obj := range cases {
		t.Run(name, func(t *testing.T) {
			err := kube.Create(context.Background(), obj)
			if err == nil || !strings.Contains(err.Error(), "secretRef is required") {
				t.Fatalf("Create() error = %v, want secretRef is required", err)
			}
		})
	}
}
