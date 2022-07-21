/*
Copyright (c) 2022 Red Hat, Inc.

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

// This file contains the types and functions used to manage the configuration of the command line
// client when running in FedRAMP mode.

package fedramp

import "fmt"

var regions = []string{"us-gov-west-1", "us-gov-east-1"}

func IsGovRegion(region string) bool {
	for _, r := range regions {
		if r == region {
			return true
		}
	}
	return false
}

// JumpAccounts are the various of AWS accounts used for the installer jump role in the various OCM environments
var JumpAccounts = map[string]string{
	"production":  "448648337690",
	"staging":     "448870092490",
	"integration": "449053620653",
}

// LoginURLs allows the value of the `--env` option to map to the various login URLs.
var LoginURLs = map[string]string{
	"production":  "https://api.openshiftusgov.com/auth",
	"staging":     "https://api.stage.openshiftusgov.com/auth",
	"integration": "https://api.int.openshiftusgov.com/auth",
}

// URLAliases allows the value of the `--env` option to map to the various API URLs.
var URLAliases = map[string]string{
	"production":  "https://api.openshiftusgov.com",
	"staging":     "https://api.stage.openshiftusgov.com",
	"integration": "https://api.int.openshiftusgov.com",
}

const cognitoURL = "auth-fips.us-gov-west-1.amazoncognito.com/oauth2/token"

// TokenURLs allows the value of the `--env` option to map to the various AWS Cognito token URLs.
var TokenURLs = map[string]string{
	"production":  fmt.Sprintf("https://ocm-ra-production-domain.%s", cognitoURL),
	"staging":     fmt.Sprintf("https://ocm-ra-stage-domain.%s", cognitoURL),
	"integration": fmt.Sprintf("https://rh-ocm-appsre-integration.%s", cognitoURL),
}

// ClientIDs allows the value of the `--env` option to map to the various AWS Cognito user pool clients.
var ClientIDs = map[string]string{
	"production":  "72ekjh5laouap6qcfis521jlgi",
	"staging":     "1lb687dlpsmsfuj53r3je06vpp",
	"integration": "20fbrpgl28f8oehp6709mk3nnr",
}
