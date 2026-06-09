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

package userrole

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
)

// Happy-path integration test using teatest: runs the full Bubble Tea program,
// sends key input, and asserts the final model after the wizard completes.
func TestWizardHappyPathTeatest(t *testing.T) {
	input := WizardInput{
		Prefix:                  aws.DefaultPrefix,
		PrefixHelp:              "User-defined prefix for ocm-user role",
		PermissionsBoundaryHelp: "Permissions boundary help",
		PathHelp:                "Role path help",
		ModeHelp:                "Role creation mode help",
	}

	tm := teatest.NewTestModel(
		t,
		newWizardModel(input),
		teatest.WithInitialTermSize(100, 40),
	)
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	waitContains(t, tm, "Role prefix")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Permissions boundary ARN")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Role Path")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitContains(t, tm, "Role creation mode")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	finalModel := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
	wm, ok := finalModel.(wizardModel)
	if !ok {
		t.Fatalf("expected wizardModel, got %T", finalModel)
	}
	if !wm.done {
		t.Fatal("expected wizard to be done")
	}
	if wm.result.Prefix != aws.DefaultPrefix {
		t.Fatalf("expected prefix %q, got %q", aws.DefaultPrefix, wm.result.Prefix)
	}
	if wm.result.PermissionsBoundary != "" {
		t.Fatalf("expected empty permissions boundary, got %q", wm.result.PermissionsBoundary)
	}
	if wm.result.Path != "" {
		t.Fatalf("expected empty path, got %q", wm.result.Path)
	}
	if wm.result.Mode != interactive.ModeAuto {
		t.Fatalf("expected mode %q, got %q", interactive.ModeAuto, wm.result.Mode)
	}
	if len(wm.completed) != 4 {
		t.Fatalf("expected 4 completed answers, got %d", len(wm.completed))
	}
	if wm.completed[0].label != "Role prefix" || wm.completed[0].value != aws.DefaultPrefix {
		t.Fatalf("unexpected first completed answer: %+v", wm.completed[0])
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
