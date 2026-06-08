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
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/bubbletea"
	rosauserrole "github.com/openshift/rosa/pkg/userrole"
)

func TestUserRoleWizardFlow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User role Bubble Tea wizard flow")
}

var _ = Describe("User role Bubble Tea interactive flow", func() {
	defaultWizardInput := func() WizardInput {
		return WizardInput{
			Prefix:                  aws.DefaultPrefix,
			PrefixHelp:              "User-defined prefix for ocm-user role",
			PermissionsBoundaryHelp: "Permissions boundary help",
			PathHelp:                "Role path help",
			ModeHelp:                "Role creation mode help",
		}
	}

	advanceToPermissionsBoundary := func(m wizardModel) wizardModel {
		m.text.SetValue("ManagedOpenShift")
		return pressEnter(m)
	}

	advanceToPath := func(m wizardModel) wizardModel {
		m = advanceToPermissionsBoundary(m)
		Expect(m.step).To(Equal(stepPermissionsBoundary))
		return pressEnter(m)
	}

	advanceToMode := func(m wizardModel) wizardModel {
		m = advanceToPath(m)
		Expect(m.step).To(Equal(stepPath))
		return pressEnter(m)
	}

	// 1. Happy path step order: prefix → permissions boundary → path → mode.
	It("walks the user through prompts in the expected order", func() {
		m := newWizardModel(defaultWizardInput())

		Expect(m.step).To(Equal(stepPrefix))
		Expect(m.View()).To(ContainSubstring("Role prefix"))
		Expect(m.View()).To(ContainSubstring("User-defined prefix for ocm-user role"))

		m = advanceToPermissionsBoundary(m)
		Expect(m.step).To(Equal(stepPermissionsBoundary))
		Expect(m.result.Prefix).To(Equal("ManagedOpenShift"))
		Expect(m.View()).To(ContainSubstring("Permissions boundary ARN"))
		Expect(m.View()).To(ContainSubstring("Permissions boundary help"))

		m = pressEnter(m)
		Expect(m.step).To(Equal(stepPath))
		Expect(m.result.PermissionsBoundary).To(BeEmpty())
		Expect(m.View()).To(ContainSubstring("Role Path"))

		m = pressEnter(m)
		Expect(m.step).To(Equal(stepMode))
		Expect(m.result.Path).To(BeEmpty())
		Expect(m.View()).To(ContainSubstring("Role creation mode help"))

		m = pressEnter(m)
		Expect(m.done).To(BeTrue())
		Expect(m.result.Mode).To(Equal(interactive.ModeAuto))
		Expect(m.result).To(Equal(userroleInput(
			"ManagedOpenShift",
			"",
			"",
			interactive.ModeAuto,
		)))
	})

	// 2. Required prefix empty → error, stay on prefix.
	It("rejects an empty role prefix and keeps the user on that step", func() {
		m := newWizardModel(WizardInput{PrefixHelp: "prefix help"})
		Expect(m.step).To(Equal(stepPrefix))

		m = pressEnter(m)

		Expect(m.step).To(Equal(stepPrefix))
		Expect(m.errMsg).To(Equal("role prefix is required"))
		Expect(m.View()).To(ContainSubstring("role prefix is required"))
	})

	// 3. Invalid permissions boundary ARN → error, stay on boundary.
	It("rejects an invalid permissions boundary ARN and keeps the user on that step", func() {
		m := advanceToPermissionsBoundary(newWizardModel(defaultWizardInput()))
		m.text.SetValue("not-an-arn")

		m = pressEnter(m)

		Expect(m.step).To(Equal(stepPermissionsBoundary))
		Expect(m.errMsg).To(ContainSubstring("expected a valid policy ARN for permissions boundary"))
		Expect(m.result.PermissionsBoundary).To(BeEmpty())
	})

	// 4. Valid empty permissions boundary → advance to path.
	It("accepts an empty optional permissions boundary and advances to role path", func() {
		m := advanceToPermissionsBoundary(newWizardModel(defaultWizardInput()))
		m.text.SetValue("")

		m = pressEnter(m)

		Expect(m.step).To(Equal(stepPath))
		Expect(m.errMsg).To(BeEmpty())
		Expect(m.result.PermissionsBoundary).To(BeEmpty())
		Expect(m.View()).To(ContainSubstring("Role Path"))
	})

	// 5. Mode branching: manual vs auto selection.
	Describe("mode branching", func() {
		It("records auto mode when auto stays selected", func() {
			m := advanceToMode(newWizardModel(defaultWizardInput()))
			Expect(m.step).To(Equal(stepMode))

			m = pressEnter(m)

			Expect(m.done).To(BeTrue())
			Expect(m.result.Mode).To(Equal(interactive.ModeAuto))
		})

		It("records manual mode when the user selects manual", func() {
			m := advanceToMode(newWizardModel(defaultWizardInput()))
			m = pressDown(m)
			m = pressEnter(m)

			Expect(m.done).To(BeTrue())
			Expect(m.result.Mode).To(Equal(interactive.ModeManual))
		})
	})

	// 6. Confirm step behavior for auto mode role creation (cmd layer uses this after the wizard).
	// The -y flag skips this prompt in the command; these tests cover the Bubble Tea confirm model.
	Describe("confirm step behavior", func() {
		It("accepts yes with y", func() {
			m := bubbletea.NewConfirmModel("Create the 'ManagedOpenShift-User-jdoe-Role' role?", true)
			m = pressConfirmKey(m, "y")

			confirmed, aborted, ok := bubbletea.ReadConfirmOutcome(m)
			Expect(ok).To(BeTrue())
			Expect(aborted).To(BeFalse())
			Expect(confirmed).To(BeTrue())
		})

		It("declines with n", func() {
			m := bubbletea.NewConfirmModel("Create the 'ManagedOpenShift-User-jdoe-Role' role?", true)
			m = pressConfirmKey(m, "n")

			confirmed, aborted, ok := bubbletea.ReadConfirmOutcome(m)
			Expect(ok).To(BeTrue())
			Expect(aborted).To(BeFalse())
			Expect(confirmed).To(BeFalse())
		})

		It("uses the default on enter when default is yes", func() {
			m := bubbletea.NewConfirmModel("Create the 'ManagedOpenShift-User-jdoe-Role' role?", true)
			m = pressConfirmKey(m, "enter")

			confirmed, aborted, ok := bubbletea.ReadConfirmOutcome(m)
			Expect(ok).To(BeTrue())
			Expect(aborted).To(BeFalse())
			Expect(confirmed).To(BeTrue())
		})

		It("uses the default on enter when default is no", func() {
			m := bubbletea.NewConfirmModel("Create the 'ManagedOpenShift-User-jdoe-Role' role?", false)
			m = pressConfirmKey(m, "enter")

			confirmed, aborted, ok := bubbletea.ReadConfirmOutcome(m)
			Expect(ok).To(BeTrue())
			Expect(aborted).To(BeFalse())
			Expect(confirmed).To(BeFalse())
		})

		It("treats escape as cancellation", func() {
			m := bubbletea.NewConfirmModel("Create the 'ManagedOpenShift-User-jdoe-Role' role?", true)
			m = pressConfirmKey(m, "esc")

			confirmed, aborted, ok := bubbletea.ReadConfirmOutcome(m)
			Expect(ok).To(BeTrue())
			Expect(aborted).To(BeTrue())
			Expect(confirmed).To(BeFalse())
		})
	})
})

func userroleInput(prefix, boundary, path, mode string) rosauserrole.Input {
	return rosauserrole.Input{
		Prefix:              prefix,
		PermissionsBoundary: boundary,
		Path:                path,
		Mode:                mode,
	}
}

func pressEnter(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(wizardModel)
}

func pressDown(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	return next.(wizardModel)
}

func pressConfirmKey(m tea.Model, key string) tea.Model {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEscape}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	return next
}
