// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build envtest

package v1alpha1_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/Cliftonz/provider-opencve/apis/automation/v1alpha1"
)

func newKube(t *testing.T) client.Client {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "package", "crds", "automation.opencve.m.clifton.quest_automations.yaml")},
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

func automation(name string, fp v1alpha1.AutomationParameters) *v1alpha1.Automation {
	return &v1alpha1.Automation{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       v1alpha1.AutomationSpec{ForProvider: fp},
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCRD(t *testing.T) {
	kube := newKube(t)

	t.Run("ProjectNameAndNameAreImmutable", func(t *testing.T) {
		for field, mutate := range map[string]func(*v1alpha1.AutomationParameters){
			"projectName is immutable": func(fp *v1alpha1.AutomationParameters) { fp.ProjectName = "staging" },
			"name is immutable":        func(fp *v1alpha1.AutomationParameters) { fp.Name = "weekly" },
		} {
			a := automation(strings.ToLower(strings.ReplaceAll(field, " ", "-")), v1alpha1.AutomationParameters{ProjectName: projectName, Name: "nightly"})
			if err := kube.Create(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			mutate(&a.Spec.ForProvider)
			err := kube.Update(t.Context(), a)
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Errorf("update error = %v, want %s", err, field)
			}
		}
	})

	t.Run("ProjectIsRequired", func(t *testing.T) {
		err := kube.Create(t.Context(), automation("no-project", v1alpha1.AutomationParameters{Name: "a"}))
		if err == nil || !strings.Contains(err.Error(), ": projectName, projectNameRef or projectNameSelector is required") {
			t.Errorf("create error = %v, want project to be required", err)
		}
	})

	t.Run("ProjectNameRefAloneIsAccepted", func(t *testing.T) {
		fp := v1alpha1.AutomationParameters{Name: "a", ProjectNameRef: &xpv2.NamespacedReference{Name: "production"}}
		if err := kube.Create(t.Context(), automation("project-ref-only", fp)); err != nil {
			t.Error(err)
		}
	})

	t.Run("EnumsRejectUnknownValues", func(t *testing.T) {
		for field, fp := range map[string]v1alpha1.AutomationParameters{
			"triggerType":     {ProjectName: projectName, Name: "a", TriggerType: ptr("webhook")},
			"frequency":       {ProjectName: projectName, Name: "a", Frequency: ptr("monthly")},
			"scheduleWeekday": {ProjectName: projectName, Name: "a", ScheduleWeekday: ptr("someday")},
		} {
			err := kube.Create(t.Context(), automation("enum-"+strings.ToLower(field), fp))
			if err == nil || !strings.Contains(err.Error(), "spec.forProvider."+field) || !strings.Contains(err.Error(), "Unsupported value") {
				t.Errorf("%s: create error = %v, want unsupported value", field, err)
			}
		}
	})

	t.Run("EnumsAcceptSpecValues", func(t *testing.T) {
		a := automation("enum-valid", v1alpha1.AutomationParameters{ProjectName: projectName, Name: "a", TriggerType: ptr("report"), Frequency: ptr("weekly"), ScheduleWeekday: ptr("sunday")})
		if err := kube.Create(t.Context(), a); err != nil {
			t.Error(err)
		}
	})

	t.Run("ScheduleTimeMustBeHHMMSS", func(t *testing.T) {
		for i, v := range []string{"9:00:00", "09:00", "09:00:00Z"} {
			err := kube.Create(t.Context(), automation(fmt.Sprintf("schedule-time-rejected-%d", i), v1alpha1.AutomationParameters{ProjectName: projectName, Name: "a", ScheduleTime: ptr(v)}))
			if err == nil || !strings.Contains(err.Error(), "scheduleTime must match HH:MM:SS") {
				t.Errorf("%q: create error = %v, want scheduleTime must match HH:MM:SS", v, err)
			}
		}
		if err := kube.Create(t.Context(), automation("schedule-time-accepted", v1alpha1.AutomationParameters{ProjectName: projectName, Name: "a", ScheduleTime: ptr("09:00:00")})); err != nil {
			t.Error(err)
		}
	})

	t.Run("ProjectNameReferenceAndSelectorRoundTrip", func(t *testing.T) {
		fp := v1alpha1.AutomationParameters{
			Name:                "a",
			ProjectNameRef:      &xpv2.NamespacedReference{Name: "production", Namespace: "team-a"},
			ProjectNameSelector: &xpv2.NamespacedSelector{MatchLabels: map[string]string{"env": projectName}, Namespace: "team-a"},
		}
		if err := kube.Create(t.Context(), automation("reference", fp)); err != nil {
			t.Fatal(err)
		}
		got := &v1alpha1.Automation{}
		if err := kube.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "reference"}, got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Spec.ForProvider.ProjectNameRef, fp.ProjectNameRef) || !reflect.DeepEqual(got.Spec.ForProvider.ProjectNameSelector, fp.ProjectNameSelector) {
			t.Errorf("reference = %+v, selector = %+v, want %+v and %+v", got.Spec.ForProvider.ProjectNameRef, got.Spec.ForProvider.ProjectNameSelector, fp.ProjectNameRef, fp.ProjectNameSelector)
		}
	})

	t.Run("ConfigurationPreservesUnknownFields", func(t *testing.T) {
		raw := `{"actions":[{"type":"email","value":"a@example.com"}],"conditions":{"children":[{"type":"cvss31_score","value":7}],"operator":"OR"}}`
		fp := v1alpha1.AutomationParameters{ProjectName: projectName, Name: "a", Configuration: &runtime.RawExtension{Raw: []byte(raw)}}
		if err := kube.Create(t.Context(), automation("configuration", fp)); err != nil {
			t.Fatal(err)
		}
		got := &v1alpha1.Automation{}
		if err := kube.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "configuration"}, got); err != nil {
			t.Fatal(err)
		}
		if string(got.Spec.ForProvider.Configuration.Raw) != raw {
			t.Errorf("configuration = %s, want %s", got.Spec.ForProvider.Configuration.Raw, raw)
		}
	})
}
