// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

type ProjectParameters struct {
	// Name is the project slug and its external name. It cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the project. Omitting it clears any remote description.
	// +optional
	Description *string `json:"description,omitempty"`

	// Active is late-initialized from the remote project when omitted.
	// +optional
	Active *bool `json:"active,omitempty"`
}

type ProjectObservation struct {
	ID                 string  `json:"id,omitempty"`
	Name               string  `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	Active             *bool   `json:"active,omitempty"`
	SubscriptionsCount *int    `json:"subscriptionsCount,omitempty"`
}

type ProjectSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectParameters `json:"forProvider"`
}

type ProjectStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 ProjectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Project is an OpenCVE project within the provider's organization.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
