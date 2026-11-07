// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package automation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/automation/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	automationName = "nightly"
	automationPath = "/organizations/acme/projects/prod/automations/" + automationName
	remoteReport   = `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"nightly","is_enabled":true,"trigger_type":"report","frequency":"weekly","schedule_timezone":"Europe/Paris","schedule_time":"08:00:00","schedule_weekday":"monday","configuration":{"actions":[{"type":"email","value":"a@example.com"}],"conditions":{}},"conditions_count":2,"last_execution_at":"2026-11-02T08:00:00Z"}`
)

var scheduleKeys = []string{"frequency", "schedule_timezone", "schedule_time", "schedule_weekday"}

type request struct {
	Method string
	Path   string
	Body   map[string]any
}

type fakeAPI struct {
	status   int
	response string
	received []request
}

func connect(t *testing.T, api *fakeAPI) managed.TypedExternalClient[*v1alpha1.Automation] {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		api.received = append(api.received, request{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(api.status)
		_, _ = io.WriteString(w, api.response)
	}))
	t.Cleanup(srv.Close)
	c, err := opencve.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return newExternal(c, "acme")
}

func automation(externalName string, fp v1alpha1.AutomationParameters) *v1alpha1.Automation {
	cr := &v1alpha1.Automation{Spec: v1alpha1.AutomationSpec{ForProvider: fp}}
	meta.SetExternalName(cr, externalName)
	return cr
}

func assertReceived(t *testing.T, api *fakeAPI, want ...request) {
	t.Helper()
	if !reflect.DeepEqual(api.received, want) {
		t.Errorf("received requests\n got: %#v\nwant: %#v", api.received, want)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func raw(s string) *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(s)}
}

func reportSpec() v1alpha1.AutomationParameters {
	return v1alpha1.AutomationParameters{
		ProjectName:      "prod",
		Name:             automationName,
		IsEnabled:        ptr(true),
		TriggerType:      ptr("report"),
		Frequency:        ptr("weekly"),
		ScheduleTimezone: ptr("Europe/Paris"),
		ScheduleTime:     ptr("08:00:00"),
		ScheduleWeekday:  ptr("monday"),
		Configuration:    raw(`{"conditions":{},"actions":[{"value":"a@example.com","type":"email"}]}`),
	}
}

func TestObserveWithoutExternalNameDoesNotExist(t *testing.T) {
	api := &fakeAPI{}
	got, err := connect(t, api).Observe(t.Context(), automation("", reportSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api)
}

func TestObserveNotFoundDoesNotExist(t *testing.T) {
	api := &fakeAPI{status: http.StatusNotFound, response: `{"detail":"Not found."}`}
	got, err := connect(t, api).Observe(t.Context(), automation(automationName, reportSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: automationPath})
}

func TestObserveServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	_, err := connect(t, api).Observe(t.Context(), automation(automationName, reportSpec()))
	if err == nil || !strings.Contains(err.Error(), `cannot get automation "nightly"`) || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want an error reporting HTTP 500", err)
	}
}

func TestObserveExistingAutomationReportsAtProviderAndAvailable(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: remoteReport}
	cr := automation(automationName, reportSpec())
	got, err := connect(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	wantAt := v1alpha1.AutomationObservation{
		ID:               "3fa85f64-5717-4562-b3fc-2c963f66afa6",
		Name:             automationName,
		IsEnabled:        ptr(true),
		TriggerType:      ptr("report"),
		Frequency:        ptr("weekly"),
		ScheduleTimezone: ptr("Europe/Paris"),
		ScheduleTime:     ptr("08:00:00"),
		ScheduleWeekday:  ptr("monday"),
		Configuration:    raw(`{"actions":[{"type":"email","value":"a@example.com"}],"conditions":{}}`),
		ConditionsCount:  ptr(2),
		LastExecutionAt:  &metav1.Time{Time: time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)},
	}
	if !reflect.DeepEqual(cr.Status.AtProvider, wantAt) {
		t.Errorf("atProvider\n got: %+v\nwant: %+v", cr.Status.AtProvider, wantAt)
	}
	if !reflect.DeepEqual(cr.Spec.ForProvider, reportSpec()) {
		t.Errorf("forProvider = %+v, want it unchanged", cr.Spec.ForProvider)
	}
	if c := cr.GetCondition(xpv2.TypeReady); c.Reason != xpv2.ReasonAvailable {
		t.Errorf("ready condition = %+v, want Available", c)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: automationPath})
}

func TestObserveLateInitializesIsEnabledTriggerTypeAndConfiguration(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"nightly","is_enabled":false,"trigger_type":"alert","frequency":null,"schedule_timezone":null,"schedule_time":null,"schedule_weekday":null,"configuration":{"conditions":{"type":"vendor","value":"acme"}},"conditions_count":1,"last_execution_at":null}`}
	cr := automation(automationName, v1alpha1.AutomationParameters{ProjectName: "prod", Name: automationName})
	got, err := connect(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	wantSpec := v1alpha1.AutomationParameters{
		ProjectName:   "prod",
		Name:          automationName,
		IsEnabled:     ptr(false),
		TriggerType:   ptr("alert"),
		Configuration: raw(`{"conditions":{"type":"vendor","value":"acme"}}`),
	}
	if !reflect.DeepEqual(cr.Spec.ForProvider, wantSpec) {
		t.Errorf("forProvider\n got: %+v\nwant: %+v", cr.Spec.ForProvider, wantSpec)
	}
	if at := cr.Status.AtProvider; at.Frequency != nil || at.ScheduleTimezone != nil || at.ScheduleTime != nil || at.ScheduleWeekday != nil || at.LastExecutionAt != nil {
		t.Errorf("atProvider = %+v, want null remote fields to be absent", at)
	}
}

func TestObserveUpToDate(t *testing.T) {
	cases := map[string]struct {
		remote func(map[string]any)
		spec   func(*v1alpha1.AutomationParameters)
		want   bool
	}{
		"ConfigurationSemanticallyEqual": {
			want: true,
		},
		"ConfigurationDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.Configuration = raw(`{"conditions":{},"actions":[]}`) },
		},
		"IsEnabledDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.IsEnabled = ptr(false) },
		},
		"TriggerTypeDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.TriggerType = ptr("alert") },
		},
		"FrequencyDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.Frequency = ptr("daily") },
		},
		"ScheduleTimezoneDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.ScheduleTimezone = ptr("UTC") },
		},
		"ScheduleTimeDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.ScheduleTime = ptr("09:00:00") },
		},
		"ScheduleWeekdayDiffers": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.ScheduleWeekday = ptr("friday") },
		},
		"ScheduleOmittedButSetRemotely": {
			spec: func(fp *v1alpha1.AutomationParameters) { fp.ScheduleWeekday = nil },
		},
		"BlankAndOmittedAreEquivalent": {
			remote: func(r map[string]any) {
				for _, k := range scheduleKeys {
					r[k] = ""
				}
			},
			spec: func(fp *v1alpha1.AutomationParameters) {
				fp.Frequency = nil
				fp.ScheduleTimezone = ptr("")
				fp.ScheduleTime = nil
				fp.ScheduleWeekday = nil
			},
			want: true,
		},
		"MissingKeysAndBlankAreEquivalent": {
			remote: func(r map[string]any) {
				for _, k := range scheduleKeys {
					delete(r, k)
				}
			},
			spec: func(fp *v1alpha1.AutomationParameters) {
				fp.Frequency = ptr("")
				fp.ScheduleTimezone = nil
				fp.ScheduleTime = nil
				fp.ScheduleWeekday = nil
			},
			want: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var remote map[string]any
			if err := json.Unmarshal([]byte(remoteReport), &remote); err != nil {
				t.Fatal(err)
			}
			if tc.remote != nil {
				tc.remote(remote)
			}
			body, err := json.Marshal(remote)
			if err != nil {
				t.Fatal(err)
			}
			fp := reportSpec()
			if tc.spec != nil {
				tc.spec(&fp)
			}
			got, err := connect(t, &fakeAPI{status: http.StatusOK, response: string(body)}).Observe(t.Context(), automation(automationName, fp))
			if err != nil {
				t.Fatal(err)
			}
			if got.ResourceUpToDate != tc.want || got.ResourceLateInitialized {
				t.Errorf("observation = %+v, want up to date %t and no late initialization", got, tc.want)
			}
		})
	}
}

func TestCreatePostsAutomationAndSetsExternalName(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteReport}
	cr := automation("", reportSpec())
	if _, err := connect(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	if got := meta.GetExternalName(cr); got != automationName {
		t.Errorf("external name = %q, want nightly", got)
	}
	assertReceived(t, api, request{Method: http.MethodPost, Path: "/organizations/acme/projects/prod/automations", Body: map[string]any{
		"name":              automationName,
		"is_enabled":        true,
		"trigger_type":      "report",
		"frequency":         "weekly",
		"schedule_timezone": "Europe/Paris",
		"schedule_time":     "08:00:00",
		"schedule_weekday":  "monday",
		"configuration":     map[string]any{"conditions": map[string]any{}, "actions": []any{map[string]any{"type": "email", "value": "a@example.com"}}},
	}})
}

func TestCreateSendsNullsForBlankOrOmittedScheduleAndOmitsUnsetOptionals(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteReport}
	cr := automation("", v1alpha1.AutomationParameters{ProjectName: "prod", Name: automationName, ScheduleTime: ptr("")})
	if _, err := connect(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPost, Path: "/organizations/acme/projects/prod/automations", Body: map[string]any{
		"name":              automationName,
		"frequency":         nil,
		"schedule_timezone": nil,
		"schedule_time":     nil,
		"schedule_weekday":  nil,
	}})
}

func TestCreateFailureIsAnErrorAndLeavesExternalNameUnset(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"name":["automation with this name already exists."]}`}
	cr := automation("", reportSpec())
	_, err := connect(t, api).Create(t.Context(), cr)
	if err == nil || !strings.Contains(err.Error(), `cannot create automation "nightly"`) || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want the API's rejection", err)
	}
	if got := meta.GetExternalName(cr); got != "" {
		t.Errorf("external name = %q, want it unset", got)
	}
}

func TestUpdatePatchesMutableFieldsWithNullsForClearedSchedule(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: remoteReport}
	fp := reportSpec()
	fp.Frequency = nil
	fp.ScheduleTimezone = ptr("")
	fp.ScheduleWeekday = nil
	if _, err := connect(t, api).Update(t.Context(), automation(automationName, fp)); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPatch, Path: automationPath, Body: map[string]any{
		"is_enabled":        true,
		"trigger_type":      "report",
		"frequency":         nil,
		"schedule_timezone": nil,
		"schedule_time":     "08:00:00",
		"schedule_weekday":  nil,
		"configuration":     map[string]any{"conditions": map[string]any{}, "actions": []any{map[string]any{"type": "email", "value": "a@example.com"}}},
	}})
}

func TestUpdateFailureIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"schedule_time":["required for reports"]}`}
	_, err := connect(t, api).Update(t.Context(), automation(automationName, reportSpec()))
	if err == nil || !strings.Contains(err.Error(), `cannot update automation "nightly"`) || !strings.Contains(err.Error(), "required for reports") {
		t.Errorf("error = %v, want the API's rejection", err)
	}
}

func TestDeleteSendsDeleteAndTreatsNotFoundAsDeleted(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound} {
		api := &fakeAPI{status: status}
		if _, err := connect(t, api).Delete(t.Context(), automation(automationName, reportSpec())); err != nil {
			t.Errorf("delete with HTTP %d: %v", status, err)
		}
		assertReceived(t, api, request{Method: http.MethodDelete, Path: automationPath})
	}
}

func TestDeleteServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	_, err := connect(t, api).Delete(t.Context(), automation(automationName, reportSpec()))
	if err == nil || !strings.Contains(err.Error(), `cannot delete automation "nightly"`) || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want an error reporting HTTP 500", err)
	}
}

func TestObserveNullOrMissingConfigurationIsNoConfiguration(t *testing.T) {
	for name, override := range map[string]func(map[string]any){
		"Null":    func(r map[string]any) { r["configuration"] = nil },
		"Missing": func(r map[string]any) { delete(r, "configuration") },
	} {
		t.Run(name, func(t *testing.T) {
			var remote map[string]any
			if err := json.Unmarshal([]byte(remoteReport), &remote); err != nil {
				t.Fatal(err)
			}
			override(remote)
			body, err := json.Marshal(remote)
			if err != nil {
				t.Fatal(err)
			}

			fp := reportSpec()
			fp.Configuration = nil
			cr := automation(automationName, fp)
			got, err := connect(t, &fakeAPI{status: http.StatusOK, response: string(body)}).Observe(t.Context(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}); !reflect.DeepEqual(got, want) {
				t.Errorf("observation = %+v, want %+v", got, want)
			}
			if cr.Spec.ForProvider.Configuration != nil || cr.Status.AtProvider.Configuration != nil {
				t.Errorf("spec configuration = %v, atProvider configuration = %v, want both absent", cr.Spec.ForProvider.Configuration, cr.Status.AtProvider.Configuration)
			}

			got, err = connect(t, &fakeAPI{status: http.StatusOK, response: string(body)}).Observe(t.Context(), automation(automationName, reportSpec()))
			if err != nil {
				t.Fatal(err)
			}
			if got.ResourceUpToDate {
				t.Errorf("observation = %+v, want a spec configuration to drift from no remote configuration", got)
			}
		})
	}
}

func TestObserveDoesNotLateInitializeWhatTheRemoteOmits(t *testing.T) {
	var remote map[string]any
	if err := json.Unmarshal([]byte(remoteReport), &remote); err != nil {
		t.Fatal(err)
	}
	delete(remote, "is_enabled")
	delete(remote, "trigger_type")
	body, err := json.Marshal(remote)
	if err != nil {
		t.Fatal(err)
	}
	fp := reportSpec()
	fp.IsEnabled = nil
	fp.TriggerType = nil
	cr := automation(automationName, fp)
	got, err := connect(t, &fakeAPI{status: http.StatusOK, response: string(body)}).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if got.ResourceLateInitialized {
		t.Errorf("observation = %+v, want no late initialization", got)
	}
	if cr.Spec.ForProvider.IsEnabled != nil || cr.Spec.ForProvider.TriggerType != nil {
		t.Errorf("forProvider = %+v, want isEnabled and triggerType to stay unset", cr.Spec.ForProvider)
	}
}

func TestObserveInvalidSpecConfigurationIsAnError(t *testing.T) {
	fp := reportSpec()
	fp.Configuration = raw(`{"conditions":`)
	_, err := connect(t, &fakeAPI{status: http.StatusOK, response: remoteReport}).Observe(t.Context(), automation(automationName, fp))
	if err == nil {
		t.Error("error = nil, want the configuration decoding error")
	}
}

func TestUpdateSetsAtProviderFromResponse(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: remoteReport}
	cr := automation(automationName, reportSpec())
	if _, err := connect(t, api).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.AutomationObservation{
		ID:               "3fa85f64-5717-4562-b3fc-2c963f66afa6",
		Name:             automationName,
		IsEnabled:        ptr(true),
		TriggerType:      ptr("report"),
		Frequency:        ptr("weekly"),
		ScheduleTimezone: ptr("Europe/Paris"),
		ScheduleTime:     ptr("08:00:00"),
		ScheduleWeekday:  ptr("monday"),
		Configuration:    raw(`{"actions":[{"type":"email","value":"a@example.com"}],"conditions":{}}`),
		ConditionsCount:  ptr(2),
		LastExecutionAt:  &metav1.Time{Time: time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)},
	}
	if !reflect.DeepEqual(cr.Status.AtProvider, want) {
		t.Errorf("atProvider\n got: %+v\nwant: %+v", cr.Status.AtProvider, want)
	}
}
