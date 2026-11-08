// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "subscription.opencve.m.clifton.quest"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder()
)

var (
	ProjectSubscriptionsKind             = reflect.TypeOf(ProjectSubscriptions{}).Name()
	ProjectSubscriptionsGroupVersionKind = SchemeGroupVersion.WithKind(ProjectSubscriptionsKind)
)

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ProjectSubscriptions{}, &ProjectSubscriptionsList{})
		metav1.AddToGroupVersion(s, SchemeGroupVersion)
		return nil
	})
}
