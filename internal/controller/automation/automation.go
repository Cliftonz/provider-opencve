// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/automation/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
	"github.com/Cliftonz/provider-opencve/internal/controller/kit"
)

type external struct {
	client *opencve.ClientWithResponses
	org    string
}

func newExternal(c *opencve.ClientWithResponses, org string) managed.TypedExternalClient[*v1alpha1.Automation] {
	return &external{client: c, org: org}
}

func (e *external) Observe(ctx context.Context, cr *v1alpha1.Automation) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}
	resp, err := e.client.OrganizationsProjectsAutomationsRetrieveWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalObservation{}, nil
	}
	if err := kit.CheckStatus(fmt.Sprintf("get automation %q", name), resp); err != nil {
		return managed.ExternalObservation{}, err
	}
	at, err := observation(resp.JSON200)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	cr.Status.AtProvider = at
	lateInitialized := lateInitialize(&cr.Spec.ForProvider, at)
	cr.SetConditions(xpv2.Available())
	current, err := upToDate(cr.Spec.ForProvider, at, resp.JSON200.Configuration)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        current,
		ResourceLateInitialized: lateInitialized,
	}, nil
}

func observation(remote *opencve.Automation) (v1alpha1.AutomationObservation, error) {
	frequency, err := kit.OptionalAs(remote.Frequency, opencve.Automation_Frequency.AsPeriodFrequencyEnum)
	if err != nil {
		return v1alpha1.AutomationObservation{}, err
	}
	weekday, err := kit.OptionalAs(remote.ScheduleWeekday, opencve.Automation_ScheduleWeekday.AsScheduleWeekdayEnum)
	if err != nil {
		return v1alpha1.AutomationObservation{}, err
	}
	at := v1alpha1.AutomationObservation{
		ID:               remote.Id.String(),
		Name:             remote.Name,
		IsEnabled:        remote.IsEnabled,
		TriggerType:      (*string)(remote.TriggerType),
		Frequency:        frequency,
		ScheduleTimezone: kit.Optional(remote.ScheduleTimezone),
		ScheduleTime:     kit.Optional(remote.ScheduleTime),
		ScheduleWeekday:  weekday,
		ConditionsCount:  remote.ConditionsCount,
	}
	if remote.Configuration != nil {
		configuration, err := json.Marshal(remote.Configuration)
		if err != nil {
			return v1alpha1.AutomationObservation{}, err
		}
		at.Configuration = &runtime.RawExtension{Raw: configuration}
	}
	if t, err := remote.LastExecutionAt.Get(); err == nil {
		at.LastExecutionAt = &metav1.Time{Time: t}
	}
	return at, nil
}

func lateInitialize(fp *v1alpha1.AutomationParameters, at v1alpha1.AutomationObservation) bool {
	changed := false
	if fp.IsEnabled == nil && at.IsEnabled != nil {
		fp.IsEnabled = at.IsEnabled
		changed = true
	}
	if fp.TriggerType == nil && at.TriggerType != nil {
		fp.TriggerType = at.TriggerType
		changed = true
	}
	if fp.Configuration == nil && at.Configuration != nil {
		fp.Configuration = at.Configuration
		changed = true
	}
	return changed
}

func upToDate(fp v1alpha1.AutomationParameters, at v1alpha1.AutomationObservation, configuration any) (bool, error) {
	want, err := decode(fp.Configuration)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(fp.IsEnabled, at.IsEnabled) &&
		reflect.DeepEqual(fp.TriggerType, at.TriggerType) &&
		kit.Value(fp.Frequency) == kit.Value(at.Frequency) &&
		kit.Value(fp.ScheduleTimezone) == kit.Value(at.ScheduleTimezone) &&
		kit.Value(fp.ScheduleTime) == kit.Value(at.ScheduleTime) &&
		kit.Value(fp.ScheduleWeekday) == kit.Value(at.ScheduleWeekday) &&
		reflect.DeepEqual(want, configuration), nil
}

func decode(c *runtime.RawExtension) (any, error) {
	if c == nil {
		return nil, nil
	}
	var v any
	err := json.Unmarshal(c.Raw, &v)
	return v, err
}

func (e *external) Create(ctx context.Context, cr *v1alpha1.Automation) (managed.ExternalCreation, error) {
	fp := cr.Spec.ForProvider
	frequency, err := kit.NullableFrom(fp.Frequency, (*opencve.AutomationWriteRequest_Frequency).FromPeriodFrequencyEnum)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	weekday, err := kit.NullableFrom(fp.ScheduleWeekday, (*opencve.AutomationWriteRequest_ScheduleWeekday).FromScheduleWeekdayEnum)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	resp, err := e.client.OrganizationsProjectsAutomationsCreateWithResponse(ctx, e.org, fp.ProjectName, opencve.AutomationWriteRequest{
		Name:             fp.Name,
		IsEnabled:        fp.IsEnabled,
		TriggerType:      (*opencve.TriggerTypeEnum)(fp.TriggerType),
		Frequency:        frequency,
		ScheduleTimezone: kit.NullableOf[string](fp.ScheduleTimezone),
		ScheduleTime:     kit.NullableOf[string](fp.ScheduleTime),
		ScheduleWeekday:  weekday,
		Configuration:    configuration(fp.Configuration),
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("create automation %q", fp.Name), resp); err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, fp.Name)
	return managed.ExternalCreation{}, nil
}

func configuration(c *runtime.RawExtension) any {
	if c == nil {
		return nil
	}
	return json.RawMessage(c.Raw)
}

func (e *external) Update(ctx context.Context, cr *v1alpha1.Automation) (managed.ExternalUpdate, error) {
	fp := cr.Spec.ForProvider
	name := meta.GetExternalName(cr)
	frequency, err := kit.NullableFrom(fp.Frequency, (*opencve.PatchedAutomationWriteRequest_Frequency).FromPeriodFrequencyEnum)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	weekday, err := kit.NullableFrom(fp.ScheduleWeekday, (*opencve.PatchedAutomationWriteRequest_ScheduleWeekday).FromScheduleWeekdayEnum)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	resp, err := e.client.OrganizationsProjectsAutomationsPartialUpdateWithResponse(ctx, e.org, fp.ProjectName, name, opencve.PatchedAutomationWriteRequest{
		IsEnabled:        fp.IsEnabled,
		TriggerType:      (*opencve.TriggerTypeEnum)(fp.TriggerType),
		Frequency:        frequency,
		ScheduleTimezone: kit.NullableOf[string](fp.ScheduleTimezone),
		ScheduleTime:     kit.NullableOf[string](fp.ScheduleTime),
		ScheduleWeekday:  weekday,
		Configuration:    configuration(fp.Configuration),
	})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := kit.CheckStatus(fmt.Sprintf("update automation %q", name), resp); err != nil {
		return managed.ExternalUpdate{}, err
	}
	at, err := observation(resp.JSON200)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	cr.Status.AtProvider = at
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, cr *v1alpha1.Automation) (managed.ExternalDelete, error) {
	name := meta.GetExternalName(cr)
	resp, err := e.client.OrganizationsProjectsAutomationsDestroyWithResponse(ctx, e.org, cr.Spec.ForProvider.ProjectName, name)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	if kit.NotFound(resp) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, kit.CheckStatus(fmt.Sprintf("delete automation %q", name), resp)
}

func (e *external) Disconnect(context.Context) error {
	return nil
}
