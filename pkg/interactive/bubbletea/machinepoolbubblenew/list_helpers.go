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
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

const (
	listWidth                     = 80
	maxListItemsWithoutPagination = 10
	listTitleLines                = 2
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
	delegate := list.NewDefaultDelegate()
	styles := list.NewDefaultItemStyles()
	styles.SelectedTitle = styles.SelectedTitle.Foreground(accentColor)
	styles.SelectedDesc = styles.SelectedDesc.Foreground(accentColor)
	delegate.Styles = styles
	delegate.SetHeight(1)
	delegate.SetSpacing(0)

	itemCount := len(items)
	l := list.New(items, delegate, listWidth, listHeight(itemCount))
	l.Title = title
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowStatusBar(true)
	l.SetShowPagination(itemCount > maxListItemsWithoutPagination)
	l.SetShowHelp(true)
	applyListStyles(&l)

	if defaultIndex >= 0 && defaultIndex < len(items) {
		l.Select(defaultIndex)
	}

	return l
}

func applyListStyles(l *list.Model) {
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(mutedColor)
	l.Styles.StatusBar = lipgloss.NewStyle().Foreground(mutedColor)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(accentColor)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(accentColor)
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
