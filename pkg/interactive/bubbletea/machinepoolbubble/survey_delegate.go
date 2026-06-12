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
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const listEllipsis = "…"

// surveySelectDelegate renders list rows like Survey select: cyan bold ">" on the
// focused row and two spaces on others. Native bubbles/list ItemDelegate API.
type surveySelectDelegate struct {
	styles          list.DefaultItemStyles
	showDescription bool
	height          int
	spacing         int
}

func newSurveySelectDelegate() surveySelectDelegate {
	styles := list.NewDefaultItemStyles()
	styles.NormalTitle = lipgloss.NewStyle()
	styles.NormalDesc = lipgloss.NewStyle()
	styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))
	styles.SelectedDesc = styles.SelectedTitle
	styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})
	styles.DimmedDesc = styles.DimmedTitle
	styles.FilterMatch = lipgloss.NewStyle().Underline(true)

	return surveySelectDelegate{
		styles:          styles,
		showDescription: false,
		height:          1,
		spacing:         0,
	}
}

func (d surveySelectDelegate) Height() int {
	if d.showDescription {
		return d.height
	}
	return 1
}

func (d surveySelectDelegate) Spacing() int {
	return d.spacing
}

func (d surveySelectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d surveySelectDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	title := itemTitle(item)
	if title == "" || m.Width() <= 0 {
		return
	}

	textWidth := m.Width() - 2
	if textWidth < 1 {
		textWidth = m.Width()
	}
	title = ansi.Truncate(title, textWidth, listEllipsis)

	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	isSelected := index == m.Index() && !emptyFilter

	var matchedRunes []int
	if isFiltered && m.FilterValue() != "" {
		matchedRunes = m.MatchesForItem(index)
	}

	prefix := "  "
	style := d.styles.NormalTitle
	if isSelected && m.FilterState() != list.Filtering {
		prefix = "> "
		style = d.styles.SelectedTitle
	} else if emptyFilter {
		style = d.styles.DimmedTitle
	}

	line := prefix + title
	if isFiltered && len(matchedRunes) > 0 && !emptyFilter {
		unmatched := style.Inline(true)
		matched := unmatched.Inherit(d.styles.FilterMatch)
		offset := len(prefix)
		adjusted := make([]int, len(matchedRunes))
		for i, r := range matchedRunes {
			adjusted[i] = r + offset
		}
		line = lipgloss.StyleRunes(line, adjusted, matched, unmatched)
	} else {
		line = style.Render(line)
	}

	fmt.Fprint(w, line)
}

func itemTitle(item list.Item) string {
	switch i := item.(type) {
	case list.DefaultItem:
		return i.Title()
	case optionItem:
		return i.Title()
	case boolItem:
		return i.Title()
	default:
		return ""
	}
}

func applySurveyListStyles(l *list.Model) {
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.Styles.HelpStyle = lipgloss.NewStyle().Padding(1, 0, 0, 0)
}

func listPromptLabel(title string) string {
	return strings.TrimSuffix(title, ":")
}
