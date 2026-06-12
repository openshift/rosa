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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/openshift/rosa/pkg/color"
)

type multiSelectModel struct {
	title     string
	help      string
	options   []string
	cursor    int
	selected  map[int]struct{}
	maxSelect int
	errMsg    string
}

func newMultiSelect(title, help string, options []string, maxSelect int) multiSelectModel {
	return multiSelectModel{
		title:     title,
		help:      help,
		options:   options,
		selected:  map[int]struct{}{},
		maxSelect: maxSelect,
	}
}

func (m *multiSelectModel) Update(msg tea.Msg) (done bool, values []string) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ":
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				if m.maxSelect == 1 {
					m.selected = map[int]struct{}{m.cursor: {}}
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		case "enter":
			values = make([]string, 0, len(m.selected))
			for i := range m.options {
				if _, ok := m.selected[i]; ok {
					values = append(values, m.options[i])
				}
			}
			return true, values
		}
	}
	return false, nil
}

func (m multiSelectModel) View() string {
	var b strings.Builder
	b.WriteString(surveyPromptTitle(m.title))
	b.WriteString("\n")
	if m.help != "" {
		b.WriteString(m.help)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	for i, option := range m.options {
		_, selected := m.selected[i]
		b.WriteString(multiSelectRow(i == m.cursor, selected, option))
		b.WriteString("\n")
	}
	b.WriteString("\n↑/↓ move • space toggle • enter confirm\n")
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(m.errMsg)
		b.WriteString("\n")
	}
	return b.String()
}

func multiSelectRow(focused, selected bool, option string) string {
	check := "[ ]"
	if selected {
		check = "[x]"
	}
	cursor := " "
	if focused {
		cursor = ">"
	}

	if !color.UseColor() {
		return fmt.Sprintf("%s %s %s", cursor, check, option)
	}

	cursorStyle := lipgloss.NewStyle()
	checkStyle := lipgloss.NewStyle().Bold(true)
	if focused {
		cursorStyle = cursorStyle.Bold(true).Foreground(lipgloss.Color("6"))
	}
	if selected {
		checkStyle = checkStyle.Foreground(lipgloss.Color("2"))
	}
	return cursorStyle.Render(cursor) + " " + checkStyle.Render(check) + " " + option
}
