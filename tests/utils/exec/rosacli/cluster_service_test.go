package rosacli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/utils/constants"
)

const testClusterID = "1234abcd"

var _ = Describe("Y-stream channel helpers", func() {
	It("computes the next-minor channel from the profile channel group", func() {
		channel, err := computeNextMinorChannel("stable", "4.19.12")

		Expect(err).ToNot(HaveOccurred())
		Expect(channel).To(Equal("stable-4.20"))
	})

	It("fails when the cluster version cannot be parsed", func() {
		_, err := computeNextMinorChannel("stable", "y-1")

		Expect(err).To(MatchError(ContainSubstring("failed to parse cluster version")))
	})

	It("fails when the channel group is empty", func() {
		_, err := computeNextMinorChannel("", "4.19.12")

		Expect(err).To(MatchError(ContainSubstring("channel group is required")))
	})

	It("updates the channel when the current channel is empty", func() {
		var capturedClusterID string
		var capturedFlags []string

		preparation, err := prepareYStreamUpgradeChannel(
			testClusterID,
			"",
			"4.19.12",
			"stable",
			func(clusterID string, flags ...string) (bytes.Buffer, error) {
				capturedClusterID = clusterID
				capturedFlags = append([]string{}, flags...)
				return bytes.Buffer{}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(preparation.ClusterVersion).To(Equal("4.19.12"))
		Expect(preparation.CurrentChannel).To(Equal("stable-4.20"))
		Expect(preparation.DesiredChannel).To(Equal("stable-4.20"))
		Expect(capturedClusterID).To(Equal(testClusterID))
		Expect(capturedFlags).To(Equal([]string{"--channel", "stable-4.20", "-y"}))
	})

	It("does not update the channel when the desired channel is already set", func() {
		edited := false

		preparation, err := prepareYStreamUpgradeChannel(
			testClusterID,
			"stable-4.20",
			"4.19.12",
			"stable",
			func(clusterID string, flags ...string) (bytes.Buffer, error) {
				edited = true
				return bytes.Buffer{}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(preparation.DesiredChannel).To(Equal("stable-4.20"))
		Expect(edited).To(BeFalse())
	})

	It("fails when it needs to edit but no edit function is provided", func() {
		_, err := prepareYStreamUpgradeChannel(
			testClusterID,
			"",
			"4.19.12",
			"stable",
			nil,
		)

		Expect(err).To(MatchError(ContainSubstring("edit cluster function is required")))
	})

	It("wraps the editCluster error when channel update fails", func() {
		_, err := prepareYStreamUpgradeChannel(
			testClusterID,
			"stable-4.19",
			"4.19.12",
			"stable",
			func(clusterID string, flags ...string) (bytes.Buffer, error) {
				return bytes.Buffer{}, fmt.Errorf("boom")
			},
		)

		Expect(err).To(MatchError(ContainSubstring("failed to update cluster 1234abcd channel")))
		Expect(err).To(MatchError(ContainSubstring("boom")))
	})
})

var _ = Describe("Cluster wait helpers", func() {
	const clusterID = testClusterID

	It("fails fast when waiting without a cluster ID", func() {
		service := &clusterService{}

		err := service.WaitClusterStatus("", constants.Ready, 1, 1)

		Expect(err).To(MatchError(ContainSubstring("cluster ID is required")))
	})

	It("waits for ready status using describe output", func() {
		tempDir, err := os.MkdirTemp("", "fake-rosa-*")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tempDir)

		textCountFile := filepath.Join(tempDir, "describe-count")
		jsonCountFile := filepath.Join(tempDir, "json-describe-count")
		scriptPath := filepath.Join(tempDir, "rosa")
		script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
text_count_file=%q
json_count_file=%q

if [[ "$1" == "describe" && "$2" == "cluster" ]]; then
  if [[ "$*" == *"--output json"* ]]; then
    count=0
    if [[ -f "$json_count_file" ]]; then
      count=$(<"$json_count_file")
    fi
    count=$((count + 1))
    printf '%%s' "$count" > "$json_count_file"
    printf '{"state":"waiting"}\n'
    exit 0
  fi

  count=0
  if [[ -f "$text_count_file" ]]; then
    count=$(<"$text_count_file")
  fi
  count=$((count + 1))
  printf '%%s' "$count" > "$text_count_file"

  if [[ "$count" -eq 1 ]]; then
    cat <<'EOF'
Name: fake
ID: 1234abcd
State: resuming
EOF
    else
    cat <<'EOF'
Name: fake
ID: 1234abcd
State: ready
EOF
    fi
    exit 0
fi

echo "unexpected args: $*" >&2
exit 1
`, textCountFile, jsonCountFile)
		err = os.WriteFile(scriptPath, []byte(script), 0755)
		Expect(err).ToNot(HaveOccurred())

		originalPath := os.Getenv("PATH")
		Expect(os.Setenv("PATH", tempDir+":"+originalPath)).To(Succeed())
		DeferCleanup(os.Setenv, "PATH", originalPath)

		client := NewClient()
		client.Runner.envs = prependPathEnv(tempDir)
		service := NewClusterService(client)

		err = service.WaitClusterStatus(clusterID, constants.Ready, 0, 0)

		Expect(err).ToNot(HaveOccurred())

		jsonCalls := "0"
		if content, readErr := os.ReadFile(jsonCountFile); readErr == nil {
			jsonCalls = string(content)
		}
		Expect(jsonCalls).To(Equal("0"))
	})

	It("keeps waiting while the cluster is still in waiting state", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State: constants.Waiting,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(ready).To(BeFalse())
	})

	It("returns an error when the cluster description is missing", func() {
		ready, err := evaluateReadyClusterState(clusterID, nil)

		Expect(err).To(MatchError(ContainSubstring("no cluster description found")))
		Expect(ready).To(BeFalse())
	})

	It("returns an error when the cluster state is empty", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State: "   ",
		})

		Expect(err).To(MatchError(ContainSubstring("returned an empty state")))
		Expect(ready).To(BeFalse())
	})

	It("returns a detailed error when the cluster enters an error state", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State:                    constants.Error,
			ProvisioningErrorCode:    "AccessDenied",
			ProvisioningErrorMessage: "Missing operator role permissions",
			FailedInflightChecks:     "operator role validation failed",
		})

		Expect(err).To(MatchError(ContainSubstring("cluster 1234abcd is in error state")))
		Expect(err).To(MatchError(ContainSubstring("AccessDenied")))
		Expect(err).To(MatchError(ContainSubstring("Missing operator role permissions")))
		Expect(err).To(MatchError(ContainSubstring("operator role validation failed")))
		Expect(ready).To(BeFalse())
	})

	It("returns a basic error when the cluster enters an error state without provisioning details", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State: constants.Error,
		})

		Expect(err).To(MatchError(ContainSubstring("cluster 1234abcd is in error state")))
		Expect(ready).To(BeFalse())
	})

	It("returns an uninstalling error while waiting for ready", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State: constants.Uninstalling,
		})

		Expect(err).To(MatchError(ContainSubstring("Cannot wait for it ready")))
		Expect(ready).To(BeFalse())
	})

	It("keeps waiting while the cluster is resuming", func() {
		ready, err := evaluateReadyClusterState(clusterID, &ClusterDescription{
			State: "resuming",
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(ready).To(BeFalse())
	})

	It("waits through transient states until the cluster becomes ready", func() {
		states := []string{constants.Waiting, constants.Ready}
		stateIndex := 0

		err := waitForClusterReadyStatus(
			clusterID,
			time.Nanosecond,
			time.Millisecond,
			func() (*ClusterDescription, error) {
				current := states[stateIndex]
				if stateIndex < len(states)-1 {
					stateIndex++
				}
				return &ClusterDescription{State: current}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
	})

	It("checks the first ready state immediately before waiting", func() {
		err := waitForClusterReadyStatus(
			clusterID,
			time.Hour,
			time.Millisecond,
			func() (*ClusterDescription, error) {
				return &ClusterDescription{State: constants.Ready}, nil
			},
		)

		Expect(err).ToNot(HaveOccurred())
	})

	It("returns an explicit not-found error while waiting for ready", func() {
		err := waitForClusterReadyStatus(
			clusterID,
			time.Nanosecond,
			time.Millisecond,
			func() (*ClusterDescription, error) {
				return nil, fmt.Errorf("Cluster '%s' not found", clusterID)
			},
		)

		Expect(err).To(MatchError(ContainSubstring("cluster 1234abcd not found while waiting for it to become ready")))
	})

	It("recognizes the alternate not-found error wording", func() {
		err := waitForClusterReadyStatus(
			clusterID,
			time.Nanosecond,
			time.Millisecond,
			func() (*ClusterDescription, error) {
				return nil, fmt.Errorf("There is no cluster with identifier or name '%s'", clusterID)
			},
		)

		Expect(err).To(MatchError(ContainSubstring("cluster 1234abcd not found while waiting for it to become ready")))
	})

	It("includes the last known state in timeout errors", func() {
		err := formatReadyClusterTimeout(clusterID, 60*time.Minute, &ClusterDescription{
			State: constants.Installing,
		})

		Expect(err).To(MatchError(ContainSubstring("timeout for cluster ready waiting after 60 mins")))
		Expect(err).To(MatchError(ContainSubstring("last state: installing")))
	})

	It("handles timeout errors without a last description", func() {
		err := formatReadyClusterTimeout(clusterID, time.Minute, nil)

		Expect(err).To(MatchError(ContainSubstring("timeout for cluster ready waiting after 1 mins")))
	})

	It("resets runner format after JSON describe errors", func() {
		tempDir, err := os.MkdirTemp("", "fake-rosa-error-*")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tempDir)

		scriptPath := filepath.Join(tempDir, "rosa")
		script := `#!/usr/bin/env bash
echo "boom" >&2
exit 1
`
		err = os.WriteFile(scriptPath, []byte(script), 0755)
		Expect(err).ToNot(HaveOccurred())

		originalPath := os.Getenv("PATH")
		Expect(os.Setenv("PATH", tempDir+":"+originalPath)).To(Succeed())
		DeferCleanup(os.Setenv, "PATH", originalPath)

		client := NewClient()
		client.Runner.envs = prependPathEnv(tempDir)
		service := NewClusterService(client)

		_, err = service.GetJSONClusterDescription(clusterID)

		Expect(err).To(HaveOccurred())
		Expect(client.Runner.runnerCfg.format).To(Equal(defaultRunnerFormat))
	})
})

func prependPathEnv(prefix string) []string {
	envs := os.Environ()
	for index, env := range envs {
		if strings.HasPrefix(env, "PATH=") {
			envs[index] = "PATH=" + prefix + ":" + strings.TrimPrefix(env, "PATH=")
			return envs
		}
	}

	return append(envs, "PATH="+prefix)
}
