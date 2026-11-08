// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

type OrganizationParameters struct{}

type OrganizationObservation struct {
	ID            string       `json:"id,omitempty"`
	Name          string       `json:"name,omitempty"`
	MembersCount  *int         `json:"membersCount,omitempty"`
	ProjectsCount *int         `json:"projectsCount,omitempty"`
	CreatedAt     *metav1.Time `json:"createdAt,omitempty"`
	UpdatedAt     *metav1.Time `json:"updatedAt,omitempty"`
}

type OrganizationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationParameters `json:"forProvider,omitempty"`
}

type OrganizationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 OrganizationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Organization observes the OpenCVE organization named in the ProviderConfig credentials.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}
