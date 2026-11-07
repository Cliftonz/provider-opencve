// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "opencve.m.clifton.quest"
	Version = "v1beta1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder()
)

var (
	ProviderConfigKind             = reflect.TypeOf(ProviderConfig{}).Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigKind)
)

var (
	ClusterProviderConfigKind             = reflect.TypeOf(ClusterProviderConfig{}).Name()
	ClusterProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterProviderConfigKind}.String()
	ClusterProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigKind)
)

var (
	ProviderConfigUsageGroupVersionKind     = SchemeGroupVersion.WithKind(reflect.TypeOf(ProviderConfigUsage{}).Name())
	ProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(reflect.TypeOf(ProviderConfigUsageList{}).Name())
)

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion,
			&ProviderConfig{}, &ProviderConfigList{},
			&ClusterProviderConfig{}, &ClusterProviderConfigList{},
			&ProviderConfigUsage{}, &ProviderConfigUsageList{},
		)
		metav1.AddToGroupVersion(s, SchemeGroupVersion)
		return nil
	})
}
