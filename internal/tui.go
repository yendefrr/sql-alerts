package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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
	config     Config
	cursor     int
	selected   *int
	topButtons []string

	inputsSettings []textinput.Model
	inputsDB       []textinput.Model
	inputsQuery    []textinput.Model
	inputTextQuery textarea.Model
	focusIndex     int
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
		selected:       nil,
		focusIndex:     0,
		topButtons:     []string{"Configure settings", "Configure database", "Create new query\n"},
		inputsSettings: make([]textinput.Model, 3),
		inputsDB:       make([]textinput.Model, 5),
		inputsQuery:    make([]textinput.Model, 3),
	}

	m.SetInputs()

	return m
}

func (m *model) SetInputs() {
	var t textinput.Model
	for i := range m.inputsQuery {
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
			t.Placeholder = "URL"
		case 2:
			t.Placeholder = "Disabled (y/n)"
		}

		m.inputsQuery[i] = t
	}
	ti := textarea.New()
	ti.Placeholder = "SQL"
	ti.CharLimit = 0
	m.inputTextQuery = ti

	for i := range m.inputsDB {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 512

		switch i {
		case 0:
			t.Placeholder = "username"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		case 2:
			t.Placeholder = "host"
		case 3:
			t.Placeholder = "port"
		case 4:
			t.Placeholder = "name"
		}

		m.inputsDB[i] = t
	}

	for i := range m.inputsSettings {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 512

		switch i {
		case 0:
			t.Placeholder = "Base notification URL (https://ntfy.sh/)"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Notification message (must inclide one %d for rows number including)"
		case 2:
			t.Placeholder = "Check interval in seconds"
		}

		m.inputsSettings[i] = t
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
		m.focusIndex = 0

		return m, nil
	case "tab", "shift+tab":
		if *m.selected == 0 {
			return m.navigateInputsSettings(msg.String())
		}
		if *m.selected == 1 {
			return m.navigateInputsDB(msg.String())
		}
		if *m.selected > 1 {
			return m.navigateInputsQuery(msg.String())
		}
	case "up", "down", "enter":
		if *m.selected == 0 {
			return m.navigateInputsSettings(msg.String())
		}
		if *m.selected == 1 {
			return m.navigateInputsDB(msg.String())
		}
		if *m.selected > 1 && m.focusIndex != len(m.inputsQuery) {
			return m.navigateInputsQuery(msg.String())
		}
	}

	if *m.selected == 0 {
		return m, m.updateInputsSettings(msg)
	}
	if *m.selected == 1 {
		return m, m.updateInputsDB(msg)
	}
	if *m.selected > 1 {
		return m, m.updateInputsQuery(msg)
	}

	return m, nil
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

func (m model) navigateInputsQuery(s string) (tea.Model, tea.Cmd) {
	if s == "enter" && m.focusIndex == len(m.inputsQuery)+1 {
		var disabled bool
		if m.inputsQuery[2].Value() == "y" {
			disabled = true
		}
		if m.inputsQuery[2].Value() == "n" {
			disabled = false
		}
		if m.selected != nil && *m.selected >= len(m.topButtons) {
			queryIndex := *m.selected - len(m.topButtons)

			newQuery := QueryConfig{
				Name:            m.inputsQuery[0].Value(),
				NotificationURL: m.inputsQuery[1].Value(),
				Query:           m.inputTextQuery.Value(),
				Disabled:        disabled,
			}
			m.config.UpdateQuery(queryIndex, newQuery)
			m.config.SaveToFile(filePath)
			m.SetInputs()
		} else {
			newQuery := QueryConfig{
				Name:            m.inputsQuery[0].Value(),
				NotificationURL: m.inputsQuery[1].Value(),
				Query:           m.inputTextQuery.Value(),
				Disabled:        disabled,
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

	if m.focusIndex > len(m.inputsQuery)+1 {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputsQuery)
	}

	cmds := make([]tea.Cmd, len(m.inputsQuery))
	for i := 0; i <= len(m.inputsQuery)-1; i++ {
		if i == m.focusIndex {
			// Set focused state
			cmds[i] = m.inputsQuery[i].Focus()
			m.inputsQuery[i].PromptStyle = focusedStyle
			m.inputsQuery[i].TextStyle = focusedStyle
			continue
		}
		// Remove focused state
		m.inputsQuery[i].Blur()
		m.inputsQuery[i].PromptStyle = noStyle
		m.inputsQuery[i].TextStyle = noStyle
	}
	if m.focusIndex == len(m.inputsQuery) {
		cmds = append(cmds, m.inputTextQuery.Focus())
	} else {
		m.inputTextQuery.Blur()
	}

	return m, tea.Batch(cmds...)
}

func (m model) navigateInputsDB(s string) (tea.Model, tea.Cmd) {
	if s == "enter" && m.focusIndex == len(m.inputsDB) {
		newDB := DatabaseConfig{
			Username: m.inputsDB[0].Value(),
			Password: m.inputsDB[1].Value(),
			Host:     m.inputsDB[2].Value(),
			Port:     m.inputsDB[3].Value(),
			Name:     m.inputsDB[4].Value(),
		}
		m.config.UpdateDB(newDB)
		m.config.SaveToFile(filePath)
		m.SetInputs()

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

	if m.focusIndex > len(m.inputsDB) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputsDB)
	}

	cmds := make([]tea.Cmd, len(m.inputsDB))
	for i := 0; i <= len(m.inputsDB)-1; i++ {
		if i == m.focusIndex {
			// Set focused state
			cmds[i] = m.inputsDB[i].Focus()
			m.inputsDB[i].PromptStyle = focusedStyle
			m.inputsDB[i].TextStyle = focusedStyle
			continue
		}
		// Remove focused state
		m.inputsDB[i].Blur()
		m.inputsDB[i].PromptStyle = noStyle
		m.inputsDB[i].TextStyle = noStyle
	}

	return m, tea.Batch(cmds...)
}

func (m model) navigateInputsSettings(s string) (tea.Model, tea.Cmd) {
	if s == "enter" && m.focusIndex == len(m.inputsSettings) {
		seconds, _ := strconv.Atoi(m.inputsSettings[2].Value())

		newSettings := Config{
			Database:             m.config.Database,
			Queries:              m.config.Queries,
			BaseNotificationURL:  m.inputsSettings[0].Value(),
			NotificationMessage:  m.inputsSettings[1].Value(),
			CheckIntervalSeconds: seconds,
		}
		m.config.UpdateSettings(&newSettings)
		m.config.SaveToFile(filePath)
		m.SetInputs()

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

	if m.focusIndex > len(m.inputsSettings) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputsSettings)
	}

	cmds := make([]tea.Cmd, len(m.inputsSettings))
	for i := 0; i <= len(m.inputsSettings)-1; i++ {
		if i == m.focusIndex {
			// Set focused state
			cmds[i] = m.inputsSettings[i].Focus()
			m.inputsSettings[i].PromptStyle = focusedStyle
			m.inputsSettings[i].TextStyle = focusedStyle
			continue
		}
		// Remove focused state
		m.inputsSettings[i].Blur()
		m.inputsSettings[i].PromptStyle = noStyle
		m.inputsSettings[i].TextStyle = noStyle
	}

	return m, tea.Batch(cmds...)
}

func (m model) deleteQuery() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.topButtons) {
		m.config.DeleteQueryByIndex(m.cursor - len(m.topButtons))
		m.config.SaveToFile(filePath)

		m.cursor = m.cursor - 1
	}

	return m, nil
}

func (m model) moveCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	} else {
		m.cursor = len(m.config.GetQueryNames()) + len(m.topButtons) - 1
	}

	return m, nil
}

func (m model) moveCursorDown() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.config.GetQueryNames())+len(m.topButtons)-1 {
		m.cursor++
	} else {
		m.cursor = 0
	}

	return m, nil
}

func (m *model) toggleSelected() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		m.selected = &m.cursor
		setInputsFromSettings(&m.config, m.inputsSettings)
	case 1:
		m.selected = &m.cursor
		setInputsFromDB(&m.config, m.inputsDB)
	case 2:
		m.selected = &m.cursor
		clearInputs(m.inputsQuery, &m.inputTextQuery)
	default:
		m.selected = &m.cursor
		setInputsFromQuery(&m.config, m.inputsQuery, &m.inputTextQuery, m.cursor, len(m.topButtons))
	}
	return m, nil
}

func clearInputs(inputs []textinput.Model, textInput *textarea.Model) {
	for i := range inputs {
		inputs[i].SetValue("")
	}
	textInput.SetValue("")
}

func setInputsFromQuery(config *Config, inputs []textinput.Model, inputText *textarea.Model, cursor, topButtonLen int) {
	selectedIndex := cursor - topButtonLen
	if selectedIndex >= 0 && selectedIndex < len(config.Queries)+topButtonLen {
		query := config.Queries[selectedIndex]
		for i, input := range inputs {
			switch i {
			case 0:
				input.SetValue(query.Name)
			case 1:
				input.SetValue(query.NotificationURL)
			case 2:
				if query.Disabled {
					input.SetValue("y")
				} else {
					input.SetValue("n")
				}
			}

			inputs[i] = input
		}
		inputText.SetValue(query.Query)
	}
}

func setInputsFromDB(config *Config, inputs []textinput.Model) {
	for i, input := range inputs {
		switch i {
		case 0:
			input.SetValue(config.Database.Username)
		case 1:
			input.SetValue(config.Database.Password)
		case 2:
			input.SetValue(config.Database.Host)
		case 3:
			input.SetValue(config.Database.Port)
		case 4:
			input.SetValue(config.Database.Name)
		}
		inputs[i] = input
	}
}

func setInputsFromSettings(config *Config, inputs []textinput.Model) {
	for i, input := range inputs {
		switch i {
		case 0:
			input.SetValue(config.BaseNotificationURL)
		case 1:
			input.SetValue(config.NotificationMessage)
		case 2:
			input.SetValue(strconv.Itoa(config.CheckIntervalSeconds))
		}
		inputs[i] = input
	}
}

func (m *model) updateInputsQuery(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputsQuery))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputsQuery {
		m.inputsQuery[i], cmds[i] = m.inputsQuery[i].Update(msg)
	}

	var cmd tea.Cmd
	m.inputTextQuery, cmd = m.inputTextQuery.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (m *model) updateInputsDB(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputsDB))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputsDB {
		m.inputsDB[i], cmds[i] = m.inputsDB[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m *model) updateInputsSettings(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputsSettings))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputsSettings {
		m.inputsSettings[i], cmds[i] = m.inputsSettings[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	if m.shouldShowInputsView() {
		return m.renderInputsView(m.inputsQuery, &m.inputTextQuery)
	} else if m.shouldShowDBView() {
		return m.renderInputsView(m.inputsDB, nil)
	} else if m.shouldShowSettingsView() {
		return m.renderInputsView(m.inputsSettings, nil)
	} else {
		return m.renderConfigMenuView()
	}
}

func (m model) shouldShowInputsView() bool {
	return m.selected != nil && *m.selected > 1
}

func (m model) shouldShowDBView() bool {
	return m.selected != nil && *m.selected == 1
}

func (m model) shouldShowSettingsView() bool {
	return m.selected != nil && *m.selected == 0
}

func (m model) renderInputsView(inputs []textinput.Model, inputText *textarea.Model) string {
	var b strings.Builder

	renderInputs(&b, inputs, inputText)

	inputsLen := len(inputs)
	if inputText != nil {
		inputsLen += 1
	}
	renderButton(&b, m.focusIndex == inputsLen)

	return b.String()
}

func renderInputs(b *strings.Builder, inputs []textinput.Model, inputText *textarea.Model) {
	for _, input := range inputs {
		b.WriteString(input.View())
		b.WriteRune('\n')
	}
	if inputText != nil {
		b.WriteString(inputText.View())
	}
}

func renderButton(b *strings.Builder, isFocused bool) {
	button := &blurredButton
	if isFocused {
		button = &focusedButton
	}
	fmt.Fprintf(b, "\n\n%s\n\n%s\n", *button, helpStyle.Render("[enter] - submit; [tab] - unfocus textarea; [esc] - go back;"))
}

func (m model) renderConfigMenuView() string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color("#A4F4D7")).Render("SQL Alerts configuration menu") + "\n\n"

	var cursor string
	for i, button := range m.topButtons {
		cursor = " "
		if m.cursor == i {
			cursor = ">"
			lipgloss.NewStyle().Foreground(lipgloss.Color(blueColor)).Render(button)
		}
		s += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color(blueColor)).Render(cursor), button)
	}

	for i, choice := range m.config.GetQueryNames() {
		cursor = " "
		if m.cursor == i+len(m.topButtons) {
			cursor = ">"
			choice = lipgloss.NewStyle().Foreground(lipgloss.Color(yellowColor)).Render(choice)
		}
		s += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color(blueColor)).Render(cursor), choice)
	}

	if m.cursor < len(m.topButtons) {
		s += helpStyle.Render("\n\n[enter] - open; [q] - quit;\n")
	} else {
		s += helpStyle.Render("\n\n[enter] - edit; [x] - delete; [q] - quit;\n")
	}

	return s
}
