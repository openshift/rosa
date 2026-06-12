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

package machinepoolbubble

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/openshift/rosa/pkg/color"
)

// optionalQuestionLabel mirrors interactive.GetString/GetMultipleOptions when
// Required is false and there is no default.
func optionalQuestionLabel(question string) string {
	return question + " (optional)"
}

// surveyPromptTitle formats an active prompt title like Survey (green+hb "?",
// bold question). Used for text-input and multi-select steps.
func surveyPromptTitle(title string) string {
	if !color.UseColor() {
		return "? " + title
	}

	questionMark := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("?")
	titleText := lipgloss.NewStyle().Bold(true).Render(title)
	return questionMark + " " + titleText
}

// surveyTranscriptLine formats a completed prompt like Survey (green+hb "?",
// bold question, cyan answer). Matches survey/v2 default IconSet.Question.Format.
func surveyTranscriptLine(question, answer string) string {
	if !color.UseColor() {
		return fmt.Sprintf("? %s: %s", question, answer)
	}

	questionMark := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("?")
	questionText := lipgloss.NewStyle().Bold(true).Render(question + ":")
	if answer == "" {
		return questionMark + " " + questionText
	}
	answerText := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(answer)
	return questionMark + " " + questionText + " " + answerText
}

func (m *wizardModel) recordAnswer(question, answer string) tea.Cmd {
	m.completed = append(m.completed, completedAnswer{label: question, value: answer})
	return tea.Println(surveyTranscriptLine(question, answer))
}

func mergeCmds(cmds ...tea.Cmd) tea.Cmd {
	nonNil := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			nonNil = append(nonNil, cmd)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	return tea.Batch(nonNil...)
}

func wizardResult(final tea.Model) (wizardModel, bool) {
	model, ok := final.(*wizardModel)
	if !ok {
		return wizardModel{}, false
	}
	return *model, true
}
