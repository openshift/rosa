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
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/reporter"
)

type Client struct {
	ocm *sdk.Connection
}

// ClientBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewClient function instead.
type ClientBuilder struct {
	logger *logrus.Logger
	cfg    *config.Config
}

// NewClient creates a builder that can then be used to configure and build an OCM connection.
func NewClient() *ClientBuilder {
	return &ClientBuilder{}
}

// NewClientWithConnection creates a client with a preexisting connection for testing purpose
func NewClientWithConnection(connection *sdk.Connection) *Client {
	return &Client{
		ocm: connection,
	}
}

func CreateNewClientOrExit(logger *logrus.Logger, reporter *reporter.Object) *Client {
	client, err := NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}

	return client
}

// Logger sets the logger that the connection will use to send messages to the log. This is
// mandatory.
func (b *ClientBuilder) Logger(value *logrus.Logger) *ClientBuilder {
	b.logger = value
	return b
}

// Config sets the configuration that the connection will use to authenticate the user
func (b *ClientBuilder) Config(value *config.Config) *ClientBuilder {
	b.cfg = value
	return b
}

// Build uses the information stored in the builder to create a new OCM connection.
func (b *ClientBuilder) Build() (result *Client, err error) {
	if b.cfg == nil {
		// Load the configuration file:
		b.cfg, err = config.Load()
		if err != nil {
			err = fmt.Errorf("Failed to load config file: %v", err)
			return nil, err
		}
		if b.cfg == nil {
			err = fmt.Errorf("Not logged in, run the 'rosa login' command")
			return nil, err
		}
	}

	// Enable the FedRAMP flag globally
	if b.cfg.FedRAMP {
		fedramp.Enable()
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

	userAgent := info.DefaultUserAgent
	version := info.DefaultVersion
	if b.cfg.UserAgent != "" {
		userAgent = b.cfg.UserAgent
	}
	if b.cfg.Version != "" {
		version = b.cfg.Version
	}
	builder.Agent(userAgent + "/" + version + " " + sdk.DefaultAgent)
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
	conn, err := builder.Build()
	if err != nil {
		return
	}
	accessToken, refreshToken, err := conn.Tokens(10 * time.Minute)
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			return nil, fmt.Errorf("your authorization token needs to be updated. " +
				"Please login again using rosa login")
		}
		return nil, fmt.Errorf("error creating connection. Not able to get authentication token: %s", err)
	}

	// Only execute if the refresh token has changed. This helps limit warnings for users
	// to only on login and when their refresh token is cycled by SSO instead of on every command.
	if b.cfg.RefreshToken != refreshToken {
		offlineTokenDeprecationWarning(refreshToken)
	}

	// Persist tokens in the configuration file, the SDK may have refreshed them
	err = config.PersistTokens(b.cfg, accessToken, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("error creating connection. Can't persist tokens to config: %s", err)
	}

	return &Client{
		ocm: conn,
	}, nil
}

func (c *Client) Close() error {
	return c.ocm.Close()
}

func (c *Client) GetConnectionURL() string {
	return c.ocm.URL()
}

func (c *Client) GetConnectionTokens(expiresIn ...time.Duration) (string, string, error) {
	return c.ocm.Tokens(expiresIn...)
}

func (c *Client) KeepTokensAlive() error {
	if c.ocm == nil {
		return fmt.Errorf("Connection is nil")
	}

	accessToken, refreshToken, err := c.GetConnectionTokens(10 * time.Minute)
	if err != nil {
		return fmt.Errorf("Can't get new tokens: %v", err)
	}

	err = config.PersistTokens(nil, accessToken, refreshToken)
	if err != nil {
		return fmt.Errorf("Can't persist tokens to config: %v", err)
	}

	return nil
}

// Prints a deprecation warning if tokens have changed and the new refresh token contains the 'offline_access' scope
// Swallow and log errors as debug as this is a non-essential warning that should not block the user
func offlineTokenDeprecationWarning(refreshToken string) {
	const offlineTokenDeprecationMessage = "Logging in with offline tokens is being deprecated and will no longer " +
		"be maintained or enhanced. Instead, log in with --use-auth-code or --use-device-code. See 'rosa login --help' " +
		"for usage. Learn more about deprecating offline tokens via https://console.redhat.com/openshift/token/rosa"

	rprtr := reporter.CreateReporter()
	parser := new(jwt.Parser)
	token, _, err := parser.ParseUnverified(refreshToken, jwt.MapClaims{})
	if err != nil {
		rprtr.Debugf("Failed to parse refresh token for deprecation warning: %v", err)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		rprtr.Debugf("Failed to get claims from refresh token for deprecation warning: %v", err)
		return
	}
	scopes, ok := claims["scope"].(string)
	if !ok {
		rprtr.Debugf("Failed to get scopes from refresh token for deprecation warning: %v", err)
		return
	}
	if strings.Contains(scopes, "offline_access") {
		rprtr.Warnf(offlineTokenDeprecationMessage)
		return
	}
}
