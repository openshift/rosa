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
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const checkboxHelp = "space toggle · enter confirm · / filter"

type checkboxItem struct {
	value string
}

func (i checkboxItem) Title() string       { return i.value }
func (i checkboxItem) Description() string { return "" }
func (i checkboxItem) FilterValue() string { return i.value }

// checkboxDelegate renders list rows with checkboxes using the native bubbles/list
// ItemDelegate API. Selection state is shared via selected map pointer.
type checkboxDelegate struct {
	styles   list.DefaultItemStyles
	selected *map[string]struct{}
}

func newCheckboxDelegate(selected *map[string]struct{}) checkboxDelegate {
	styles := list.NewDefaultItemStyles()
	styles.SelectedTitle = styles.SelectedTitle.Foreground(accentColor)
	styles.SelectedDesc = styles.SelectedDesc.Foreground(accentColor)
	return checkboxDelegate{styles: styles, selected: selected}
}

func (d checkboxDelegate) Height() int  { return 1 }
func (d checkboxDelegate) Spacing() int { return 0 }

func (d checkboxDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d checkboxDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	title := item.(checkboxItem).value
	if title == "" || m.Width() <= 0 {
		return
	}

	textWidth := m.Width() - 6
	if textWidth < 1 {
		textWidth = m.Width()
	}
	title = ansi.Truncate(title, textWidth, "…")

	_, checked := (*d.selected)[item.(checkboxItem).value]
	marker := uncheckedMarker()
	if checked {
		marker = checkedMarker()
	}

	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFocused := index == m.Index() && !emptyFilter

	style := d.styles.NormalTitle
	if isFocused && m.FilterState() != list.Filtering {
		style = d.styles.SelectedTitle
	} else if emptyFilter {
		style = d.styles.DimmedTitle
	}

	line := marker + " " + style.Render(title)
	fmt.Fprint(w, line)
}

func checkedMarker() string {
	return lipgloss.NewStyle().Foreground(okColor).Bold(true).Render("✓")
}

func uncheckedMarker() string {
	return lipgloss.NewStyle().Foreground(mutedColor).Render("○")
}

type checkboxList struct {
	list      list.Model
	selected  map[string]struct{}
	allValues []string
	maxSelect int
	errMsg    string
	optional  bool
}

func newCheckboxList(title string, options []string, maxSelect int, optional bool) checkboxList {
	selected := map[string]struct{}{}
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = checkboxItem{value: opt}
	}

	delegate := newCheckboxDelegate(&selected)
	itemCount := len(items)
	l := list.New(items, delegate, listWidth, listHeight(itemCount))
	l.Title = title
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowStatusBar(true)
	l.SetShowPagination(itemCount > maxListItemsWithoutPagination)
	l.SetShowHelp(true)
	applyListStyles(&l)

	return checkboxList{
		list:      l,
		selected:  selected,
		allValues: append([]string{}, options...),
		maxSelect: maxSelect,
		optional:  optional,
	}
}

func (c *checkboxList) toggleFocused() {
	if c.list.FilterState() == list.Filtering {
		return
	}
	item, ok := c.list.SelectedItem().(checkboxItem)
	if !ok {
		return
	}
	if _, ok := c.selected[item.value]; ok {
		delete(c.selected, item.value)
		return
	}
	if c.maxSelect == 1 {
		for key := range c.selected {
			delete(c.selected, key)
		}
		c.selected[item.value] = struct{}{}
		return
	}
	c.selected[item.value] = struct{}{}
}

func (c *checkboxList) selectedValues() []string {
	values := make([]string, 0, len(c.selected))
	for _, value := range c.allValues {
		if _, ok := c.selected[value]; ok {
			values = append(values, value)
		}
	}
	return values
}

func (c *checkboxList) Update(msg tea.Msg) (done bool, values []string, cmd tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case " ":
			c.toggleFocused()
			c.errMsg = ""
			return false, nil, nil
		case "enter":
			if c.list.FilterState() == list.Filtering {
				break
			}
			return true, c.selectedValues(), nil
		}
	}

	var updateCmd tea.Cmd
	c.list, updateCmd = c.list.Update(msg)
	return false, nil, updateCmd
}

func (c checkboxList) View() string {
	var b strings.Builder
	if c.optional {
		b.WriteString(renderTextPrompt(c.list.Title, true, ""))
		b.WriteString("\n\n")
	}
	b.WriteString(c.list.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(checkboxHelp))
	if c.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(c.errMsg))
	}
	return b.String()
}
