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
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	userrole "github.com/openshift/rosa/pkg/userrole"
)

// WizardInput contains defaults and help text for the Bubble Tea wizard.
type WizardInput struct {
	Prefix                  string
	PermissionsBoundary     string
	Path                    string
	Mode                    string
	PrefixHelp              string
	PermissionsBoundaryHelp string
	PathHelp                string
	ModeHelp                string
}

type wizardStep int

const (
	stepPrefix wizardStep = iota
	stepPermissionsBoundary
	stepPath
	stepMode
)

type modeItem struct {
	value string
}

func (i modeItem) Title() string       { return i.value }
func (i modeItem) Description() string { return "" }
func (i modeItem) FilterValue() string { return i.value }

type completedAnswer struct {
	label string
	value string
}

type wizardModel struct {
	step      wizardStep
	input     WizardInput
	result    userrole.Input
	text      textinput.Model
	list      list.Model
	completed []completedAnswer
	errMsg    string
	done      bool
	aborted   bool
}

// RunWizard collects user role settings using Bubble Tea prompts.
func RunWizard(input WizardInput) (userrole.Input, error) {
	if !isTerminal(os.Stdout) {
		result := userrole.Input{
			Prefix:              input.Prefix,
			PermissionsBoundary: input.PermissionsBoundary,
			Path:                input.Path,
			Mode:                input.Mode,
		}
		if result.Mode == "" {
			result.Mode = interactive.ModeAuto
		}
		if err := userrole.Validate(result); err != nil {
			return userrole.Input{}, err
		}
		return result, nil
	}

	model := newWizardModel(input)
	final, err := tea.NewProgram(model, tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return userrole.Input{}, err
	}
	resultModel, ok := final.(wizardModel)
	if !ok {
		return userrole.Input{}, fmt.Errorf("unexpected wizard result")
	}
	if resultModel.aborted {
		return userrole.Input{}, fmt.Errorf("interactive input cancelled")
	}
	if err := userrole.Validate(resultModel.result); err != nil {
		return userrole.Input{}, err
	}
	return resultModel.result, nil
}

func newWizardModel(input WizardInput) wizardModel {
	text := textinput.New()
	text.Placeholder = input.Prefix
	text.Focus()
	text.CharLimit = 32
	text.SetValue(input.Prefix)

	items := []list.Item{
		modeItem{value: interactive.ModeAuto},
		modeItem{value: interactive.ModeManual},
	}
	modeList := list.New(items, list.NewDefaultDelegate(), 20, 4)
	modeList.Title = "Role creation mode"
	modeList.SetShowHelp(true)
	if input.Mode != "" {
		for i, item := range items {
			if item.(modeItem).value == input.Mode {
				modeList.Select(i)
				break
			}
		}
	}

	return wizardModel{
		step:   stepPrefix,
		input:  input,
		text:   text,
		list:   modeList,
		result: userrole.Input{Mode: input.Mode},
	}
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		}
	}

	switch m.step {
	case stepPrefix, stepPermissionsBoundary, stepPath:
		var cmd tea.Cmd
		m.text, cmd = m.text.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
			value := m.text.Value()
			switch m.step {
			case stepPrefix:
				if value == "" {
					m.errMsg = "role prefix is required"
					return m, cmd
				}
				if err := validatePrefix(value); err != nil {
					m.errMsg = err.Error()
					return m, cmd
				}
				m.result.Prefix = value
				m.completed = append(m.completed, completedAnswer{
					label: "Role prefix",
					value: value,
				})
				m.step = stepPermissionsBoundary
				m.errMsg = ""
				m.text.SetValue(m.input.PermissionsBoundary)
				m.text.Placeholder = "optional"
				m.text.CharLimit = 2048
			case stepPermissionsBoundary:
				if value != "" {
					if err := aws.ARNValidator(value); err != nil {
						m.errMsg = fmt.Sprintf("expected a valid policy ARN for permissions boundary: %s", err)
						return m, cmd
					}
				}
				m.result.PermissionsBoundary = value
				m.completed = append(m.completed, completedAnswer{
					label: "Permissions boundary ARN",
					value: value,
				})
				m.step = stepPath
				m.errMsg = ""
				m.text.SetValue(m.input.Path)
				m.text.Placeholder = "optional"
				m.text.CharLimit = 512
			case stepPath:
				if value != "" && !aws.ARNPath.MatchString(value) {
					m.errMsg = "the specified value for path is invalid. " +
						"It must begin and end with '/' and contain only alphanumeric characters and/or '/' characters."
					return m, cmd
				}
				m.result.Path = value
				m.completed = append(m.completed, completedAnswer{
					label: "Role Path",
					value: value,
				})
				m.step = stepMode
				m.errMsg = ""
			}
		}
		return m, cmd
	case stepMode:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
			selected, ok := m.list.SelectedItem().(modeItem)
			if !ok {
				m.errMsg = "expected a valid role creation mode"
				return m, cmd
			}
			m.result.Mode = selected.value
			m.completed = append(m.completed, completedAnswer{
				label: "Role creation mode",
				value: selected.value,
			})
			m.done = true
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m wizardModel) View() string {
	if m.done {
		return ""
	}

	var current string
	switch m.step {
	case stepPrefix:
		current = renderTextStep("Role prefix", m.input.PrefixHelp, m.text.View(), m.errMsg)
	case stepPermissionsBoundary:
		current = renderTextStep("Permissions boundary ARN", m.input.PermissionsBoundaryHelp, m.text.View(), m.errMsg)
	case stepPath:
		current = renderTextStep("Role Path", m.input.PathHelp, m.text.View(), m.errMsg)
	case stepMode:
		help := m.input.ModeHelp
		if help == "" {
			help = "How to perform the operation"
		}
		view := m.list.View()
		if m.errMsg != "" {
			current = fmt.Sprintf("%s\n\n%s\n", view, m.errMsg)
		} else {
			current = fmt.Sprintf("%s\n\n%s\n", help, view)
		}
	default:
		return ""
	}

	return renderCompletedSummary(m.completed) + current
}

func renderCompletedSummary(completed []completedAnswer) string {
	if len(completed) == 0 {
		return ""
	}

	var b strings.Builder
	for _, answer := range completed {
		fmt.Fprintf(&b, "%s: %s\n", answer.label, answer.value)
	}
	b.WriteString("\n")
	return b.String()
}

func renderTextStep(title, help, inputView, errMsg string) string {
	view := fmt.Sprintf("%s\n", title)
	if help != "" {
		view += fmt.Sprintf("%s\n", help)
	}
	view += fmt.Sprintf("\n%s\n", inputView)
	if errMsg != "" {
		view += fmt.Sprintf("\n%s\n", errMsg)
	}
	return view
}

func validatePrefix(prefix string) error {
	if len(prefix) > 32 {
		return fmt.Errorf("expected a prefix with no more than 32 characters")
	}
	re := regexp.MustCompile(`[\w+=,.@-]+`)
	if !re.MatchString(prefix) {
		return fmt.Errorf("%s does not match regular expression %s", prefix, re.String())
	}
	if !aws.RoleNameRE.MatchString(prefix) {
		return fmt.Errorf("expected a valid role prefix matching %s", aws.RoleNameRE.String())
	}
	return nil
}

func isTerminal(out *os.File) bool {
	info, err := out.Stat()
	if err != nil {
		return true
	}
	return info.Mode()&os.ModeCharDevice != 0
}
