// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:validation:XValidation:rule="has(self.projectName) || has(self.projectNameRef) || has(self.projectNameSelector)",message="projectName, projectNameRef or projectNameSelector is required"
type CveTrackerParameters struct {
	// ProjectName is the name of the project the CVE belongs to. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/project/v1alpha1.Project
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectName is immutable"
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// +optional
	ProjectNameRef *xpv2.NamespacedReference `json:"projectNameRef,omitempty"`

	// +optional
	ProjectNameSelector *xpv2.NamespacedSelector `json:"projectNameSelector,omitempty"`

	// CveID identifies the CVE within the project. It cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="cveId is immutable"
	// +kubebuilder:validation:XValidation:rule="self.matches('^CVE-[0-9]{4}-[0-9]+$')",message="cveId must match CVE-YYYY-N"
	CveID string `json:"cveId"`

	// Status of the tracker. Omitting it clears the remote status.
	// +kubebuilder:validation:Enum=to_evaluate;pending_review;analysis_in_progress;remediation_in_progress;evaluated;resolved;not_applicable;risk_accepted
	// +optional
	Status *string `json:"status,omitempty"`

	// Assignee is the email of the user assigned to the CVE. Omitting it clears the remote assignee.
	// +optional
	Assignee *string `json:"assignee,omitempty"`
}

type CveTrackerObservation struct {
	CveID    string `json:"cveId,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Assignee string `json:"assignee,omitempty"`
}

type CveTrackerSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              CveTrackerParameters `json:"forProvider"`
}

type CveTrackerStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 CveTrackerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A CveTracker adopts the tracker status and assignee of one CVE within one OpenCVE project.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type CveTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CveTrackerSpec   `json:"spec"`
	Status CveTrackerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type CveTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CveTracker `json:"items"`
}
