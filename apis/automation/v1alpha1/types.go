// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// +kubebuilder:validation:XValidation:rule="has(self.projectName) || has(self.projectNameRef) || has(self.projectNameSelector)",message="projectName, projectNameRef or projectNameSelector is required"
type AutomationParameters struct {
	// ProjectName is the name of the project owning the automation. It cannot be changed.
	// +crossplane:generate:reference:type=github.com/Cliftonz/provider-opencve/apis/project/v1alpha1.Project
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectName is immutable"
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// +optional
	ProjectNameRef *xpv2.NamespacedReference `json:"projectNameRef,omitempty"`

	// +optional
	ProjectNameSelector *xpv2.NamespacedSelector `json:"projectNameSelector,omitempty"`

	// Name is the automation name and its external name. It cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// IsEnabled is late-initialized from the remote automation when omitted.
	// +optional
	IsEnabled *bool `json:"isEnabled,omitempty"`

	// TriggerType is late-initialized from the remote automation when omitted.
	// +kubebuilder:validation:Enum=alert;report
	// +optional
	TriggerType *string `json:"triggerType,omitempty"`

	// Frequency of a report. Omitting it clears the remote frequency.
	// +kubebuilder:validation:Enum=daily;weekly
	// +optional
	Frequency *string `json:"frequency,omitempty"`

	// ScheduleTimezone of a report. Omitting it clears the remote timezone.
	// +optional
	ScheduleTimezone *string `json:"scheduleTimezone,omitempty"`

	// ScheduleTime of a report, as HH:MM:SS. Omitting it clears the remote time.
	// +kubebuilder:validation:XValidation:rule="self.matches('^[0-9]{2}:[0-9]{2}:[0-9]{2}$')",message="scheduleTime must match HH:MM:SS"
	// +optional
	ScheduleTime *string `json:"scheduleTime,omitempty"`

	// ScheduleWeekday of a weekly report. Omitting it clears the remote weekday.
	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday
	// +optional
	ScheduleWeekday *string `json:"scheduleWeekday,omitempty"`

	// Configuration is the automation's free-form conditions and actions. It is
	// late-initialized from the remote automation when omitted.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Configuration *runtime.RawExtension `json:"configuration,omitempty"`
}

type AutomationObservation struct {
	ID               string  `json:"id,omitempty"`
	Name             string  `json:"name,omitempty"`
	IsEnabled        *bool   `json:"isEnabled,omitempty"`
	TriggerType      *string `json:"triggerType,omitempty"`
	Frequency        *string `json:"frequency,omitempty"`
	ScheduleTimezone *string `json:"scheduleTimezone,omitempty"`
	ScheduleTime     *string `json:"scheduleTime,omitempty"`
	ScheduleWeekday  *string `json:"scheduleWeekday,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Configuration   *runtime.RawExtension `json:"configuration,omitempty"`
	ConditionsCount *int                  `json:"conditionsCount,omitempty"`
	LastExecutionAt *metav1.Time          `json:"lastExecutionAt,omitempty"`
}

type AutomationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              AutomationParameters `json:"forProvider"`
}

type AutomationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 AutomationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Automation is an OpenCVE project automation.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opencve}
type Automation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutomationSpec   `json:"spec"`
	Status AutomationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type AutomationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Automation `json:"items"`
}
