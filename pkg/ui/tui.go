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

// progressMsg carries live status lines from a running transfer goroutine.
type progressMsg struct{ lines []string }

// cmdDoneMsg signals that the current async command finished.
type cmdDoneMsg struct{ lines []string }

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

	// Async command state
	running     bool        // true while a command goroutine is in flight
	statusLines []string    // live progress lines shown above the prompt
	transferCh  chan tea.Msg // goroutine sends progressMsg / cmdDoneMsg here
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

// waitForTransfer returns a tea.Cmd that blocks until the goroutine sends the
// next message on the transfer channel.
func waitForTransfer(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// Update handles messages and updates the model
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 7 // Leave room for header and input
		return m, nil

	// ── Progress update from a running transfer goroutine ────────────────────
	case progressMsg:
		m.statusLines = msg.lines
		return m, waitForTransfer(m.transferCh)

	// ── Transfer / command goroutine finished ────────────────────────────────
	case cmdDoneMsg:
		m.Output = append(m.Output, msg.lines...)
		m.statusLines = nil
		m.running = false
		m.transferCh = nil

		// Trim to max lines
		if len(m.Output) > m.maxOutputLines {
			m.Output = m.Output[len(m.Output)-m.maxOutputLines:]
		}
		m.viewport.SetContent(strings.Join(m.Output, "\n"))
		m.viewport.GotoBottom()

		// Re-focus the input and restart the blink ticker — it stops while
		// running=true since we don't forward messages to the textinput.
		m.textInput.Focus()
		return m, textinput.Blink

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
			// Block new commands while one is already running
			if m.running {
				return m, nil
			}

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

			// Echo command in viewport immediately
			m.Output = append(m.Output, styles.PromptStyle.Render("nfs> ")+command)
			m.viewport.SetContent(strings.Join(m.Output, "\n"))
			m.viewport.GotoBottom()

			// Determine transfer direction for the arrow in the progress bar
			trimmed := strings.TrimSpace(command)
			arrow := "⬇"
			if strings.HasPrefix(trimmed, "put") || strings.HasPrefix(trimmed, "mput") {
				arrow = "⬆"
			}

			// Launch goroutine — all commands run async so the TUI stays responsive
			ch := make(chan tea.Msg, 128)
			m.transferCh = ch
			m.running = true

			client := m.Client // capture pointer

			// Compute progress bar layout from current terminal width.
			// viewport.Width is set by WindowSizeMsg and defaults to 80.
			termWidth := m.viewport.Width
			if termWidth <= 0 {
				termWidth = 80
			}
			labelWidth, barWidth := nfs.ProgressLayout(termWidth)

			go func() {
				client.SetProgressFunc(func(u nfs.ProgressUpdate) {
					fileLine := arrow + " " + nfs.RenderFileProgress(u, labelWidth, barWidth)
					lines := []string{fileLine}
					if globalLine := nfs.RenderGlobalProgress(u, labelWidth, barWidth); globalLine != "" {
						lines = append(lines, "  "+globalLine)
					}
					ch <- progressMsg{lines: lines}
				})
				output := client.ExecuteCommand(command)
				client.SetProgressFunc(nil)
				ch <- cmdDoneMsg{lines: output}
			}()

			return m, waitForTransfer(ch)

		case tea.KeyUp:
			if !m.running && len(m.commandHistory) > 0 {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.textInput.SetValue(m.commandHistory[m.historyIndex])
				}
			}
			return m, nil

		case tea.KeyDown:
			if !m.running && len(m.commandHistory) > 0 {
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

	// Only forward key events to the text input when not running
	if !m.running {
		m.textInput, cmd = m.textInput.Update(msg)
	}
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

	// Scrollable output
	s += m.viewport.View() + "\n"

	// Live status lines — file bar in blue, global total bar in green
	for i, line := range m.statusLines {
		color := lipgloss.Color("#4B9BFF") // current file
		if i > 0 {
			color = lipgloss.Color("#04B575") // total batch progress
		}
		s += lipgloss.NewStyle().Foreground(color).Render(line) + "\n"
	}

	// Prompt — dimmed / locked while a command is running
	if m.running {
		s += lipgloss.NewStyle().Faint(true).Render("nfs (running)> ") + "\n"
	} else {
		s += styles.PromptStyle.Render("nfs> ") + m.textInput.View() + "\n"
	}

	s += lipgloss.NewStyle().Faint(true).Render("(PgUp/PgDn: scroll | ↑↓: history | Ctrl+C: quit)") + "\n"

	return s
}
