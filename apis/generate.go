// SPDX-FileCopyrightText: 2026 The provider-opencve authors
//
// SPDX-License-Identifier: Apache-2.0

//go:build generate

//go:generate rm -rf ../package/crds
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=../package/crds
//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=../hack/boilerplate.go.txt ./...

package apis

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen" //nolint:typecheck // a main package, imported only to pin the generator in go.mod

	_ "github.com/crossplane/crossplane-tools/cmd/angryjet" //nolint:typecheck // a main package, imported only to pin the generator in go.mod
)
