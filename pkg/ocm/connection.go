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

	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/config"
)

// ConnectionBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewConnection function instead.
type ConnectionBuilder struct {
	logger *logrus.Logger
	token  string
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

// Token sets the token that the connection will use to authenticate the user
func (b *ConnectionBuilder) Token(value string) *ConnectionBuilder {
	b.token = value
	return b
}

// Build uses the information stored in the builder to create a new OCM connection.
func (b *ConnectionBuilder) Build() (result *sdk.Connection, err error) {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		err = fmt.Errorf("Failed to load config file: %v", err)
		return
	}
	if cfg == nil {
		if b.token != "" {
			cfg = new(config.Config)
			cfg.AccessToken = b.token
		} else {
			err = fmt.Errorf("Not logged in, run the 'moactl login' command")
			return
		}
	}

	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("logger is mandatory")
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
	if cfg.TokenURL != "" {
		builder.TokenURL(cfg.TokenURL)
	}
	if cfg.ClientID != "" || cfg.ClientSecret != "" {
		builder.Client(cfg.ClientID, cfg.ClientSecret)
	}
	if cfg.Scopes != nil {
		builder.Scopes(cfg.Scopes...)
	}
	if cfg.URL != "" {
		builder.URL(cfg.URL)
	}
	tokens := make([]string, 0, 2)
	if cfg.AccessToken != "" {
		tokens = append(tokens, cfg.AccessToken)
	}
	if cfg.RefreshToken != "" {
		tokens = append(tokens, cfg.RefreshToken)
	}
	if len(tokens) > 0 {
		builder.Tokens(tokens...)
	}
	builder.Insecure(cfg.Insecure)

	// Create the connection:
	result, err = builder.Build()
	if err != nil {
		return
	}

	return
}
