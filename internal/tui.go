package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var blueColor = "#A4D6F4"
var yellowColor = "#F0D7AE"
var grayColor = "#737373"

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(blueColor))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(grayColor))
	cursorStyle  = focusedStyle.Copy()
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle.Copy()

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type model struct {
	config   Config
	cursor   int  // which to-do list item our cursor is pointing at
	selected *int // which to-do items are selected

	// new
	inputs     []textinput.Model
	focusIndex int
}

var filePath string

func checkConfig() tea.Msg {
	root, _ := os.UserHomeDir()
	filePath = fmt.Sprintf("%s/.config/sqlal/config.json", root)
	_, err := os.Stat(filePath)

	if err != nil {
		config := NewDefaultConfig()
		config.SaveToFile(filePath)

		return config
	}

	config := Config{}
	config.LoadFromFile(filePath)

	return config
}

func InitialModel() model {
	m := model{
		// Our to-do list is a grocery list
		selected: nil,
		inputs:   make([]textinput.Model, 3),
	}

	m.SetInputs()

	return m
}

func (m *model) SetInputs() {
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 512

		switch i {
		case 0:
			t.Placeholder = "Name"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "SQL"
		case 2:
			t.Placeholder = "URL"
		}

		m.inputs[i] = t
	}
}

func (m model) Init() tea.Cmd {
	return checkConfig
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Config:
		m.config = Config(msg)
		return m, nil
	case tea.KeyMsg:
		if m.selected != nil {
			return m.handleInputsKey(msg)
		} else {
			return m.handleConfigMenuKey(msg)
		}
	}

	return m, nil
}

func (m *model) handleInputsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.selected = nil
		return m, nil
	case "tab", "shift+tab", "enter", "up", "down":
		return m.navigateInputs(msg.String())
	}
	return m, m.updateInputs(msg) // Return the updated model and the commands generated by inputs
}

func (m model) handleConfigMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "x":
		return m.deleteQuery()
	case "up", "k":
		return m.moveCursorUp()
	case "down", "j":
		return m.moveCursorDown()
	case "enter", " ":
		return m.toggleSelected()
	}
	return m, nil
}

func (m model) navigateInputs(s string) (tea.Model, tea.Cmd) {
	if s == "enter" && m.focusIndex == len(m.inputs) {
		if m.selected != nil && *m.selected > 0 {
			// Update query
			queryIndex := *m.selected - 1
			newQuery := QueryConfig{
				Name:            m.inputs[0].Value(),
				Query:           m.inputs[1].Value(),
				NotificationURL: m.inputs[2].Value(),
			}
			m.config.UpdateQuery(queryIndex, newQuery)
			m.config.SaveToFile(filePath)
			m.SetInputs()
		} else {
			newQuery := QueryConfig{
				Name:            m.inputs[0].Value(),
				Query:           m.inputs[1].Value(),
				NotificationURL: m.inputs[2].Value(),
			}
			m.config.AddQuery(newQuery)
			m.config.SaveToFile(filePath)
			m.SetInputs()
		}

		m.selected = nil
		m.focusIndex = 0

		return m, nil
	}

	// Cycle indexes
	if s == "up" || s == "shift+tab" {
		m.focusIndex--
	} else {
		m.focusIndex++
	}

	if m.focusIndex > len(m.inputs) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs)
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i <= len(m.inputs)-1; i++ {
		if i == m.focusIndex {
			// Set focused state
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
			continue
		}
		// Remove focused state
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = noStyle
		m.inputs[i].TextStyle = noStyle
	}

	return m, tea.Batch(cmds...)
}

func (m model) deleteQuery() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.config.DeleteQueryByIndex(m.cursor - 1)
		m.config.SaveToFile(filePath)

		m.cursor = m.cursor - 1
	}

	return m, nil
}

func (m model) moveCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	}

	return m, nil
}

func (m model) moveCursorDown() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.config.GetQueryNames())-1 {
		m.cursor++
	}

	return m, nil
}

func (m *model) toggleSelected() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		m.selected = &m.cursor
	} else {
		selectedIndex := m.cursor - 1
		m.selected = &m.cursor

		if selectedIndex >= 0 && selectedIndex < len(m.config.Queries) {
			query := m.config.Queries[selectedIndex]
			for i, input := range m.inputs {
				switch i {
				case 0:
					input.SetValue(query.Name)
				case 1:
					input.SetValue(query.Query)
				case 2:
					input.SetValue(query.NotificationURL)
				}
				m.inputs[i] = input
			}
		}
	}
	return m, nil
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	if m.shouldShowInputsView() {
		return m.renderInputsView()
	} else {
		return m.renderConfigMenuView()
	}
}

func (m model) shouldShowInputsView() bool {
	return m.selected != nil
}

func (m model) renderInputsView() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n%s\n", *button, helpStyle.Render("[enter] - submit; [esc] - go back;"))

	return b.String()
}

func (m model) renderConfigMenuView() string {
	s := lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color("#A4F4D7")).Render("SQL Alerts configuration menu") + "\n\n"

	for i, choice := range m.config.GetQueryNames() {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			if i == 0 {
				choice = lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color(blueColor)).Render(choice)
			} else {
				choice = lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color(yellowColor)).Render(choice)
			}
		}
		s += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color(blueColor)).Render(cursor), choice)
	}

	if m.cursor == 0 {
		s += helpStyle.Render("\n\n[enter] - open; [q] - quit;\n")
	} else {
		s += helpStyle.Render("\n\n[enter] - edit; [x] - delete; [q] - quit;\n")
	}

	return s
}
