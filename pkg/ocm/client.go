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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openshift-online/ocm-common/pkg/deprecation"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/sirupsen/logrus"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/helper"
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

func CreateNewClientOrExit(logger *logrus.Logger, reporter reporter.Logger) *Client {
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
			err = fmt.Errorf("failed to load config file: %v", err)
			return nil, err
		}
		if b.cfg == nil {
			err = fmt.Errorf("not logged in, run the 'rosa login' command")
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

	// Add deprecation transport wrapper to automatically handle deprecation headers
	builder.TransportWrapper(deprecation.NewTransportWrapper())

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

	// Persist tokens in the configuration file, the SDK may have refreshed them
	err = config.PersistTokens(b.cfg, accessToken, refreshToken)
	if err != nil {
		b.logger.Warn(context.TODO(),
			fmt.Sprintf("error creating connection. Can't persist tokens to config: %v", err))
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
		return fmt.Errorf("connection is nil")
	}

	accessToken, refreshToken, err := c.GetConnectionTokens(10 * time.Minute)
	if err != nil {
		return fmt.Errorf("can't get new tokens: %v", err)
	}

	err = config.PersistTokens(nil, accessToken, refreshToken)
	if err != nil {
		c.ocm.Logger().Warn(context.TODO(),
			fmt.Sprintf("error creating connection. Can't persist tokens to config: %v", err))
	}

	return nil
}

func (c *Client) GetPolicyVersion(userRequestedVersion string, channelInfo ChannelInfo) (string, error) {
	if userRequestedVersion == "" {
		version, err := c.GetLatestVersion(channelInfo)
		if err != nil {
			return userRequestedVersion, err
		}
		return version, nil
	}

	versionList, err := c.GetVersionsList(channelInfo, false)
	if err != nil {
		err := fmt.Errorf("%v", err)
		return userRequestedVersion, err
	}

	hasVersion := false
	for _, vs := range versionList {
		if vs == userRequestedVersion {
			hasVersion = true
			break
		}
	}

	if !hasVersion {
		versionSet := helper.SliceToMap(versionList)
		err := errors.Errorf(
			"A valid policy version number must be specified\nValid versions: %v",
			helper.MapKeysToString(versionSet),
		)
		return userRequestedVersion, err
	}

	return userRequestedVersion, nil
}

func (c *Client) GetVersionsList(channelInfo ChannelInfo, defaultFirst bool) ([]string, error) {
	response, err := c.GetVersions(channelInfo, defaultFirst)
	if err != nil {
		err := fmt.Errorf("error getting versions: %s", err)
		return make([]string, 0), err
	}
	versionList := make([]string, 0)
	for _, v := range response {
		if !HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		parsedVersion, err := ParseVersion(v.RawID())
		if err != nil {
			err = fmt.Errorf("error parsing version")
			return versionList, err
		}
		versionList = append(versionList, parsedVersion)
	}

	if len(versionList) == 0 {
		err = fmt.Errorf("could not find versions for the provided channel-group: '%s'", channelInfo)
		return versionList, err
	}
	return versionList, nil
}
