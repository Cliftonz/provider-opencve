// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/notification/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

var errMissingConfigurationKey = errors.New("configuration key is missing")

type external struct {
	kube   client.Client
	client *opencve.ClientWithResponses
	org    string
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.Notification) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}
	resp, err := e.client.OrganizationsProjectsNotificationsRetrieveWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get notification %q", name), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	at := observation(resp.JSON200)
	cr.Status.AtProvider = at
	fp := &cr.Spec.ForProvider
	lateInitialized := fp.IsEnabled == nil && at.IsEnabled != nil
	if lateInitialized {
		fp.IsEnabled = at.IsEnabled
	}
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        reflect.DeepEqual(fp.IsEnabled, at.IsEnabled),
		ResourceLateInitialized: lateInitialized,
	}, nil
}

func observation(remote *opencve.Notification) v1alpha1.NotificationObservation {
	at := v1alpha1.NotificationObservation{
		ID:        remote.Id.String(),
		Name:      remote.Name,
		Type:      remote.Type,
		IsEnabled: remote.IsEnabled,
	}
	if pending, err := remote.IsPendingEmailConfirmation.Get(); err == nil {
		at.IsPendingEmailConfirmation = &pending
	}
	return at
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.Notification) (managed.ExternalCreation, error) {
	fp := cr.Spec.ForProvider
	body := opencve.NotificationWriteRequest{Name: fp.Name, Type: fp.Type, IsEnabled: fp.IsEnabled}
	if ref := fp.ConfigurationSecretRef; ref != nil {
		s := &corev1.Secret{}
		if err := e.kube.Get(ctx, types.NamespacedName{Namespace: cr.GetNamespace(), Name: ref.Name}, s); err != nil {
			return managed.ExternalCreation{}, err
		}
		raw, ok := s.Data[ref.Key]
		if !ok {
			return managed.ExternalCreation{}, fmt.Errorf("%w: key %q in Secret %q", errMissingConfigurationKey, ref.Key, ref.Name)
		}
		if err := json.Unmarshal(raw, &body.Configuration); err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("cannot decode configuration from key %q in Secret %q: %w", ref.Key, ref.Name, err)
		}
	}
	resp, err := e.client.OrganizationsProjectsNotificationsCreateWithResponse(ctx, e.org, fp.ProjectName, body)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("create notification %q", fp.Name), resp); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, fp.Name)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.Notification) (managed.ExternalUpdate, error) {
	fp := cr.Spec.ForProvider
	resp, err := e.client.OrganizationsProjectsNotificationsPartialUpdateWithResponse(ctx, e.org, fp.ProjectName, meta.GetExternalName(cr), opencve.PatchedNotificationWriteRequest{
		IsEnabled: fp.IsEnabled,
	})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update notification %q", meta.GetExternalName(cr)), resp); err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = observation(resp.JSON200)
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.Notification) (managed.ExternalDelete, error) {
	resp, err := e.client.OrganizationsProjectsNotificationsDestroyWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("delete notification %q", meta.GetExternalName(cr)), resp)
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
