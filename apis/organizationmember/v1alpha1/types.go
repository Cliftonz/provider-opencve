// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

type OrganizationMemberParameters struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="email is immutable"
	Email string `json:"email"`
	// +kubebuilder:validation:Enum=owner;admin;member
	Role string `json:"role"`
}

type OrganizationMemberObservation struct {
	ID          int          `json:"id,omitempty"`
	Email       string       `json:"email,omitempty"`
	Role        string       `json:"role,omitempty"`
	DateInvited *metav1.Time `json:"dateInvited,omitempty"`
	DateJoined  *metav1.Time `json:"dateJoined,omitempty"`
	IsInvited   bool         `json:"isInvited,omitempty"`
}

type OrganizationMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationMemberParameters `json:"forProvider"`
}

type OrganizationMemberStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 OrganizationMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An OrganizationMember is a membership of a user in an OpenCVE organization.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type OrganizationMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationMemberSpec   `json:"spec"`
	Status OrganizationMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type OrganizationMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationMember `json:"items"`
}
