// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:validation:XValidation:rule="has(self.projectName) || has(self.projectNameRef) || has(self.projectNameSelector)",message="projectName, projectNameRef or projectNameSelector is required"
type NotificationParameters struct {
	// ProjectName is the name of the project owning the notification. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/project/v1alpha1.Project
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectName is immutable"
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// +optional
	ProjectNameRef *xpv2.NamespacedReference `json:"projectNameRef,omitempty"`

	// +optional
	ProjectNameSelector *xpv2.NamespacedSelector `json:"projectNameSelector,omitempty"`

	// Name is the notification name and its external name. It cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Type is the channel the notification is delivered through, such as email
	// or webhook. It cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	Type string `json:"type"`

	// IsEnabled is late-initialized from the remote notification when omitted.
	// +optional
	IsEnabled *bool `json:"isEnabled,omitempty"`

	// ConfigurationSecretRef selects a key, in a Secret in the notification's
	// namespace, holding the notification configuration as JSON. The
	// configuration is applied when the notification is created and whenever
	// the Secret's content changes. Keys removed from the Secret are not removed
	// from OpenCVE. The reference cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="configurationSecretRef is immutable"
	// +optional
	ConfigurationSecretRef *xpv2.LocalSecretKeySelector `json:"configurationSecretRef,omitempty"`
}

type NotificationObservation struct {
	ID                         string `json:"id,omitempty"`
	Name                       string `json:"name,omitempty"`
	Type                       string `json:"type,omitempty"`
	IsEnabled                  *bool  `json:"isEnabled,omitempty"`
	IsPendingEmailConfirmation *bool  `json:"isPendingEmailConfirmation,omitempty"`
	// ConfigurationHash is the SHA-256 of the configuration last applied from
	// the Secret, used to detect changes to its content.
	ConfigurationHash string `json:"configurationHash,omitempty"`
}

type NotificationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              NotificationParameters `json:"forProvider"`
}

type NotificationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 NotificationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Notification is an OpenCVE project notification.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type Notification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationSpec   `json:"spec"`
	Status NotificationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notification `json:"items"`
}
