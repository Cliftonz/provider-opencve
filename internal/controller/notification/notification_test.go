// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/Cliftonz/provider-opencve/apis/notification/v1alpha1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	projectName      = "prod"
	notificationName = "ops"
	emailType        = "email"
	webhookType      = "webhook"
	notificationPath = "/organizations/acme/projects/prod/notifications/" + notificationName
	remoteWebhook    = `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"ops","type":"webhook","is_enabled":true,"is_pending_email_confirmation":false,"created_at":"2026-11-01T08:00:00Z","updated_at":"2026-11-02T08:00:00Z"}`
)

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

func connect(t *testing.T, api *fakeAPI, secrets ...*corev1.Secret) managed.TypedExternalClient[*v1alpha1.Notification] {
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
	b := kfake.NewClientBuilder()
	for _, s := range secrets {
		b = b.WithObjects(s)
	}
	return &external{kube: b.Build(), client: c, org: "acme"}
}

func notification(externalName string, fp v1alpha1.NotificationParameters) *v1alpha1.Notification {
	cr := &v1alpha1.Notification{
		ObjectMeta: metav1.ObjectMeta{Name: "ops-alerts", Namespace: "team-a"},
		Spec:       v1alpha1.NotificationSpec{ForProvider: fp},
	}
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

func webhookSpec() v1alpha1.NotificationParameters {
	return v1alpha1.NotificationParameters{
		ProjectName: projectName,
		Name:        notificationName,
		Type:        webhookType,
		IsEnabled:   ptr(true),
		ConfigurationSecretRef: &xpv2.LocalSecretKeySelector{
			LocalSecretReference: xpv2.LocalSecretReference{Name: "ops-webhook"},
			Key:                  "configuration",
		},
	}
}

const webhookConfiguration = `{"extras":{"url":"https://hooks.example.com/opencve","headers":{"X-Token":"s3cr3t"}},"types":["created"],"metrics":{"cvss31":"7"}}`

func webhookSecret() *corev1.Secret {
	return webhookSecretWith(webhookConfiguration)
}

func webhookSecretWith(configuration string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "ops-webhook", Namespace: "team-a"},
		Data:       map[string][]byte{"configuration": []byte(configuration)},
	}
}

func appliedHash(t *testing.T, configuration string) string {
	t.Helper()
	cr := notification(notificationName, webhookSpec())
	if _, err := connect(t, &fakeAPI{status: http.StatusOK, response: remoteWebhook}, webhookSecretWith(configuration)).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	return cr.Status.AtProvider.ConfigurationHash
}

func TestObserveWithoutExternalNameDoesNotExist(t *testing.T) {
	api := &fakeAPI{}
	got, err := connect(t, api).Observe(t.Context(), notification("", webhookSpec()))
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
	got, err := connect(t, api).Observe(t.Context(), notification(notificationName, webhookSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, managed.ExternalObservation{}) {
		t.Errorf("observation = %+v, want resource not to exist", got)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: notificationPath})
}

func TestObserveDeletedNotificationExistsWithoutItsSecret(t *testing.T) {
	cr := notification(notificationName, webhookSpec())
	cr.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
	got, err := connect(t, &fakeAPI{status: http.StatusOK, response: remoteWebhook}).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ResourceExists {
		t.Errorf("observation = %+v, want the notification to exist so it can be deleted", got)
	}
}

func TestObserveServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	_, err := connect(t, api).Observe(t.Context(), notification(notificationName, webhookSpec()))
	if err == nil || !strings.Contains(err.Error(), `get notification "ops"`) || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want an error naming the notification and reporting HTTP 500", err)
	}
}

func TestObserveExistingNotificationReportsAtProviderAndAvailable(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: remoteWebhook}
	cr := notification(notificationName, webhookSpec())
	cr.Status.AtProvider.ConfigurationHash = appliedHash(t, webhookConfiguration)
	got, err := connect(t, api, webhookSecret()).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	wantAt := v1alpha1.NotificationObservation{
		ID:                         "3fa85f64-5717-4562-b3fc-2c963f66afa6",
		Name:                       notificationName,
		Type:                       webhookType,
		IsEnabled:                  ptr(true),
		IsPendingEmailConfirmation: ptr(false),
		ConfigurationHash:          appliedHash(t, webhookConfiguration),
	}
	if !reflect.DeepEqual(cr.Status.AtProvider, wantAt) {
		t.Errorf("atProvider\n got: %+v\nwant: %+v", cr.Status.AtProvider, wantAt)
	}
	if !reflect.DeepEqual(cr.Spec.ForProvider, webhookSpec()) {
		t.Errorf("forProvider = %+v, want it unchanged", cr.Spec.ForProvider)
	}
	if c := cr.GetCondition(xpv2.TypeReady); c.Reason != xpv2.ReasonAvailable {
		t.Errorf("ready condition = %+v, want Available", c)
	}
	assertReceived(t, api, request{Method: http.MethodGet, Path: notificationPath})
}

func TestObserveNullOrMissingRemoteValuesAreEmpty(t *testing.T) {
	for name, remote := range map[string]string{
		"Null":    `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"ops","type":"email","is_enabled":null,"is_pending_email_confirmation":null}`,
		"Missing": `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"ops","type":"email"}`,
	} {
		t.Run(name, func(t *testing.T) {
			cr := notification(notificationName, v1alpha1.NotificationParameters{ProjectName: projectName, Name: notificationName, Type: emailType})
			got, err := connect(t, &fakeAPI{status: http.StatusOK, response: remote}).Observe(t.Context(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}); !reflect.DeepEqual(got, want) {
				t.Errorf("observation = %+v, want %+v", got, want)
			}
			if want := (v1alpha1.NotificationObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: notificationName, Type: emailType}); !reflect.DeepEqual(cr.Status.AtProvider, want) {
				t.Errorf("atProvider = %+v, want %+v", cr.Status.AtProvider, want)
			}
		})
	}
}

func TestObserveLateInitializesIsEnabled(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"ops","type":"email","is_enabled":false,"is_pending_email_confirmation":true}`}
	cr := notification(notificationName, v1alpha1.NotificationParameters{ProjectName: projectName, Name: notificationName, Type: emailType})
	got, err := connect(t, api).Observe(t.Context(), cr)
	if err != nil {
		t.Fatal(err)
	}
	if want := (managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("observation = %+v, want %+v", got, want)
	}
	if want := (v1alpha1.NotificationParameters{ProjectName: projectName, Name: notificationName, Type: emailType, IsEnabled: ptr(false)}); !reflect.DeepEqual(cr.Spec.ForProvider, want) {
		t.Errorf("forProvider\n got: %+v\nwant: %+v", cr.Spec.ForProvider, want)
	}
}

func TestObserveUpToDate(t *testing.T) {
	cases := map[string]struct {
		spec    func(*v1alpha1.NotificationParameters)
		applied string
		want    bool
	}{
		"Matches":                   {applied: webhookConfiguration, want: true},
		"TypeIsNotCompared":         {spec: func(fp *v1alpha1.NotificationParameters) { fp.Type = "slack" }, applied: webhookConfiguration, want: true},
		"IsEnabledDiffers":          {spec: func(fp *v1alpha1.NotificationParameters) { fp.IsEnabled = ptr(false) }, applied: webhookConfiguration},
		"ConfigurationReformatted":  {applied: "{ \"metrics\": {\"cvss31\": \"7\"},\n  \"types\": [\"created\"],\n  \"extras\": {\"headers\": {\"X-Token\": \"s3cr3t\"}, \"url\": \"https://hooks.example.com/opencve\"} }", want: true},
		"ConfigurationChanged":      {applied: `{"extras":{"url":"https://old.example.com/opencve"}}`},
		"ConfigurationNeverApplied": {},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fp := webhookSpec()
			if tc.spec != nil {
				tc.spec(&fp)
			}
			cr := notification(notificationName, fp)
			if tc.applied != "" {
				cr.Status.AtProvider.ConfigurationHash = appliedHash(t, tc.applied)
			}
			got, err := connect(t, &fakeAPI{status: http.StatusOK, response: remoteWebhook}, webhookSecret()).Observe(t.Context(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if got.ResourceUpToDate != tc.want || got.ResourceLateInitialized {
				t.Errorf("observation = %+v, want up to date %t and no late initialization", got, tc.want)
			}
		})
	}
}

func TestCreatePostsNotificationWithSecretConfigurationAndSetsExternalName(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteWebhook}
	cr := notification("", webhookSpec())
	if _, err := connect(t, api, webhookSecret()).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	if got := meta.GetExternalName(cr); got != notificationName {
		t.Errorf("external name = %q, want %s", got, notificationName)
	}
	assertReceived(t, api, request{Method: http.MethodPost, Path: "/organizations/acme/projects/prod/notifications", Body: map[string]any{
		"name":       notificationName,
		"type":       webhookType,
		"is_enabled": true,
		"configuration": map[string]any{
			"extras":  map[string]any{"url": "https://hooks.example.com/opencve", "headers": map[string]any{"X-Token": "s3cr3t"}},
			"types":   []any{"created"},
			"metrics": map[string]any{"cvss31": "7"},
		},
	}})
}

func TestCreateWithoutConfigurationSecretRefOmitsConfiguration(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteWebhook}
	cr := notification("", v1alpha1.NotificationParameters{ProjectName: projectName, Name: notificationName, Type: emailType})
	if _, err := connect(t, api).Create(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPost, Path: "/organizations/acme/projects/prod/notifications", Body: map[string]any{
		"name": notificationName,
		"type": "email",
	}})
}

func TestCreateFailureIsAnErrorAndLeavesExternalNameUnset(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"name":["notification with this name already exists."]}`}
	cr := notification("", webhookSpec())
	_, err := connect(t, api, webhookSecret()).Create(t.Context(), cr)
	if err == nil || !strings.Contains(err.Error(), `create notification "ops"`) || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want the API's rejection naming the notification", err)
	}
	if got := meta.GetExternalName(cr); got != "" {
		t.Errorf("external name = %q, want it unset", got)
	}
}

func TestCreateWithMissingSecretPropagatesNotFound(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteWebhook}
	_, err := connect(t, api).Create(t.Context(), notification("", webhookSpec()))
	if !kerrors.IsNotFound(err) {
		t.Errorf("error = %v, want the Secret's not found error", err)
	}
	assertReceived(t, api)
}

func TestCreateWithMissingSecretKeyIsANamedError(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteWebhook}
	s := webhookSecret()
	s.Data = map[string][]byte{"other": []byte(`{}`)}
	_, err := connect(t, api, s).Create(t.Context(), notification("", webhookSpec()))
	if !errors.Is(err, errMissingConfigurationKey) || !strings.Contains(err.Error(), `"configuration"`) || !strings.Contains(err.Error(), `"ops-webhook"`) {
		t.Errorf("error = %v, want errMissingConfigurationKey naming key configuration and Secret ops-webhook", err)
	}
	assertReceived(t, api)
}

func TestCreateWithInvalidConfigurationJSONWrapsTheDecodeError(t *testing.T) {
	api := &fakeAPI{status: http.StatusCreated, response: remoteWebhook}
	_, err := connect(t, api, webhookSecretWith(`{"extras":`)).Create(t.Context(), notification("", webhookSpec()))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) || !strings.Contains(err.Error(), `"configuration"`) || !strings.Contains(err.Error(), `"ops-webhook"`) {
		t.Errorf("error = %v, want a wrapped JSON syntax error naming key configuration and Secret ops-webhook", err)
	}
	assertReceived(t, api)
}

func TestUpdatePatchesIsEnabledOnlyWhenTheConfigurationIsApplied(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: remoteWebhook}
	fp := webhookSpec()
	fp.IsEnabled = ptr(false)
	cr := notification(notificationName, fp)
	cr.Status.AtProvider.ConfigurationHash = appliedHash(t, webhookConfiguration)
	if _, err := connect(t, api, webhookSecret()).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	assertReceived(t, api, request{Method: http.MethodPatch, Path: notificationPath, Body: map[string]any{
		"is_enabled": false,
	}})
}

func TestUpdatePatchesTheSecretConfigurationWhenItIsNotApplied(t *testing.T) {
	for name, applied := range map[string]string{
		"SecretChanged": `{"extras":{"url":"https://old.example.com/opencve"}}`,
		"NeverApplied":  "",
	} {
		t.Run(name, func(t *testing.T) {
			api := &fakeAPI{status: http.StatusOK, response: remoteWebhook}
			cr := notification(notificationName, webhookSpec())
			if applied != "" {
				cr.Status.AtProvider.ConfigurationHash = appliedHash(t, applied)
			}
			if _, err := connect(t, api, webhookSecret()).Update(t.Context(), cr); err != nil {
				t.Fatal(err)
			}
			assertReceived(t, api, request{Method: http.MethodPatch, Path: notificationPath, Body: map[string]any{
				"is_enabled": true,
				"configuration": map[string]any{
					"extras":  map[string]any{"url": "https://hooks.example.com/opencve", "headers": map[string]any{"X-Token": "s3cr3t"}},
					"types":   []any{"created"},
					"metrics": map[string]any{"cvss31": "7"},
				},
			}})
			got, err := connect(t, &fakeAPI{status: http.StatusOK, response: remoteWebhook}, webhookSecret()).Observe(t.Context(), cr)
			if err != nil {
				t.Fatal(err)
			}
			if !got.ResourceUpToDate {
				t.Errorf("observation after update = %+v, want up to date", got)
			}
		})
	}
}

func TestUpdateSetsAtProviderFromResponse(t *testing.T) {
	api := &fakeAPI{status: http.StatusOK, response: `{"id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","name":"ops","type":"webhook","is_enabled":false,"is_pending_email_confirmation":null}`}
	fp := webhookSpec()
	fp.IsEnabled = ptr(false)
	cr := notification(notificationName, fp)
	cr.Status.AtProvider = v1alpha1.NotificationObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: notificationName, Type: webhookType, IsEnabled: ptr(true), IsPendingEmailConfirmation: ptr(false), ConfigurationHash: appliedHash(t, webhookConfiguration)}
	if _, err := connect(t, api, webhookSecret()).Update(t.Context(), cr); err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.NotificationObservation{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: notificationName, Type: webhookType, IsEnabled: ptr(false), ConfigurationHash: appliedHash(t, webhookConfiguration)}
	if !reflect.DeepEqual(cr.Status.AtProvider, want) {
		t.Errorf("atProvider\n got: %+v\nwant: %+v", cr.Status.AtProvider, want)
	}
}

func TestUpdateFailureIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, response: `{"is_enabled":["invalid"]}`}
	_, err := connect(t, api, webhookSecret()).Update(t.Context(), notification(notificationName, webhookSpec()))
	if err == nil || !strings.Contains(err.Error(), `update notification "ops"`) || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %v, want the API's rejection naming the notification", err)
	}
}

func TestDeleteSendsDeleteAndTreatsNotFoundAsDeleted(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound} {
		api := &fakeAPI{status: status}
		if _, err := connect(t, api).Delete(t.Context(), notification(notificationName, webhookSpec())); err != nil {
			t.Errorf("delete with HTTP %d: %v", status, err)
		}
		assertReceived(t, api, request{Method: http.MethodDelete, Path: notificationPath})
	}
}

func TestDeleteServerErrorIsAnError(t *testing.T) {
	api := &fakeAPI{status: http.StatusInternalServerError, response: `{"detail":"boom"}`}
	_, err := connect(t, api).Delete(t.Context(), notification(notificationName, webhookSpec()))
	if err == nil || !strings.Contains(err.Error(), `delete notification "ops"`) || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want an error naming the notification and reporting HTTP 500", err)
	}
}
