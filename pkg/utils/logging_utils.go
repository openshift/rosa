/*
Copyright (c) 2020 Red Hat, Inc.

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

package utils

import (
	"fmt"

	"github.com/openshift/moactl/pkg/logging"
	rprtr "github.com/openshift/moactl/pkg/reporter"
	"github.com/sirupsen/logrus"
)

// CreateReporterAndLogger will create a reporter object and a logger object.
func CreateReporterAndLogger() (*rprtr.Object, *logrus.Logger, error) {
	reporter, err := rprtr.New().
		Build()

	if err != nil {
		return nil, nil, fmt.Errorf("unable to create reporter: %v", err)
	}

	logger, err := logging.NewLogger().
		Build()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create AWS logger: %v", err)
	}

	return reporter, logger, nil
}
