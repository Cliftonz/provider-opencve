// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:validation:XValidation:rule="has(self.projectName) || has(self.projectNameRef) || has(self.projectNameSelector)",message="projectName, projectNameRef or projectNameSelector is required"
// +kubebuilder:validation:XValidation:rule="has(self.membershipId) || has(self.membershipIdRef) || has(self.membershipIdSelector)",message="membershipId, membershipIdRef or membershipIdSelector is required"
type ProjectMemberParameters struct {
	// ProjectName is the slug of the project the membership belongs to. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/project/v1alpha1.Project
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectName is immutable"
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// +optional
	ProjectNameRef *xpv2.NamespacedReference `json:"projectNameRef,omitempty"`

	// +optional
	ProjectNameSelector *xpv2.NamespacedSelector `json:"projectNameSelector,omitempty"`

	// MembershipID is the numeric id of the organization membership, as a string. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1.OrganizationMember
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="membershipId is immutable"
	// +optional
	MembershipID string `json:"membershipId,omitempty"`

	// +optional
	MembershipIDRef *xpv2.NamespacedReference `json:"membershipIdRef,omitempty"`

	// +optional
	MembershipIDSelector *xpv2.NamespacedSelector `json:"membershipIdSelector,omitempty"`

	// +kubebuilder:validation:Enum=project_admin;contributor;viewer
	Role string `json:"role"`
}

type ProjectMemberObservation struct {
	ID           int    `json:"id,omitempty"`
	MembershipID int    `json:"membershipId,omitempty"`
	Role         string `json:"role,omitempty"`
	Email        string `json:"email,omitempty"`
	Username     string `json:"username,omitempty"`
}

type ProjectMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectMemberParameters `json:"forProvider"`
}

type ProjectMemberStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 ProjectMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ProjectMember grants an organization member a role in an OpenCVE project.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type ProjectMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectMemberSpec   `json:"spec"`
	Status ProjectMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProjectMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectMember `json:"items"`
}
