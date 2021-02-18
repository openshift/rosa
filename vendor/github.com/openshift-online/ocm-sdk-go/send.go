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

// This file contains the implementation of the methods of the connection that are used to send HTTP
// requests and receive HTTP responses.

package sdk

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"path"
	"regexp"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
)

var wsRegex = regexp.MustCompile(`\s+`)

// RoundTrip is the implementation of the http.RoundTripper interface.
func (c *Connection) RoundTrip(request *http.Request) (response *http.Response, err error) {
	// Check if the connection is closed:
	err = c.checkClosed()
	if err != nil {
		return
	}

	// Get the context from the request:
	ctx := request.Context()

	// Check the request URL:
	if request.URL.Path == "" {
		err = fmt.Errorf("request path is mandatory")
		return
	}
	if request.URL.Scheme != "" || request.URL.Host != "" || !path.IsAbs(request.URL.Path) {
		err = fmt.Errorf("request URL '%s' isn't absolute", request.URL)
		return
	}

	// Add the base URL to the request URL:
	base, err := c.selectBaseURL(ctx, request)
	if err != nil {
		return
	}
	request.URL = base.ResolveReference(request.URL)

	// Check the request method and body:
	switch request.Method {
	case http.MethodGet, http.MethodDelete:
		if request.Body != nil {
			c.logger.Warn(ctx,
				"Request body is not allowed for the '%s' method",
				request.Method,
			)
		}
	case http.MethodPost, http.MethodPatch, http.MethodPut:
		// POST and PATCH and PUT don't need to have a body. It is up to the server to decide if
		// this is acceptable.
	default:
		err = fmt.Errorf("method '%s' is not allowed", request.Method)
		return
	}

	// Get the access token:
	token, _, err := c.TokensContext(ctx)
	if err != nil {
		err = fmt.Errorf("can't get access token: %w", err)
		return
	}

	// Add the default headers:
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	if c.agent != "" {
		request.Header.Set("User-Agent", c.agent)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	switch request.Method {
	case http.MethodPost, http.MethodPatch, http.MethodPut:
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")

	// Select the client:
	client, err := c.selectClient(ctx, base)
	if err != nil {
		return
	}

	// Send the request and get the response:
	response, err = client.Do(request)
	if err != nil {
		err = fmt.Errorf("can't send request: %w", err)
		return
	}

	// Check that the response content type is JSON:
	err = c.checkContentType(response)
	if err != nil {
		return
	}

	return
}

// checkContentType checks that the content type of the given response is JSON. Note that if the
// content type isn't JSON this method will consume the complete body in order to generate an error
// message containing a summary of the content.
func (c *Connection) checkContentType(response *http.Response) error {
	var err error
	var mediaType string
	contentType := response.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return err
		}
	} else {
		mediaType = contentType
	}
	if !strings.EqualFold(mediaType, "application/json") {
		var summary string
		summary, err = c.contentSummary(mediaType, response)
		if err != nil {
			return fmt.Errorf(
				"expected response content type 'application/json' but received "+
					"'%s' and couldn't obtain content summary: %w",
				mediaType, err,
			)
		}
		return fmt.Errorf(
			"expected response content type 'application/json' but received '%s' and "+
				"content '%s'",
			mediaType, summary,
		)
	}
	return nil
}

// contentSummary reads the body of the given response and returns a summary it. The summary will
// be the complete body if it isn't too log. If it is too long then the summary will be the
// beginning of the content followed by ellipsis.
func (c *Connection) contentSummary(mediaType string, response *http.Response) (summary string, err error) {
	var body []byte
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	limit := 250
	runes := []rune(string(body))
	if strings.EqualFold(mediaType, "text/html") && len(runes) > limit {
		content := html.UnescapeString(strip.StripTags(string(body)))
		content = wsRegex.ReplaceAllString(strings.TrimSpace(content), " ")
		runes = []rune(content)
	}
	if len(runes) > limit {
		summary = fmt.Sprintf("%s...", string(runes[:limit]))
	} else {
		summary = string(runes)
	}
	return
}

// selectBaseURL selects the base URL that should be used for the given request, according its path
// and the alternative URLs configured when the connection was created.
func (c *Connection) selectBaseURL(ctx context.Context, request *http.Request) (base *urlInfo,
	err error) {
	// Select the base URL that has the longest matching prefix. Note that it is enough to pick
	// the first match because the entries have already been sorted by descending prefix length
	// when the connection was created.
	for _, entry := range c.urlTable {
		if entry.re.MatchString(request.URL.Path) {
			base = entry.url
			return
		}
	}
	if base == nil {
		err = fmt.Errorf(
			"can't find any matching URL for request path '%s'",
			request.URL.Path,
		)
	}
	return
}

// selectClient selects an HTTP client to use to connect to the given base URL.
func (c *Connection) selectClient(ctx context.Context, base *urlInfo) (client *http.Client,
	err error) {
	// We need a client for TCP and another client for each combination of Unix and socket name,
	// so we need to calculate the key for the clients table accordingly:
	key := fmt.Sprintf("%s:%s", base.network, base.socket)

	// We will be modifiying the table of clients so we need to acquire the lock before
	// proceeding:
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()

	// Get an existing client, or create a new one if it doesn't exist yet:
	client, ok := c.clientsTable[key]
	if ok {
		return
	}
	c.logger.Debug(ctx, "Client for key '%s' doesn't exist, will create it", key)
	client, err = c.createClient(ctx, base)
	if err != nil {
		return
	}
	c.clientsTable[key] = client

	return
}

// createClient creates a new HTTP client to use to connect to the given base URL.
func (c *Connection) createClient(ctx context.Context, base *urlInfo) (client *http.Client,
	err error) {
	// Create the transport:
	transport, err := c.createTransport(ctx, base)
	if err != nil {
		return
	}

	// Create the client:
	client = &http.Client{
		Jar:       c.cookieJar,
		Transport: transport,
	}
	if c.logger.DebugEnabled() {
		client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
			c.logger.Info(
				request.Context(),
				"Following redirect from '%s' to '%s'",
				via[0].URL,
				request.URL,
			)
			return nil
		}
	}

	return
}

// createTransport creates a new HTTP transport to use to connect to the given base URL.
func (c *Connection) createTransport(ctx context.Context, base *urlInfo) (
	result http.RoundTripper, err error) {
	// Prepare the TLS configuration:
	// #nosec 402
	config := &tls.Config{
		ServerName:         base.Hostname(),
		InsecureSkipVerify: c.insecure,
		RootCAs:            c.trustedCAs,
	}

	// Create the transport:
	transport := &http.Transport{
		TLSClientConfig:   config,
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: c.disableKeepAlives,
	}

	// In order to use Unix sockets we need to explicitly set dialers that use `unix` as network
	// and the socket file as address, otherwise the HTTP client will always use `tcp` as the
	// network and the host name from the request as the address:
	if base.network == unixNetwork {
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, base.network, base.socket)
		}
		transport.DialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			// TODO: This ignores the passed context because it isn't currently
			// supported. Once we migrate to Go 1.15 it should be done like this:
			//
			//	dialer := tls.Dialer{
			//		Config: config,
			//	}
			//	return dialer.DialContext(ctx, base.network, base.socket)
			//
			// This will only have a negative impact in applications that specify a
			// deadline or timeout in the passed context, as it will be ignored.
			return tls.Dial(base.network, base.socket, config)
		}
	}

	// Prepare the result:
	result = transport

	// The metrics wrapper should be called the first because we want the corresponding round
	// tripper to be called the last so that metrics aren't affected by users specified round
	// trippers and don't include the logging overhead. Also, we don't want to add this wrapper
	// for transports used for token requests because there are specific metrics for that.
	if c.metricsWrapper != nil && base != c.tokenURL {
		result = c.metricsWrapper(result)
	}

	// The logging wrapper should be next, because we want the corresponding round tripper
	// called after the user specified round trippers so that the information in the log
	// reflects the modifications made by those suer specified round trippers.
	if c.loggingWrapper != nil {
		result = c.loggingWrapper(result)
	}

	// User transport wrappers are stored in the order that the round trippers that they create
	// should be called. That means that we need to call them in reverse order.
	for i := len(c.userWrapers) - 1; i >= 0; i-- {
		result = c.userWrapers[i](result)
	}

	return
}
