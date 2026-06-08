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

package bubbletea

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// RunConfirm asks a yes/no question using Bubble Tea.
func RunConfirm(message string, defaultYes bool) (bool, error) {
	if !isInteractiveTerminal(os.Stdout) {
		return defaultYes, nil
	}

	model := NewConfirmModel(message, defaultYes)
	final, err := runProgram(model)
	if err != nil {
		return false, err
	}
	confirmed, aborted, ok := ReadConfirmOutcome(final)
	if !ok {
		return false, fmt.Errorf("unexpected confirm result")
	}
	if aborted {
		return false, fmt.Errorf("confirmation cancelled")
	}
	return confirmed, nil
}

// NewConfirmModel builds a yes/no confirmation Bubble Tea model.
func NewConfirmModel(message string, defaultYes bool) tea.Model {
	return confirmModel{
		message:    message,
		defaultYes: defaultYes,
	}
}

// ReadConfirmOutcome returns the outcome of a confirm model after it has quit.
func ReadConfirmOutcome(m tea.Model) (confirmed, aborted, ok bool) {
	cm, ok := m.(confirmModel)
	if !ok {
		return false, false, false
	}
	return cm.confirmed, cm.aborted, true
}

type confirmModel struct {
	message    string
	defaultYes bool
	confirmed  bool
	aborted    bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
			return m, tea.Quit
		case "n", "N":
			m.confirmed = false
			return m, tea.Quit
		case "enter":
			m.confirmed = m.defaultYes
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	defaultLabel := "No"
	if m.defaultYes {
		defaultLabel = "Yes"
	}
	return fmt.Sprintf("%s (%s) [y/N]: ", m.message, defaultLabel)
}

func isInteractiveTerminal(out io.Writer) bool {
	if file, ok := out.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return true
		}
		return info.Mode()&os.ModeCharDevice != 0
	}
	return false
}

func runProgram(model tea.Model) (tea.Model, error) {
	program := tea.NewProgram(model, tea.WithOutput(os.Stdout))
	return program.Run()
}
