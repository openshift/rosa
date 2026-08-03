/*
Copyright 2026.

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

// Package transport provides an HTTP RoundTripper that signs requests with AWS SigV4.
package transport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	headerAccountID = "X-Amz-Account-Id"
	headerCallerARN = "X-Amz-Caller-Arn"
	signingService  = "execute-api"
)

// namespaceRE matches the /namespaces/{ns} segment injected by client-gen for
// namespaced resources and captures the namespace value.
// The Hyperfleet platform API is account-scoped; the namespace is rewritten to
// the X-Amz-Account-Id signed header and removed from the URL path.
var namespaceRE = regexp.MustCompile(`/namespaces/([^/]+)`)

// SigV4RoundTripper signs each request with AWS SigV4 before forwarding it.
// When the URL contains a /namespaces/{ns} segment (added by the generated
// client for namespaced CRD types), it extracts {ns} as the per-request
// account ID, rewrites the URL to remove the segment, and sets
// X-Amz-Account-Id accordingly.
type SigV4RoundTripper struct {
	inner     http.RoundTripper
	awsCfg    aws.Config
	region    string
	accountID string // default account ID used when no namespace is present
	callerARN string
}

// New returns a SigV4RoundTripper that wraps inner (defaults to
// http.DefaultTransport when nil).
func New(inner http.RoundTripper, awsCfg aws.Config, region, accountID, callerARN string) *SigV4RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &SigV4RoundTripper{
		inner:     inner,
		awsCfg:    awsCfg,
		region:    region,
		accountID: accountID,
		callerARN: callerARN,
	}
}

// RoundTrip implements http.RoundTripper. It rewrites the URL to remove the
// /namespaces/{ns} segment injected by client-gen, sets the X-Amz-Account-Id
// and X-Amz-Caller-Arn signed headers, hashes the body, signs the request with
// AWS SigV4, and forwards it via the inner transport.
func (t *SigV4RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone to avoid mutating the caller's request.
	req = req.Clone(req.Context())

	// Extract namespace from URL (if present) and rewrite path.
	// The generated client encodes the Go namespace as /namespaces/{ns}/,
	// but platform-api uses /api/v0/clusters without any namespace segment.
	accountID := t.accountID
	if m := namespaceRE.FindStringSubmatchIndex(req.URL.Path); m != nil {
		ns := req.URL.Path[m[2]:m[3]]
		if ns != "" {
			accountID = ns
		}
		req.URL.Path = req.URL.Path[:m[0]] + req.URL.Path[m[1]:]
		req.URL.RawPath = ""
	}

	// Set signed headers before calling SignHTTP so they appear in the
	// Authorization header's SignedHeaders list.
	if accountID != "" {
		req.Header.Set(headerAccountID, accountID)
	}
	if t.callerARN != "" {
		req.Header.Set(headerCallerARN, t.callerARN)
	}

	payloadHash, err := hashBody(req)
	if err != nil {
		return nil, fmt.Errorf("sigv4: hashing request body: %w", err)
	}

	creds, err := t.awsCfg.Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("sigv4: retrieving credentials: %w", err)
	}

	signer := v4.NewSigner()
	if err := signer.SignHTTP(req.Context(), creds, req, payloadHash, signingService, t.region, time.Now()); err != nil {
		return nil, fmt.Errorf("sigv4: signing request: %w", err)
	}

	return t.inner.RoundTrip(req)
}

// hashBody reads the request body into memory (so it can be re-read after
// signing), computes its SHA-256 hex digest, and restores the body on the
// request. Returns the digest of an empty body when req.Body is nil.
func hashBody(req *http.Request) (string, error) {
	var body []byte
	if req.Body != nil {
		var readErr error
		body, readErr = io.ReadAll(req.Body)
		closeErr := req.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
