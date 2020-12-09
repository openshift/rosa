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

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm/config"
)

// ConnectionBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewConnection function instead.
type ConnectionBuilder struct {
	logger *logrus.Logger
	cfg    *config.Config
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

// Config sets the configuration that the connection will use to authenticate the user
func (b *ConnectionBuilder) Config(value *config.Config) *ConnectionBuilder {
	b.cfg = value
	return b
}

// Build uses the information stored in the builder to create a new OCM connection.
func (b *ConnectionBuilder) Build() (result *sdk.Connection, err error) {
	if b.cfg == nil {
		// Load the configuration file:
		b.cfg, err = config.Load()
		if err != nil {
			err = fmt.Errorf("Failed to load config file: %v", err)
			return result, err
		}
		if b.cfg == nil {
			err = fmt.Errorf("Not logged in, run the 'rosa login' command")
			return result, err
		}
	}

	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("Logger is mandatory")
		return
	}

	// Create the OCM logger that uses the logging framework of the project:
	logger, err := logging.NewOCMLogger().
		Logger(b.logger).
		Build()
	if err != nil {
		return
	}

	// Prepare the builder for the connection adding only the properties that have explicit
	// values in the configuration, so that default values won't be overridden:
	builder := sdk.NewConnectionBuilder()
	builder.Logger(logger)
	if b.cfg.TokenURL != "" {
		builder.TokenURL(b.cfg.TokenURL)
	}
	if b.cfg.ClientID != "" || b.cfg.ClientSecret != "" {
		builder.Client(b.cfg.ClientID, b.cfg.ClientSecret)
	}
	if b.cfg.Scopes != nil {
		builder.Scopes(b.cfg.Scopes...)
	}
	if b.cfg.URL != "" {
		builder.URL(b.cfg.URL)
	}
	tokens := make([]string, 0, 2)
	if b.cfg.AccessToken != "" {
		tokens = append(tokens, b.cfg.AccessToken)
	}
	if b.cfg.RefreshToken != "" {
		tokens = append(tokens, b.cfg.RefreshToken)
	}
	if len(tokens) > 0 {
		builder.Tokens(tokens...)
	}
	builder.Insecure(b.cfg.Insecure)

	// Create the connection:
	result, err = builder.Build()
	if err != nil {
		return
	}

	return
}
