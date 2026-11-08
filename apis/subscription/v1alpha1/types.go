// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// Create one ProjectSubscriptions per project; it owns that project's whole set
// of vendor and product subscriptions.
// +kubebuilder:validation:XValidation:rule="has(self.projectName) || has(self.projectNameRef) || has(self.projectNameSelector)",message="projectName, projectNameRef or projectNameSelector is required"
type ProjectSubscriptionsParameters struct {
	// ProjectName is the slug of the project whose subscriptions this resource owns. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/project/v1alpha1.Project
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectName is immutable"
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// +optional
	ProjectNameRef *xpv2.NamespacedReference `json:"projectNameRef,omitempty"`

	// +optional
	ProjectNameSelector *xpv2.NamespacedSelector `json:"projectNameSelector,omitempty"`

	// Vendors is the full set of subscribed vendors. Omitting it unsubscribes every vendor.
	// +optional
	Vendors []string `json:"vendors,omitempty"`

	// Products maps each vendor to its full set of subscribed products. Omitting it unsubscribes every product.
	// +optional
	Products map[string][]string `json:"products,omitempty"`
}

type ProjectSubscriptionsObservation struct {
	Vendors  []string            `json:"vendors,omitempty"`
	Products map[string][]string `json:"products,omitempty"`
}

type ProjectSubscriptionsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectSubscriptionsParameters `json:"forProvider"`
}

type ProjectSubscriptionsStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 ProjectSubscriptionsObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectSubscriptions is the authoritative set of vendor and product subscriptions of one OpenCVE project.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type ProjectSubscriptions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSubscriptionsSpec   `json:"spec"`
	Status ProjectSubscriptionsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProjectSubscriptionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectSubscriptions `json:"items"`
}
