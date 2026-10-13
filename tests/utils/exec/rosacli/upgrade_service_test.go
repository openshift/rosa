// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Y-stream upgrade discovery helpers", func() {
	It("selects the first next-minor upgrade from list output", func() {
		upgradeVersionList := UpgradeVersionList{
			UpgradeVersions: []UpgradeVersion{
				{Version: "4.19.14"},
				{Version: "4.19.17"},
				{Version: "4.20.0"},
				{Version: "4.20.3"},
			},
		}

		upgradingVersion, err := upgradeVersionList.FindNextMinorUpgrade("4.19.12")

		Expect(err).ToNot(HaveOccurred())
		Expect(upgradingVersion).To(Equal("4.20.0"))
	})

	It("prefers the recommended entry among same-minor candidates", func() {
		upgradeVersionList := UpgradeVersionList{
			UpgradeVersions: []UpgradeVersion{
				{Version: "4.20.0", Notes: ""},
				{Version: "4.20.3", Notes: "recommended"},
				{Version: "4.20.5", Notes: ""},
			},
		}

		upgradingVersion, err := upgradeVersionList.FindNextMinorUpgrade("4.19.12")

		Expect(err).ToNot(HaveOccurred())
		Expect(upgradingVersion).To(Equal("4.20.3"))
	})

	It("falls back to first same-minor candidate when none is recommended", func() {
		upgradeVersionList := UpgradeVersionList{
			UpgradeVersions: []UpgradeVersion{
				{Version: "4.19.14", Notes: ""},
				{Version: "4.20.0", Notes: ""},
				{Version: "4.20.3", Notes: ""},
			},
		}

		upgradingVersion, err := upgradeVersionList.FindNextMinorUpgrade("4.19.12")

		Expect(err).ToNot(HaveOccurred())
		Expect(upgradingVersion).To(Equal("4.20.0"))
	})

	It("fails when the cluster version cannot be parsed", func() {
		upgradeVersionList := UpgradeVersionList{
			UpgradeVersions: []UpgradeVersion{{Version: "4.20.0"}},
		}

		_, err := upgradeVersionList.FindNextMinorUpgrade("y-1")

		Expect(err).To(MatchError(ContainSubstring("failed to parse cluster version")))
	})

	It("fails when an upgrade version cannot be parsed", func() {
		upgradeVersionList := UpgradeVersionList{
			UpgradeVersions: []UpgradeVersion{{Version: "not-a-version"}},
		}

		_, err := upgradeVersionList.FindNextMinorUpgrade("4.19.12")

		Expect(err).To(MatchError(ContainSubstring("failed to parse upgrade version")))
	})

	It("waits until a next-minor target appears", func() {
		callCount := 0
		preparation := &YStreamUpgradePreparation{
			ClusterVersion: "4.19.12",
			CurrentChannel: "stable-4.19",
			DesiredChannel: "stable-4.20",
		}

		upgradingVersion, _, err := waitForAvailableYStreamUpgrade(
			testClusterID,
			preparation,
			time.Nanosecond,
			time.Millisecond,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				callCount++
				if callCount == 1 {
					return *bytes.NewBufferString("VERSION\n4.19.14\n"), UpgradeVersionList{
						UpgradeVersions: []UpgradeVersion{{Version: "4.19.14"}},
					}, nil
				}

				return *bytes.NewBufferString("VERSION\n4.20.0\n"), UpgradeVersionList{
					UpgradeVersions: []UpgradeVersion{{Version: "4.20.0"}},
				}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(upgradingVersion).To(Equal("4.20.0"))
		Expect(callCount).To(Equal(2))
	})

	It("uses default poll values when interval and timeout are non-positive", func() {
		callCount := 0
		preparation := &YStreamUpgradePreparation{
			ClusterVersion: "4.19.12",
			CurrentChannel: "stable-4.19",
			DesiredChannel: "stable-4.20",
		}

		upgradingVersion, _, err := waitForAvailableYStreamUpgrade(
			testClusterID,
			preparation,
			0,
			0,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				callCount++
				return *bytes.NewBufferString("VERSION\n4.20.0\n"), UpgradeVersionList{
					UpgradeVersions: []UpgradeVersion{{Version: "4.20.0"}},
				}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(upgradingVersion).To(Equal("4.20.0"))
		Expect(callCount).To(Equal(1))
	})

	It("returns a detailed timeout when no next-minor target appears", func() {
		preparation := &YStreamUpgradePreparation{
			ClusterVersion: "4.19.12",
			CurrentChannel: "",
			DesiredChannel: "stable-4.20",
		}

		_, _, err := waitForAvailableYStreamUpgrade(
			testClusterID,
			preparation,
			time.Nanosecond,
			time.Nanosecond,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				return *bytes.NewBufferString("VERSION\n4.19.14  recommended\n"), UpgradeVersionList{
					UpgradeVersions: []UpgradeVersion{{Version: "4.19.14"}},
				}, nil
			},
		)

		Expect(err).To(MatchError(ContainSubstring("timeout after")))
		Expect(err).To(MatchError(ContainSubstring("cluster 1234abcd")))
		Expect(err).To(MatchError(ContainSubstring("version=4.19.12")))
		Expect(err).To(MatchError(ContainSubstring(`desired channel="stable-4.20"`)))
		Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
		Expect(err).To(MatchError(ContainSubstring("4.19.14")))
	})

	It("includes the last fetch error in the timeout details", func() {
		preparation := &YStreamUpgradePreparation{
			ClusterVersion: "4.19.12",
			CurrentChannel: "stable-4.19",
			DesiredChannel: "stable-4.20",
		}

		_, _, err := waitForAvailableYStreamUpgrade(
			testClusterID,
			preparation,
			time.Nanosecond,
			time.Nanosecond,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				return bytes.Buffer{}, UpgradeVersionList{}, fmt.Errorf("boom")
			},
		)

		Expect(err).To(MatchError(ContainSubstring("last error: boom")))
	})

	It("fails when cluster ID is empty", func() {
		_, _, err := waitForAvailableYStreamUpgrade(
			"",
			&YStreamUpgradePreparation{ClusterVersion: "4.19.12"},
			time.Millisecond,
			time.Millisecond,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				return bytes.Buffer{}, UpgradeVersionList{}, nil
			},
		)

		Expect(err).To(MatchError(ContainSubstring("cluster ID is required")))
	})

	It("fails when preparation is nil", func() {
		_, _, err := waitForAvailableYStreamUpgrade(
			testClusterID,
			nil,
			time.Millisecond,
			time.Millisecond,
			func() (bytes.Buffer, UpgradeVersionList, error) {
				return bytes.Buffer{}, UpgradeVersionList{}, nil
			},
		)

		Expect(err).To(MatchError(ContainSubstring("y-stream upgrade preparation is required")))
	})
})

var _ = Describe("Upgrade scheduling retry helpers", func() {
	updateInProgressErr := fmt.Errorf("exit status 1: ERR: failed to schedule upgrade for cluster '%s': "+
		"status is 400, identifier is '400', code is 'CLUSTERS-MGMT-400': "+
		"An update is already in progress and the details are in the Progressing condition", testClusterID)

	It("returns immediately when the upgrade is scheduled on the first attempt", func() {
		callCount := 0

		output, err := upgradeRetryingWhileUpdateInProgress(
			time.Nanosecond,
			time.Second,
			func() (bytes.Buffer, error) {
				callCount++
				return *bytes.NewBufferString("Upgrade successfully scheduled for cluster"), nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))
		Expect(callCount).To(Equal(1))
	})

	It("retries while the cluster update is still in progress", func() {
		callCount := 0

		output, err := upgradeRetryingWhileUpdateInProgress(
			time.Nanosecond,
			time.Second,
			func() (bytes.Buffer, error) {
				callCount++
				if callCount < 3 {
					return *bytes.NewBufferString(updateInProgressErr.Error()), updateInProgressErr
				}
				return *bytes.NewBufferString("Upgrade successfully scheduled for cluster"), nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))
		Expect(callCount).To(Equal(3))
	})

	It("detects the in-progress message in the command output", func() {
		callCount := 0

		_, err := upgradeRetryingWhileUpdateInProgress(
			time.Nanosecond,
			time.Second,
			func() (bytes.Buffer, error) {
				callCount++
				if callCount == 1 {
					return *bytes.NewBufferString(updateInProgressErr.Error()), fmt.Errorf("exit status 1")
				}
				return bytes.Buffer{}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(callCount).To(Equal(2))
	})

	It("does not retry unrelated errors", func() {
		callCount := 0

		output, err := upgradeRetryingWhileUpdateInProgress(
			time.Nanosecond,
			time.Second,
			func() (bytes.Buffer, error) {
				callCount++
				return *bytes.NewBufferString("There is already a scheduled upgrade"), fmt.Errorf("boom")
			},
		)

		Expect(err).To(MatchError("boom"))
		Expect(output.String()).To(ContainSubstring("There is already a scheduled upgrade"))
		Expect(callCount).To(Equal(1))
	})

	It("uses the default retry timeout when interval and timeout are non-positive", func() {
		_, err := upgradeRetryingWhileUpdateInProgress(
			0,
			0,
			func() (bytes.Buffer, error) {
				return *bytes.NewBufferString(updateInProgressErr.Error()), updateInProgressErr
			},
		)

		Expect(err).To(MatchError(ContainSubstring(
			fmt.Sprintf("timeout after %s", defaultUpdateInProgressRetryTimeout))))
	})

	It("returns a detailed timeout that wraps the last error when the cluster update never finishes", func() {
		_, err := upgradeRetryingWhileUpdateInProgress(
			time.Millisecond,
			5*time.Millisecond,
			func() (bytes.Buffer, error) {
				return *bytes.NewBufferString(updateInProgressErr.Error()), updateInProgressErr
			},
		)

		Expect(err).To(MatchError(ContainSubstring("timeout after 5ms")))
		Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
		Expect(err).To(MatchError(ContainSubstring(updateInProgressMessage)))
		Expect(errors.Is(err, updateInProgressErr)).To(BeTrue())
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
	})
})
