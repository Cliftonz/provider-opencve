// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Cliftonz/provider-opencve/apis/v1beta1"
	"github.com/Cliftonz/provider-opencve/internal/clients/opencve"
)

const (
	errGetProviderConfig  = "cannot get referenced ProviderConfig"
	errTrackUsage         = "cannot track ProviderConfig usage"
	errExtractCredentials = "cannot extract credentials"
)

var (
	ErrMalformedCredentials = errors.New("OpenCVE credentials are malformed JSON")
	ErrMissingToken         = errors.New("OpenCVE credentials are missing token")
	ErrMissingOrganization  = errors.New("OpenCVE credentials are missing organization")
	ErrMissingURL           = errors.New("OpenCVE credentials are missing url")

	ErrUnknownProviderConfigKind = errors.New("providerConfigRef kind must be ProviderConfig or ClusterProviderConfig")
)

func NewClient(ctx context.Context, c client.Client, mg resource.ModernManaged) (*opencve.ClientWithResponses, string, error) {
	pcSpec, err := resolveProviderConfig(ctx, c, mg)
	if err != nil {
		return nil, "", err
	}
	data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, c, pcSpec.Credentials.CommonCredentialSelectors)
	if err != nil {
		return nil, "", errors.Wrap(err, errExtractCredentials)
	}
	var creds struct {
		Token        string `json:"token"`
		Organization string `json:"organization"`
		URL          string `json:"url"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, "", fmt.Errorf("%w: %w", ErrMalformedCredentials, err)
	}
	switch {
	case creds.Token == "":
		return nil, "", ErrMissingToken
	case creds.Organization == "":
		return nil, "", ErrMissingOrganization
	case creds.URL == "":
		return nil, "", ErrMissingURL
	}
	oc, err := opencve.NewClientWithResponses(creds.URL+"/api/v2", opencve.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+creds.Token)
		return nil
	}))
	if err != nil {
		return nil, "", err
	}
	return oc, creds.Organization, nil
}

func resolveProviderConfig(ctx context.Context, c client.Client, mg resource.ModernManaged) (*v1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	var pcSpec v1beta1.ProviderConfigSpec
	switch configRef.Kind {
	case v1beta1.ProviderConfigKind:
		pc := &v1beta1.ProviderConfig{}
		if err := c.Get(ctx, types.NamespacedName{Name: configRef.Name, Namespace: mg.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, errGetProviderConfig)
		}
		pcSpec = pc.Spec
		pcSpec.Credentials.SecretRef.Namespace = mg.GetNamespace()
	case v1beta1.ClusterProviderConfigKind:
		pc := &v1beta1.ClusterProviderConfig{}
		if err := c.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
			return nil, errors.Wrap(err, errGetProviderConfig)
		}
		pcSpec = pc.Spec
	default:
		return nil, ErrUnknownProviderConfigKind
	}
	t := resource.NewProviderConfigUsageTracker(c, &v1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	return &pcSpec, nil
}
