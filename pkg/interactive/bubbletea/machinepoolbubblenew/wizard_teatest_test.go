/*
Copyright (c) 2021 Red Hat, Inc.

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

package machinepoolbubblenew

import (
	"bytes"
	"testing"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/openshift/rosa/pkg/machinepooldemo"
)

func TestWizardHappyPathTeatest(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		newWizardModel(),
		teatest.WithInitialTermSize(100, 40),
	)
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	waitContains(t, tm, "Machine pool name")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("worker-pool")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Image type")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "OpenShift version")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Select subnet")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "AWS availability zone")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Subnet ID")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Enable autoscaling")
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Minimum replicas")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Maximum replicas")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Labels")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc=123")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Taints")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "security group")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Tags")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Instance type")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Autorepair")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Tuning configs")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Capacity reservation ID")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Capacity reservation preference")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Kubelet config")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "IMDSv2")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Root disk size")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Node drain grace period")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Max surge")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Max unavailable")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	finalModel := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
	wm, ok := finalModel.(wizardModel)
	if !ok {
		t.Fatalf("expected wizardModel, got %T", finalModel)
	}
	if !wm.done {
		t.Fatal("expected wizard to be done")
	}
	if wm.result.Name != "worker-pool" {
		t.Fatalf("expected name worker-pool, got %q", wm.result.Name)
	}
	if wm.result.Version != machinepooldemo.DemoClusterVersion {
		t.Fatalf("expected version %q, got %q", machinepooldemo.DemoClusterVersion, wm.result.Version)
	}
	if wm.result.Labels != "abc=123" {
		t.Fatalf("expected labels abc=123, got %q", wm.result.Labels)
	}
	if wm.result.HTTPTokens != string(cmv1.Ec2MetadataHttpTokensOptional) {
		t.Fatalf("unexpected http tokens %q", wm.result.HTTPTokens)
	}
	if len(wm.completed) != 24 {
		t.Fatalf("expected 24 completed answers, got %d", len(wm.completed))
	}
}

func waitContains(t *testing.T, tm *teatest.TestModel, text string) {
	t.Helper()
	teatest.WaitFor(
		t,
		tm.Output(),
		func(out []byte) bool {
			return bytes.Contains(out, []byte(text))
		},
		teatest.WithDuration(5*time.Second),
		teatest.WithCheckInterval(50*time.Millisecond),
	)
}
