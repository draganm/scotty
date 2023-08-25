package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CurrentSelection struct {
	Namespace string
	Pod       string
	Container string
	Command   string
	Err       error
}

type step int

const (
	loadNamespacesStep = iota
	selectNamespacesStep
	loadPodsStep
	selectPodStep
	loadContainersStep
	selectContainerStep
	selectCommandStep
	doneStep
)

func (m model) setValue(v string) (model, tea.Cmd) {
	switch m.currentStep {
	case selectNamespacesStep:
		m.cs.Namespace = v
		m.currentStep = loadPodsStep
		return m, func() tea.Msg {
			nss, err := m.lister.ListPodsInNamespace(v)
			if err != nil {
				m.cs.Err = err
				return tea.Quit
			}

			return podsLoadedMsg(nss)
		}
	case selectPodStep:
		m.cs.Pod = v
		m.currentStep = loadPodsStep
		return m, func() tea.Msg {
			nss, err := m.lister.ListContainersInPod(m.cs.Namespace, v)
			if err != nil {
				m.cs.Err = err
				return tea.Quit
			}

			return containersLoadedMsg(nss)
		}
	case selectContainerStep:
		m.cs.Container = v
		m.currentStep = selectCommandStep
		return m, tea.Batch()
	case selectCommandStep:
		m.cs.Command = v
		m.currentStep = doneStep
		return m, tea.Quit
	}

	return m, nil
}

type Lister interface {
	ListNamespaces() ([]string, error)
	ListPodsInNamespace(namespace string) ([]string, error)
	ListContainersInPod(namespace, pod string) ([]string, error)
}

func createItems(lst []string) []list.Item {
	items := []list.Item{}
	for _, ns := range lst {
		items = append(items, item(ns))
	}

	return items
}

func SelectContainerAndCommand(cs *CurrentSelection, lister Lister) tea.Model {

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	lm := list.New([]list.Item{}, delegate, 0, 0)
	lm.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select"),
			),
		}
	}

	m := model{
		list:   lm,
		cs:     cs,
		lister: lister,
	}

	m.commandInput.EchoMode = textinput.EchoNormal
	m.commandInput.Prompt = "Command to run:"
	m.commandInput.SetValue("/bin/sh")

	return m

}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item string

func (i item) Title() string { return string(i) }

func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

type model struct {
	cs           *CurrentSelection
	lister       Lister
	list         list.Model
	currentStep  step
	commandInput textinput.Model
}

type namespacesLoadedMsg []string
type podsLoadedMsg []string
type containersLoadedMsg []string

func (m model) Init() tea.Cmd {

	return func() tea.Msg {
		nss, err := m.lister.ListNamespaces()
		if err != nil {
			m.cs.Err = err
			return tea.Quit
		}

		return namespacesLoadedMsg(nss)

	}

}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case namespacesLoadedMsg:
		m.currentStep = selectNamespacesStep
		m.list.Title = fmt.Sprintf("ns: %s", m.cs.Namespace)
		return m, m.list.SetItems(createItems(msg))
	case podsLoadedMsg:
		m.currentStep = selectPodStep
		m.list.Title = fmt.Sprintf("ns: %s pod: %s", m.cs.Namespace, m.cs.Pod)
		return m, m.list.SetItems(createItems(msg))
	case containersLoadedMsg:
		m.currentStep = selectContainerStep
		m.list.Title = fmt.Sprintf("ns: %s pod: %s", m.cs.Namespace, m.cs.Pod)
		return m, m.list.SetItems(createItems(msg))
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "enter" {

			if m.currentStep != selectCommandStep {
				si := m.list.SelectedItem().(item)
				return m.setValue(si.Title())
			}

			return m.setValue(m.commandInput.Value())
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	// l, cmd := m.list.Update(msg)

	if m.currentStep != selectCommandStep {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.commandInput, cmd = m.commandInput.Update(msg)
	}
	// m.list = &l

	return m, cmd
}

func (m model) View() string {
	if m.currentStep != selectCommandStep {
		return docStyle.Render(m.list.View())
	} else {
		return docStyle.Render(m.commandInput.View())
	}
}
