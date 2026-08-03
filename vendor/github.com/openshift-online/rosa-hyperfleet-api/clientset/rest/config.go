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

package rest

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// awsRegionRE matches an AWS region name within a URL host.
// Example: abc123.execute-api.us-east-1.amazonaws.com → us-east-1
var awsRegionRE = regexp.MustCompile(`[a-z]+-[a-z]+-\d+`)

// Config holds the parameters needed to connect to the Hyperfleet platform API.
type Config struct {
	// Host is the platform API base URL (e.g. https://abc123.execute-api.us-east-1.amazonaws.com).
	// Required.
	Host string

	// Region is the AWS region used for SigV4 request signing.
	// Optional: when empty it is derived from Host for execute-api endpoints
	// (format: {id}.execute-api.{region}.amazonaws.com).
	Region string

	// AccountID is sent as the X-Amz-Account-Id signed header on every request.
	// Required.
	AccountID string

	// CallerARN is sent as the X-Amz-Caller-Arn signed header when non-empty.
	// Optional.
	CallerARN string

	// AWSConfig provides AWS credentials for SigV4 signing.
	AWSConfig aws.Config
}

// ResolveRegion returns Region if set, otherwise derives it from Host.
// Returns an error if Host is not a recognizable execute-api endpoint and Region is empty.
func (c *Config) ResolveRegion() (string, error) {
	if c.Region != "" {
		return c.Region, nil
	}
	return regionFromHost(c.Host)
}

// regionFromHost parses the AWS region from an execute-api host such as
// https://abc123.execute-api.us-east-1.amazonaws.com.
func regionFromHost(host string) (string, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("invalid host %q: %w", host, err)
	}
	region := awsRegionRE.FindString(u.Hostname())
	if region == "" {
		return "", fmt.Errorf("cannot derive region from host %q; set Config.Region explicitly", host)
	}
	return region, nil
}
