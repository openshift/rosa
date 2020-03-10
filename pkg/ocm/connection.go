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

package ocm

import (
	"fmt"
	"os"
	"net/url"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/sirupsen/logrus"

	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
)

// ConnectionBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewConnection function instead.
type ConnectionBuilder struct {
	logger *logrus.Logger
	env    string
}

// NewConnection creates a builder that can then be used to configure and build an OCM connection.
// Don't create instances of this type directly; use the NewConnection function instead.
func NewConnection() *ConnectionBuilder {
	return &ConnectionBuilder{}
}

// Logger sets the logger that the connection will use to send messages to the log. This is
// mandatory.
func (b *ConnectionBuilder) Logger(value *logrus.Logger) *ConnectionBuilder {
	b.logger = value
	return b
}

func (b *ConnectionBuilder) SetEnv(value string) *ConnectionBuilder {
	b.env = value
	return b
}

func (b *ConnectionBuilder) getEnvURL() (string, error) {
	switch b.env {
	case "integration":
		return "https://api-integration.6943.hive-integration.openshiftapps.com", nil
	case "int":
		return "https://api-integration.6943.hive-integration.openshiftapps.com", nil
	case "staging":
		return "https://api.stage.openshift.com", nil
	case "stage":
		return "https://api.stage.openshift.com", nil
	case "production":
		return "https://api.openshift.com", nil
	case "prod":
		return "https://api.openshift.com", nil
	default:
		envURL, err := url.ParseRequestURI(b.env)
		if err != nil {
			return "", err
		}
		return envURL.String(), nil
	}
}

// Build uses the information stored in the builder to create a new OCM connection.
func (b *ConnectionBuilder) Build() (result *sdk.Connection, err error) {
	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("logger is mandatory")
		return
	}

	envURL, err := b.getEnvURL()
	if err != nil {
		return
	}
	if envURL == "" {
		err = fmt.Errorf("env is mandatory")
		return
	}

	// Check that there is an OCM token in the environment. This will not be needed once we are
	// able to derive OCM credentials from AWS credentials.
	token := os.Getenv("OCM_TOKEN")
	if token == "" {
		err = fmt.Errorf("environment variable 'OCM_TOKEN' isn't set")
		return
	}

	// Create the OCM logger that uses the logging framework of the project:
	logger, err := logging.NewOCMLogger().
		Logger(b.logger).
		Build()
	if err != nil {
		return
	}

	// Create and populate the object:
	result, err = sdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(token).
		URL(envURL).
		Build()

	return
}
