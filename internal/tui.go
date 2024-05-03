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

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(blueColor))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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

func (m model) SetInputs() {
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

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
	// Just return `nil`, which means "no I/O right now, please."
	return checkConfig
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Config:
		m.config = Config(msg)
		return m, nil
	// Is it a key press?
	case tea.KeyMsg:
		if m.selected != nil && *m.selected == 0 {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.selected = nil
				return m, nil

			// Set focus to next input
			case "tab", "shift+tab", "enter", "up", "down":
				s := msg.String()

				if s == "enter" && m.focusIndex == len(m.inputs) {
					m.selected = nil
					newQuery := QueryConfig{
						Name:            m.inputs[0].Value(),
						Query:           m.inputs[1].Value(),
						NotificationURL: m.inputs[2].Value(),
					}
					m.config.AddQuery(newQuery)
					m.config.SaveToFile(filePath)
					m.SetInputs()
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
		} else {
			// Cool, what was the actual key pressed?
			switch msg.String() {

			// These keys should exit the program.
			case "ctrl+c", "q":
				return m, tea.Quit

			case "x":
				if m.cursor > 0 {
					m.config.DeleteQueryByIndex(m.cursor - 1)
					m.config.SaveToFile(filePath)

					m.cursor = m.cursor - 1
				}
			// The "up" and "k" keys move the cursor up
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			// The "down" and "j" keys move the cursor down
			case "down", "j":
				if m.cursor < len(m.config.GetQueryNames())-1 {
					m.cursor++
				}

			// The "enter" key and the spacebar (a literal space) toggle
			// the selected state for the item that the cursor is pointing at.
			case "enter", " ":
				m.selected = &m.cursor
			}
		}
	}

	if m.selected != nil && *m.selected == 0 {
		cmd := m.updateInputs(msg)

		return m, cmd
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
	// The header
	s := ""
	if m.selected != nil && *m.selected == 0 {
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
	} else {
		s = lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color("#A4F4D7")).Render("SQL Alerts configuration menu") + "\n\n"

		for i, choice := range m.config.GetQueryNames() {

			cursor := " "
			if m.cursor == i {
				cursor = ">"
				choice = lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color(blueColor)).Render(choice)
			}

			s += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Blink(true).Foreground(lipgloss.Color(blueColor)).Render(cursor), choice)
		}

		if m.cursor == 0 {
			s += helpStyle.Render("\n\n[enter] - open; [q] - quit;\n")
		} else {
			s += helpStyle.Render("\n\n[enter] - edit; [x] - delete; [q] - quit;\n")
		}
	}

	return s
}
