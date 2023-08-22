package itemselector

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
)

func SelectItem(title string, lst []string, onSelect func(s string), size ssh.Window) tea.Model {

	items := []list.Item{}
	for _, ns := range lst {
		items = append(items, item(ns))
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	// docStyle.

	w, h := docStyle.GetFrameSize()

	m := model{list: list.New(items, delegate, size.Width-w, size.Height-h), vals: lst, onSelect: onSelect}
	m.list.Title = title
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select"),
			),
		}
	}

	return m

}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item string

func (i item) Title() string { return string(i) }

func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

type model struct {
	list     list.Model
	vals     []string
	onSelect func(s string)
}

func (m model) Init() tea.Cmd {

	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// fmt.Println(msg)
	switch msg := msg.(type) {
	// case error:
	// 	m.list.ShowTitle()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "enter" {
			m.onSelect(m.vals[m.list.Index()])
			return m, tea.Quit
		}
		fmt.Println(msg.String())
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}
