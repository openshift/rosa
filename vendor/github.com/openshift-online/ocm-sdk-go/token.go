/*
Copyright (c) 2018 Red Hat, Inc.

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

// This file contains the implementations of the methods of the connection that handle OpenID
// authentication tokens.

package sdk

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/openshift-online/ocm-sdk-go/internal"
)

const (
	tokenExpiry = 1 * time.Minute
)

// Tokens returns the access and refresh tokens that is currently in use by the connection. If it is
// necessary to request a new token because it wasn't requested yet, or because it is expired, this
// method will do it and will return an error if it fails.
//
// This operation is potentially lengthy, as it may require network communication. Consider using a
// context and the TokensContext method.
func (c *Connection) Tokens(expiresIn ...time.Duration) (access, refresh string, err error) {
	if len(expiresIn) == 1 {
		access, refresh, err = c.TokensContext(context.Background(), expiresIn[0])
	} else {
		access, refresh, err = c.TokensContext(context.Background())
	}
	return

}

// TokensContext returns the access and refresh tokens that is currently in use by the connection.
// If it is necessary to request a new token because it wasn't requested yet, or because it is
// expired, this method will do it and will return an error if it fails.
// The function will retry the operation in an exponential-backoff method.
func (c *Connection) TokensContext(
	ctx context.Context,
	expiresIn ...time.Duration,
) (access, refresh string, err error) {
	expiresDuration := tokenExpiry
	if len(expiresIn) == 1 {
		expiresDuration = expiresIn[0]
	}

	// Configure the back-off so that it honours the deadline of the context passed
	// to the method. Note that we need to specify explicitly the type of the variable
	// because the backoff.NewExponentialBackOff function returns the implementation
	// type but backoff.WithContext returns the interface instead.
	exponentialBackoffMethod := backoff.NewExponentialBackOff()
	exponentialBackoffMethod.MaxElapsedTime = 15 * time.Second
	var backoffMethod backoff.BackOff = exponentialBackoffMethod
	if ctx != nil {
		backoffMethod = backoff.WithContext(backoffMethod, ctx)
	}

	attempt := 0
	operation := func() error {
		attempt++
		var code int
		code, access, refresh, err = c.tokensContext(ctx, attempt, expiresDuration)
		if err != nil {
			if code >= http.StatusInternalServerError {
				c.logger.Error(ctx,
					"OCM auth: failed to get tokens, got http code %d, "+
						"will attempt to retry. err: %v",
					code, err)
				return err
			}
			c.logger.Error(ctx,
				"OCM auth: failed to get tokens, got http code %d, "+
					"will not attempt to retry. err: %v",
				code, err)
			return backoff.Permanent(err)
		}

		if attempt > 1 {
			c.logger.Info(ctx, "OCM auth: got tokens on attempt %d.", attempt)
		} else {
			c.logger.Debug(ctx, "OCM auth: got tokens on attempt %d.", attempt)
		}
		return nil
	}

	// nolint
	backoff.Retry(operation, backoffMethod)
	return access, refresh, err
}

func (c *Connection) tokensContext(
	ctx context.Context,
	attempt int,
	expiresIn time.Duration,
) (code int, access, refresh string, err error) {
	// We need to make sure that this method isn't execute concurrently, as we will be updating
	// multiple attributes of the connection:
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Check the expiration times of the tokens:
	now := time.Now()
	var accessExpires bool
	var accessLeft time.Duration
	if c.accessToken != nil {
		accessExpires, accessLeft, err = GetTokenExpiry(c.accessToken, now)
		if err != nil {
			return
		}
	}
	var refreshExpires bool
	var refreshLeft time.Duration
	if c.refreshToken != nil {
		refreshExpires, refreshLeft, err = GetTokenExpiry(c.refreshToken, now)
		if err != nil {
			return
		}
	}
	if c.logger.DebugEnabled() {
		c.debugExpiry(ctx, "Bearer", c.accessToken, accessExpires, accessLeft)
		c.debugExpiry(ctx, "Refresh", c.refreshToken, refreshExpires, refreshLeft)
	}

	// If the access token is available and it isn't expired or about to expire then we can
	// return the current tokens directly:
	if c.accessToken != nil && (!accessExpires || accessLeft >= expiresIn) {
		access, refresh = c.currentTokens()
		return
	}

	// At this point we know that the access token is unavailable, expired or about to expire.
	c.logger.Debug(ctx, "OCM auth: trying to get new tokens (attempt %d)", attempt)

	// So we need to check if we can use the refresh token to request a new one.
	if c.refreshToken != nil && (!refreshExpires || refreshLeft >= expiresIn) {
		code, _, err = c.sendRefreshTokenForm(ctx, attempt)
		if err != nil {
			return
		}
		access, refresh = c.currentTokens()
		return
	}

	// Now we know that both the access and refresh tokens are unavailable, expired or about to
	// expire. So we need to check if we have other credentials that can be used to request a
	// new token, and use them.
	if c.haveCredentials() {
		code, _, err = c.sendRequestTokenForm(ctx, attempt)
		if err != nil {
			return
		}
		access, refresh = c.currentTokens()
		return
	}

	// Here we know that the access and refresh tokens are unavailable, expired or about to
	// expire. We also know that we don't have credentials to request new ones. But we could
	// still use the refresh token if it isn't completely expired.
	if c.refreshToken != nil && refreshLeft > 0 {
		c.logger.Warn(
			ctx,
			"OCM auth: refresh token expires in only %s, but there is no other mechanism to "+
				"obtain a new token, so will try to use it anyhow",
			refreshLeft,
		)
		code, _, err = c.sendRefreshTokenForm(ctx, attempt)
		if err != nil {
			return
		}
		access, refresh = c.currentTokens()
		return
	}

	// At this point we know that the access token is expired or about to expire. We know also
	// that the refresh token is unavailable or completely expired. And we know that we don't
	// have credentials to request new tokens. But we can still use the access token if it isn't
	// expired.
	if c.accessToken != nil && accessLeft > 0 {
		c.logger.Warn(
			ctx,
			"OCM auth: access token expires in only %s, but there is no other mechanism to "+
				"obtain a new token, so will try to use it anyhow",
			accessLeft,
		)
		access, refresh = c.currentTokens()
		return
	}

	// There is no way to get a valid access token, so all we can do is report the failure:
	err = fmt.Errorf(
		"OCM auth: access and refresh tokens are unavailable or expired, and there are no " +
			"password or client secret to request new ones",
	)

	return
}

// currentTokens returns the current tokens without trying to send any request to refresh them, and
// checking that they are actually available. If they aren't available then it will return empty
// strings.
func (c *Connection) currentTokens() (access, refresh string) {
	if c.accessToken != nil {
		access = c.accessToken.Raw
	}
	if c.refreshToken != nil {
		refresh = c.refreshToken.Raw
	}
	return
}

func (c *Connection) sendRequestTokenForm(ctx context.Context, attempt int) (code int,
	result *internal.TokenResponse, err error) {
	form := url.Values{}
	if c.havePassword() {
		c.logger.Debug(ctx, "OCM auth: requesting new token using the password grant")
		form.Set("grant_type", "password")
		form.Set("client_id", c.clientID)
		form.Set("username", c.user)
		form.Set("password", c.password)
	} else if c.haveSecret() {
		c.logger.Debug(ctx, "OCM auth: requesting new token using the client credentials grant")
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", c.clientID)
		form.Set("client_secret", c.clientSecret)
	} else {
		err = fmt.Errorf(
			"either password or client secret must be provided",
		)
		return
	}
	form.Set("scope", strings.Join(c.scopes, " "))
	return c.sendTokenForm(ctx, form, attempt)
}

func (c *Connection) sendRefreshTokenForm(ctx context.Context, attempt int) (code int,
	result *internal.TokenResponse, err error) {
	// Send the refresh token grant form:
	c.logger.Debug(ctx, "OCM auth: requesting new token using the refresh token grant")
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("refresh_token", c.refreshToken.Raw)
	code, result, err = c.sendTokenForm(ctx, form, attempt)

	// If the server returns an 'invalid_grant' error response then it may be that the
	// session has expired even if the tokens have not expired. This may happen when the SSO
	// server has been restarted or its session caches have been cleared. In theory that should
	// not happen, but in practice it happens from time to time, specially when using the client
	// credentials grant. To handle that smoothly we request new tokens if we have credentials
	// to do so.
	if err != nil && result != nil {
		var errorCode string
		if result.Error != nil {
			errorCode = *result.Error
		}
		var errorDescription string
		if result.ErrorDescription != nil {
			errorDescription = *result.ErrorDescription
		}
		if errorCode == "invalid_grant" && c.haveCredentials() {
			c.logger.Info(
				ctx,
				"OCM auth: server returned error code '%s' and error description '%s' "+
					"when the refresh token isn't expired",
				errorCode, errorDescription,
			)
			return c.sendRequestTokenForm(ctx, attempt)
		}
	}

	return
}

func (c *Connection) sendTokenForm(
	ctx context.Context,
	form url.Values,
	attempt int,
) (code int, result *internal.TokenResponse, err error) {
	// Measure the time that it takes to send the request and receive the response:
	start := time.Now()
	code, result, err = c.sendTokenFormTimed(ctx, form)
	elapsed := time.Since(start)

	// Update the metrics:
	if c.tokenCountMetric != nil || c.tokenDurationMetric != nil {
		labels := map[string]string{
			metricsAttemptLabel: strconv.Itoa(attempt),
			metricsCodeLabel:    strconv.Itoa(code),
		}
		if c.tokenCountMetric != nil {
			c.tokenCountMetric.With(labels).Inc()
		}
		if c.tokenDurationMetric != nil {
			c.tokenDurationMetric.With(labels).Observe(elapsed.Seconds())
		}
	}

	// Return the original error:
	return
}

func (c *Connection) sendTokenFormTimed(ctx context.Context, form url.Values) (code int,
	result *internal.TokenResponse, err error) {
	// Create the HTTP request:
	body := []byte(form.Encode())
	request, err := http.NewRequest(http.MethodPost, c.tokenURL.String(), bytes.NewReader(body))
	request.Close = true
	header := request.Header
	if c.agent != "" {
		header.Set("User-Agent", c.agent)
	}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Accept", "application/json")
	if err != nil {
		err = fmt.Errorf("can't create request: %w", err)
		return
	}

	// Set the context:
	if ctx != nil {
		request = request.WithContext(ctx)
	}

	// Select the HTTP client:
	client, err := c.selectClient(ctx, c.tokenURL)
	if err != nil {
		return
	}

	// Send the HTTP request:
	response, err := client.Do(request)
	if err != nil {
		err = fmt.Errorf("can't send request: %w", err)
		return
	}
	defer response.Body.Close()

	code = response.StatusCode

	// Check that the response content type is JSON:
	err = c.checkContentType(response)
	if err != nil {
		return
	}

	// Read the response body:
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("can't read response: %w", err)
		return
	}

	// Parse the response body:
	result = &internal.TokenResponse{}
	err = json.Unmarshal(body, result)
	if err != nil {
		err = fmt.Errorf("can't parse JSON response: %w", err)
		return
	}
	if result.Error != nil {
		if result.ErrorDescription != nil {
			err = fmt.Errorf("%s: %s", *result.Error, *result.ErrorDescription)
			return
		}
		err = fmt.Errorf("%s", *result.Error)
		return
	}
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("token response status code is '%d'", response.StatusCode)
		return
	}
	if result.TokenType != nil && *result.TokenType != "bearer" {
		err = fmt.Errorf("expected 'bearer' token type but got '%s", *result.TokenType)
		return
	}
	if result.AccessToken == nil {
		err = fmt.Errorf("no access token was received")
		return
	}
	accessToken, _, err := c.tokenParser.ParseUnverified(*result.AccessToken, jwt.MapClaims{})
	if err != nil {
		return
	}
	if result.RefreshToken == nil {
		err = fmt.Errorf("no refresh token was received")
		return
	}
	refreshToken, _, err := c.tokenParser.ParseUnverified(*result.RefreshToken, jwt.MapClaims{})
	if err != nil {
		return
	}

	// Save the new tokens:
	c.accessToken = accessToken
	c.refreshToken = refreshToken

	return
}

// haveCredentials returns true if the connection has credentials that can be used to request new
// tokens.
func (c *Connection) haveCredentials() bool {
	return c.havePassword() || c.haveSecret()
}

func (c *Connection) havePassword() bool {
	return c.user != "" && c.password != ""
}

func (c *Connection) haveSecret() bool {
	return c.clientID != "" && c.clientSecret != ""
}

// debugExpiry sends to the log information about the expiration of the given token.
func (c *Connection) debugExpiry(ctx context.Context, typ string, token *jwt.Token, expires bool,
	left time.Duration) {
	if token != nil {
		if expires {
			if left < 0 {
				c.logger.Debug(ctx, "OCM auth: %s token expired %s ago", typ, -left)
			} else if left > 0 {
				c.logger.Debug(ctx, "OCM auth: %s token expires in %s", typ, left)
			} else {
				c.logger.Debug(ctx, "OCM auth: %s token expired just now", typ)
			}
		}
	} else {
		c.logger.Debug(ctx, "OCM auth: %s token isn't available", typ)
	}
}

// GetTokenExpiry determines if the given token expires, and the time that remains till it expires.
func GetTokenExpiry(token *jwt.Token, now time.Time) (expires bool,
	left time.Duration, err error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("expected map claims bug got %T", claims)
		return
	}
	var exp float64
	claim, ok := claims["exp"]
	if ok {
		exp, ok = claim.(float64)
		if !ok {
			err = fmt.Errorf("expected floating point 'exp' but got %T", claim)
			return
		}
	}
	if exp == 0 {
		expires = false
		left = 0
	} else {
		expires = true
		left = time.Unix(int64(exp), 0).Sub(now)
	}
	return
}
