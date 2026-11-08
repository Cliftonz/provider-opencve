// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package apis_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Cliftonz/provider-opencve/apis"
)

func TestAddToSchemeRegistersConfigKinds(t *testing.T) {
	s := runtime.NewScheme()
	if err := apis.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"ProviderConfig", "ClusterProviderConfig", "ProviderConfigUsage"} {
		gvk := schema.GroupVersionKind{Group: "opencve.m.clifton.quest", Version: "v1beta1", Kind: kind}
		if !s.Recognizes(gvk) {
			t.Errorf("scheme does not recognize %s", gvk)
		}
	}
}

func TestAddToSchemeRegistersManagedKinds(t *testing.T) {
	s := runtime.NewScheme()
	if err := apis.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	for group, kind := range map[string]string{
		"project":            "Project",
		"organization":       "Organization",
		"organizationmember": "OrganizationMember",
		"automation":         "Automation",
		"notification":       "Notification",
		"subscription":       "ProjectSubscriptions",
		"cvetracker":         "CveTracker",
		"projectmember":      "ProjectMember",
	} {
		gvk := schema.GroupVersionKind{Group: group + ".opencve.m.clifton.quest", Version: "v1alpha1", Kind: kind}
		if !s.Recognizes(gvk) {
			t.Errorf("scheme does not recognize %s", gvk)
		}
	}
}
