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
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	listWidth                     = 100
	maxListItemsWithoutPagination = 25
	listTitleLines                = 3
	listHelpLines                 = 2
	listFilterReserveLines        = 1
)

func newROSAOptionList(input rosaOptionInput) list.Model {
	title, options, defaultValue := buildROSAOptionPrompt(input)
	return newOptionListFromValues(title, options, defaultValue)
}

func newOptionListFromValues(title string, options []string, defaultValue string) list.Model {
	items := make([]list.Item, len(options))
	defaultIndex := 0
	for i, opt := range options {
		items[i] = optionItem{value: opt}
		if opt == defaultValue {
			defaultIndex = i
		}
	}
	return newSelectList(title, items, defaultIndex)
}

func newSelectList(title string, items []list.Item, defaultIndex int) list.Model {
	delegate := newSurveySelectDelegate()

	itemCount := len(items)
	l := list.New(items, delegate, listWidth, listHeight(itemCount))
	l.Title = title
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowStatusBar(false)
	l.SetShowPagination(itemCount > maxListItemsWithoutPagination)
	l.SetShowHelp(true)
	applySurveyListStyles(&l)

	if defaultIndex >= 0 && defaultIndex < len(items) {
		l.Select(defaultIndex)
	}

	return l
}

func listHeight(itemCount int) int {
	visible := itemCount
	if visible > maxListItemsWithoutPagination {
		visible = maxListItemsWithoutPagination
	}
	if visible < 1 {
		visible = 1
	}
	return listTitleLines + listFilterReserveLines + visible + listHelpLines
}

// maybeStartTypeToFilter mirrors Survey select: the first printable character
// enters filter mode with that character. Navigation keys are left alone.
func maybeStartTypeToFilter(l list.Model, msg tea.Msg) (list.Model, tea.Cmd, bool) {
	if l.FilterState() != list.Unfiltered {
		return l, nil, false
	}
	if !l.FilteringEnabled() {
		return l, nil, false
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !isTypeToFilterKey(keyMsg) {
		return l, nil, false
	}

	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	var cmd tea.Cmd
	l, cmd = l.Update(slashMsg)

	charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: keyMsg.Runes}
	l, cmd2 := l.Update(charMsg)
	return l, tea.Batch(cmd, cmd2), true
}

func isTypeToFilterKey(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 {
		return false
	}
	switch msg.String() {
	case "j", "k", "h", "l", "g", "G", "f", "d", "b", "u", "q":
		return false
	}
	r := msg.Runes[0]
	return r >= 32 && r <= 126
}
