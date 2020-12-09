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

// This file contains the code to build the default loggers used by the project.

package logging

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/debug"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// LoggerBuilder contains the information and logic needed to create the default loggers used by
// the project. Don't create instances of this type directly; use the NewLogger function instead.
type LoggerBuilder struct {
}

// NewLogger creates new builder that can then be used to configure and build an OCM logger that
// uses the logging framework of the project.
func NewLogger() *LoggerBuilder {
	return &LoggerBuilder{}
}

// Build uses the information stored in the builder to create a new logger.
func (b *LoggerBuilder) Build() (result *logrus.Logger, err error) {
	// Create the logger:
	result = logrus.New()
	result.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Enable the debug level if needed:
	if debug.Enabled() {
		result.SetLevel(logrus.DebugLevel)
	}

	return
}

// CreateLoggerOrExit creates the logger instance or exits to the console
// noting the error on failure.
func CreateLoggerOrExit(reporter *rprtr.Object) *logrus.Logger {
	// Create the logger:
	logger, err := NewLogger().Build()
	if err != nil {
		reporter.Errorf("Failed to create logger: %v", err)
		os.Exit(1)
	}
	return logger
}
