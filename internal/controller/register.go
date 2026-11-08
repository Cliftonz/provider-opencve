// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Cliftonz/provider-opencve/internal/controller/automation"
	"github.com/Cliftonz/provider-opencve/internal/controller/config"
	"github.com/Cliftonz/provider-opencve/internal/controller/cvetracker"
	"github.com/Cliftonz/provider-opencve/internal/controller/notification"
	"github.com/Cliftonz/provider-opencve/internal/controller/organization"
	"github.com/Cliftonz/provider-opencve/internal/controller/organizationmember"
	"github.com/Cliftonz/provider-opencve/internal/controller/project"
	"github.com/Cliftonz/provider-opencve/internal/controller/projectmember"
	"github.com/Cliftonz/provider-opencve/internal/controller/subscription"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	return run(mgr, o,
		config.Setup,
		project.Setup,
		organization.Setup,
		organizationmember.Setup,
		automation.Setup,
		notification.Setup,
		subscription.Setup,
		cvetracker.Setup,
		projectmember.Setup,
	)
}

func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return run(mgr, o,
		config.SetupGated,
		project.SetupGated,
		organization.SetupGated,
		organizationmember.SetupGated,
		automation.SetupGated,
		notification.SetupGated,
		subscription.SetupGated,
		cvetracker.SetupGated,
		projectmember.SetupGated,
	)
}

func run(mgr ctrl.Manager, o controller.Options, setups ...func(ctrl.Manager, controller.Options) error) error {
	for _, setup := range setups {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
