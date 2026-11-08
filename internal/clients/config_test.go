// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
)

const namespace = "team-a"

func credentialsSpec(secretNamespace string) v1beta1.ProviderConfigSpec {
	return v1beta1.ProviderConfigSpec{Credentials: v1beta1.ProviderCredentials{
		Source: xpv2.CredentialsSourceSecret,
		CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: &xpv2.SecretKeySelector{
			SecretReference: xpv2.SecretReference{Name: "opencve-creds", Namespace: secretNamespace},
			Key:             "credentials",
		}},
	}}
}

func kubeClient(t *testing.T, pc client.Object, secretNamespace, credentials string) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := v1beta1.SchemeBuilder.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "opencve-creds", Namespace: secretNamespace},
		Data:       map[string][]byte{"credentials": []byte(credentials)},
	}
	return kfake.NewClientBuilder().WithScheme(s).WithObjects(pc, secret).Build()
}

func managed(kind string) *fake.ModernManaged {
	mg := &fake.ModernManaged{ObjectMeta: metav1.ObjectMeta{Name: "cve-project", Namespace: namespace, UID: "mg-uid"}}
	mg.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: kind, Name: "default"})
	return mg
}

func namespacedProviderConfig() *v1beta1.ProviderConfig {
	return &v1beta1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: namespace},
		Spec:       credentialsSpec(""),
	}
}

func TestNewClientCredentialErrors(t *testing.T) {
	cases := map[string]struct {
		credentials string
		want        error
	}{
		"MalformedJSON": {
			credentials: `{"token":`,
			want:        ErrMalformedCredentials,
		},
		"MissingToken": {
			credentials: `{"organization":"acme","url":"https://app.opencve.io"}`,
			want:        ErrMissingToken,
		},
		"MissingOrganization": {
			credentials: `{"token":"t","url":"https://app.opencve.io"}`,
			want:        ErrMissingOrganization,
		},
		"MissingURL": {
			credentials: `{"token":"t","organization":"acme"}`,
			want:        ErrMissingURL,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			kube := kubeClient(t, namespacedProviderConfig(), namespace, tc.credentials)
			_, _, err := NewClient(context.Background(), kube, managed(v1beta1.ProviderConfigKind))
			if !errors.Is(err, tc.want) {
				t.Fatalf("NewClient() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewClientUnknownProviderConfigKind(t *testing.T) {
	for _, kind := range []string{"UnregisteredKind", "ProviderConfigUsage"} {
		t.Run(kind, func(t *testing.T) {
			kube := kubeClient(t, namespacedProviderConfig(), namespace, `{"token":"t","organization":"acme","url":"https://app.opencve.io"}`)
			_, _, err := NewClient(context.Background(), kube, managed(kind))
			if !errors.Is(err, ErrUnknownProviderConfigKind) {
				t.Fatalf("NewClient() error = %v, want %v", err, ErrUnknownProviderConfigKind)
			}
		})
	}
}

func TestNewClientSuccess(t *testing.T) {
	cases := map[string]struct {
		pc              client.Object
		kind            string
		secretNamespace string
	}{
		"ProviderConfig": {
			pc:              namespacedProviderConfig(),
			kind:            v1beta1.ProviderConfigKind,
			secretNamespace: namespace,
		},
		"ClusterProviderConfig": {
			pc: &v1beta1.ClusterProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec:       credentialsSpec("crossplane-system"),
			},
			kind:            v1beta1.ClusterProviderConfigKind,
			secretNamespace: "crossplane-system",
		},
		"ProviderConfigSecretRefNamespaceReplacedByManagedNamespace": {
			pc: &v1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: namespace},
				Spec:       credentialsSpec("elsewhere"),
			},
			kind:            v1beta1.ProviderConfigKind,
			secretNamespace: namespace,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var gotPath, gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			credentials := `{"token":"secret-token","organization":"acme","url":"` + srv.URL + `"}`
			kube := kubeClient(t, tc.pc, tc.secretNamespace, credentials)
			oc, org, err := NewClient(context.Background(), kube, managed(tc.kind))
			if err != nil {
				t.Fatalf("NewClient() unexpected error: %v", err)
			}
			if org != "acme" {
				t.Errorf("organization = %q, want %q", org, "acme")
			}
			if _, err := oc.OrganizationsRetrieveWithResponse(context.Background(), "acme"); err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if gotPath != "/api/v2/organizations/acme" {
				t.Errorf("path = %q, want %q", gotPath, "/api/v2/organizations/acme")
			}
			if gotAuth != "Bearer secret-token" {
				t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secret-token")
			}
		})
	}
}
