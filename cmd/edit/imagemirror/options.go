/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imagemirror

import (
	"github.com/openshift/rosa/pkg/reporter"
)

type EditImageMirrorUserOptions struct {
	Id      string
	Type    string
	Mirrors []string
}

type EditImageMirrorOptions struct {
	reporter reporter.Logger
	args     *EditImageMirrorUserOptions
}

func NewEditImageMirrorUserOptions() *EditImageMirrorUserOptions {
	return &EditImageMirrorUserOptions{
		Id:      "",
		Type:    "digest",
		Mirrors: []string{},
	}
}

func NewEditImageMirrorOptions() *EditImageMirrorOptions {
	return &EditImageMirrorOptions{
		reporter: reporter.CreateReporter(),
		args:     NewEditImageMirrorUserOptions(),
	}
}

func (o *EditImageMirrorOptions) Args() *EditImageMirrorUserOptions {
	return o.args
}
