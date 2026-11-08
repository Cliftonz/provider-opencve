// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	automationv1alpha1 "github.com/Cliftonz/provider-opencve/apis/automation/v1alpha1"
	cvetrackerv1alpha1 "github.com/Cliftonz/provider-opencve/apis/cvetracker/v1alpha1"
	notificationv1alpha1 "github.com/Cliftonz/provider-opencve/apis/notification/v1alpha1"
	organizationv1alpha1 "github.com/Cliftonz/provider-opencve/apis/organization/v1alpha1"
	organizationmemberv1alpha1 "github.com/Cliftonz/provider-opencve/apis/organizationmember/v1alpha1"
	projectv1alpha1 "github.com/Cliftonz/provider-opencve/apis/project/v1alpha1"
	projectmemberv1alpha1 "github.com/Cliftonz/provider-opencve/apis/projectmember/v1alpha1"
	subscriptionv1alpha1 "github.com/Cliftonz/provider-opencve/apis/subscription/v1alpha1"
	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
)

var schemeBuilder = runtime.SchemeBuilder{
	v1beta1.SchemeBuilder.AddToScheme,
	projectv1alpha1.SchemeBuilder.AddToScheme,
	organizationv1alpha1.SchemeBuilder.AddToScheme,
	organizationmemberv1alpha1.SchemeBuilder.AddToScheme,
	automationv1alpha1.SchemeBuilder.AddToScheme,
	notificationv1alpha1.SchemeBuilder.AddToScheme,
	subscriptionv1alpha1.SchemeBuilder.AddToScheme,
	cvetrackerv1alpha1.SchemeBuilder.AddToScheme,
	projectmemberv1alpha1.SchemeBuilder.AddToScheme,
}

var AddToScheme = schemeBuilder.AddToScheme
