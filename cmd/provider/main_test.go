// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type reviewClient struct {
	client.Client
	deniedVerb string
}

func (c reviewClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	sar := obj.(*authv1.SelfSubjectAccessReview)
	sar.Status.Allowed = sar.Spec.ResourceAttributes.Verb != c.deniedVerb
	return nil
}

type reviewManager struct {
	manager.Manager
	client client.Client
}

func (m reviewManager) GetClient() client.Client {
	return m.client
}

func TestCanWatchCRD(t *testing.T) {
	cases := map[string]struct {
		deniedVerb string
		want       bool
	}{
		"AllVerbsAllowed": {want: true},
		"GetDenied":       {deniedVerb: "get", want: false},
		"ListDenied":      {deniedVerb: "list", want: false},
		"WatchDenied":     {deniedVerb: "watch", want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := canWatchCRD(context.Background(), reviewManager{client: reviewClient{deniedVerb: tc.deniedVerb}})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("canWatchCRD() = %v, want %v", got, tc.want)
			}
		})
	}
}
