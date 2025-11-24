package ui

import (
	"fmt"
	"strings"

	"github.com/AnvithLobo/EvilNFSClient/pkg/nfs"
	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TUIModel represents the state of the terminal user interface
type TUIModel struct {
	Client         *nfs.NFSClient
	textInput      textinput.Model
	viewport       viewport.Model
	err            error
	quitting       bool
	commandHistory []string
	historyIndex   int
	Output         []string
	maxOutputLines int
}

// InitialModel creates and returns a new TUI model
func InitialModel(client *nfs.NFSClient) TUIModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80

	vp := viewport.New(80, 20)

	return TUIModel{
		Client:         client,
		textInput:      ti,
		viewport:       vp,
		commandHistory: []string{},
		historyIndex:   -1,
		Output:         []string{},
		maxOutputLines: 500,
	}
}

// Init initializes the model
func (m TUIModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 7 // Leave room for header and input
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyPgUp:
			m.viewport.PageUp()
			return m, nil
		case tea.KeyPgDown:
			m.viewport.PageDown()
			return m, nil
		case tea.KeyEnter:
			command := m.textInput.Value()
			m.textInput.Reset()
			if command == "exit" || command == "quit" {
				m.quitting = true
				return m, tea.Quit
			}
			if command != "" {
				m.commandHistory = append(m.commandHistory, command)
				m.historyIndex = len(m.commandHistory)
			}

			// Buffer command and output for display in TUI
			m.Output = append(m.Output, styles.PromptStyle.Render("nfs> ")+command)

			// Execute command
			output := m.Client.ExecuteCommand(command)
			m.Output = append(m.Output, output...)

			// Keep only last maxOutputLines
			if len(m.Output) > m.maxOutputLines {
				m.Output = m.Output[len(m.Output)-m.maxOutputLines:]
			}

			// Update viewport content and scroll to bottom
			m.viewport.SetContent(strings.Join(m.Output, "\n"))
			m.viewport.GotoBottom()

			return m, nil
		case tea.KeyUp:
			if len(m.commandHistory) > 0 {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.textInput.SetValue(m.commandHistory[m.historyIndex])
				}
			}
			return m, nil
		case tea.KeyDown:
			if len(m.commandHistory) > 0 {
				if m.historyIndex < len(m.commandHistory)-1 {
					m.historyIndex++
					m.textInput.SetValue(m.commandHistory[m.historyIndex])
				} else if m.historyIndex == len(m.commandHistory)-1 {
					m.historyIndex++
					m.textInput.SetValue("")
				}
			}
			return m, nil
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the TUI
func (m TUIModel) View() string {
	if m.quitting {
		return ""
	}

	s := styles.TitleStyle.Render("🔥 EvilNFSClient") + "\n"
	s += lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Connected to %s:%s (UID: %d, GID: %d) | Path: %s",
		m.Client.Server, m.Client.Export, m.Client.UID, m.Client.GID, m.Client.CurrentPath)) + "\n"

	// Render scrollable viewport with output
	s += m.viewport.View() + "\n"

	s += styles.PromptStyle.Render("nfs> ") + m.textInput.View() + "\n"
	s += lipgloss.NewStyle().Faint(true).Render("(PgUp/PgDn: scroll | ↑↓: history | Ctrl+C: quit)") + "\n"

	return s
}
