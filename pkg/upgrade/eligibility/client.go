/*
Copyright (c) 2026 Red Hat, Inc.

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

package eligibility

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/ocm"
)

type OCMClient interface {
	GetLimitedSupportReasons(clusterID string) ([]*cmv1.LimitedSupportReason, error)
	GetScheduledUpgrade(clusterID string) (*cmv1.UpgradePolicy, *cmv1.UpgradePolicyState, error)
	GetUpgradePolicies(clusterID string) ([]*cmv1.UpgradePolicy, error)
	GetAvailableUpgrades(versionID string) ([]string, error)
	GetAvailableUpgradesWithProduct(product string, versionID string) ([]string, error)
	GetRawAvailableUpgrades(versionID string) ([]string, error)
	ListAllOcpGates(version string) ([]*cmv1.VersionGate, error)
	ListGateAgreements(clusterID string) ([]*cmv1.VersionGateAgreement, error)
}

type ocmClientAdapter struct {
	client *ocm.Client
}

func NewOCMClient(client *ocm.Client) OCMClient {
	return &ocmClientAdapter{client: client}
}

func (a *ocmClientAdapter) GetLimitedSupportReasons(clusterID string) ([]*cmv1.LimitedSupportReason, error) {
	return a.client.GetLimitedSupportReasons(clusterID)
}

func (a *ocmClientAdapter) GetScheduledUpgrade(clusterID string) (*cmv1.UpgradePolicy, *cmv1.UpgradePolicyState, error) {
	return a.client.GetScheduledUpgrade(clusterID)
}

func (a *ocmClientAdapter) GetUpgradePolicies(clusterID string) ([]*cmv1.UpgradePolicy, error) {
	return a.client.GetUpgradePolicies(clusterID)
}

func (a *ocmClientAdapter) GetAvailableUpgrades(versionID string) ([]string, error) {
	return a.client.GetAvailableUpgrades(versionID)
}

func (a *ocmClientAdapter) GetAvailableUpgradesWithProduct(product string, versionID string) ([]string, error) {
	return a.client.GetAvailableUpgradesWithProduct(product, versionID)
}

func (a *ocmClientAdapter) GetRawAvailableUpgrades(versionID string) ([]string, error) {
	return a.client.GetRawAvailableUpgrades(versionID)
}

func (a *ocmClientAdapter) ListAllOcpGates(version string) ([]*cmv1.VersionGate, error) {
	return a.client.ListAllOcpGates(version)
}

func (a *ocmClientAdapter) ListGateAgreements(clusterID string) ([]*cmv1.VersionGateAgreement, error) {
	return a.client.ListGateAgreements(clusterID)
}
