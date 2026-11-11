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

	"github.com/Cliftonz/provider-opencve/apis/notification/v1alpha1"
)

func newKube(t *testing.T) client.Client {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "package", "crds", "notification.opencve.m.clifton.quest_notifications.yaml")},
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

func notification(name string, fp v1alpha1.NotificationParameters) *v1alpha1.Notification {
	return &v1alpha1.Notification{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       v1alpha1.NotificationSpec{ForProvider: fp},
	}
}

func secretRef(name string) *xpv2.LocalSecretKeySelector {
	return &xpv2.LocalSecretKeySelector{LocalSecretReference: xpv2.LocalSecretReference{Name: name}, Key: "configuration"}
}

func TestCRD(t *testing.T) {
	kube := newKube(t)

	t.Run("ProjectNameNameTypeAndConfigurationSecretRefAreImmutable", func(t *testing.T) {
		for field, mutate := range map[string]func(*v1alpha1.NotificationParameters){
			"projectName is immutable":            func(fp *v1alpha1.NotificationParameters) { fp.ProjectName = "staging" },
			"name is immutable":                   func(fp *v1alpha1.NotificationParameters) { fp.Name = "dev" },
			"type is immutable":                   func(fp *v1alpha1.NotificationParameters) { fp.Type = "slack" },
			"configurationSecretRef is immutable": func(fp *v1alpha1.NotificationParameters) { fp.ConfigurationSecretRef = secretRef("other") },
		} {
			n := notification(strings.ToLower(strings.ReplaceAll(field, " ", "-")), v1alpha1.NotificationParameters{ProjectName: "prod", Name: notificationName, Type: webhookType, ConfigurationSecretRef: secretRef(notificationName)})
			if err := kube.Create(t.Context(), n); err != nil {
				t.Fatal(err)
			}
			mutate(&n.Spec.ForProvider)
			err := kube.Update(t.Context(), n)
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Errorf("update error = %v, want %s", err, field)
			}
		}
	})

	t.Run("ConfigurationSecretRefCannotBeAddedOrRemoved", func(t *testing.T) {
		for name, tc := range map[string]struct {
			before, after *xpv2.LocalSecretKeySelector
		}{
			"add-ref":    {after: secretRef(notificationName)},
			"remove-ref": {before: secretRef(notificationName)},
		} {
			n := notification(name, v1alpha1.NotificationParameters{ProjectName: "prod", Name: notificationName, Type: webhookType, ConfigurationSecretRef: tc.before})
			if err := kube.Create(t.Context(), n); err != nil {
				t.Fatal(err)
			}
			n.Spec.ForProvider.ConfigurationSecretRef = tc.after
			err := kube.Update(t.Context(), n)
			if err == nil || !strings.Contains(err.Error(), "configurationSecretRef cannot be added or removed after creation") {
				t.Errorf("%s: update error = %v, want configurationSecretRef cannot be added or removed after creation", name, err)
			}
		}
	})

	t.Run("UnchangedConfigurationSecretRefIsAccepted", func(t *testing.T) {
		n := notification("keep-ref", v1alpha1.NotificationParameters{ProjectName: "prod", Name: notificationName, Type: webhookType, ConfigurationSecretRef: secretRef(notificationName)})
		if err := kube.Create(t.Context(), n); err != nil {
			t.Fatal(err)
		}
		enabled := false
		n.Spec.ForProvider.IsEnabled = &enabled
		if err := kube.Update(t.Context(), n); err != nil {
			t.Errorf("update error = %v, want the update accepted", err)
		}
	})

	t.Run("ProjectIsRequired", func(t *testing.T) {
		err := kube.Create(t.Context(), notification("no-project", v1alpha1.NotificationParameters{Name: notificationName, Type: webhookType}))
		if err == nil || !strings.Contains(err.Error(), ": projectName, projectNameRef or projectNameSelector is required") {
			t.Errorf("create error = %v, want project to be required", err)
		}
	})

	t.Run("ProjectNameRefOrSelectorSatisfiesProject", func(t *testing.T) {
		for name, fp := range map[string]v1alpha1.NotificationParameters{
			"by-ref":      {Name: notificationName, Type: webhookType, ProjectNameRef: &xpv2.NamespacedReference{Name: "production"}},
			"by-selector": {Name: notificationName, Type: webhookType, ProjectNameSelector: &xpv2.NamespacedSelector{MatchLabels: map[string]string{"env": "prod"}}},
		} {
			if err := kube.Create(t.Context(), notification(name, fp)); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
	})
}
