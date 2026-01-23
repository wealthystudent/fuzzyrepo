package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	selectedRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))
)

type reposUpdatedMsg []Repository
type refreshStartedMsg struct{}
type refreshFinishedMsg struct{}

type Action int

const (
	ActionNone Action = iota
	ActionOpen
	ActionCopy
)

type Model struct {
	all     []Repository
	query   string
	results []Repository

	cursor int

	status     string
	refreshing bool

	width  int
	height int

	config         Config
	selectedRepo   *Repository
	selectedAction Action

	refreshChan chan<- struct{}
}

func newModel(all []Repository, config Config, refreshChan chan<- struct{}) Model {
	m := Model{
		all:         all,
		query:       "",
		config:      config,
		refreshChan: refreshChan,
	}
	m.applySearch()
	m.status = fmt.Sprintf("%d repos · ↑↓ navigate · Enter open · y copy · r refresh · , config · q quit", len(all))
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case refreshStartedMsg:
		m.refreshing = true
		m.status = "Refreshing..."
		return m, nil

	case refreshFinishedMsg:
		m.refreshing = false
		m.status = "Refresh complete."
		return m, nil

	case reposUpdatedMsg:
		m.all = []Repository(msg)
		m.applySearch()

		if m.cursor >= len(m.results) {
			m.cursor = max(0, len(m.results)-1)
		}
		m.status = fmt.Sprintf("Repos loaded: %d", len(m.all))
		return m, nil

	case configEditedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Config error: %v", msg.err)
		} else {
			m.config = msg.config
			m.status = "Config reloaded"
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.results) == 0 {
				return m, nil
			}
			r := m.results[m.cursor]
			m.selectedRepo = &r
			m.selectedAction = ActionOpen
			return m, tea.Quit

		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applySearch()
			}
			return m, nil

		default:
			if msg.Type == tea.KeyRunes {
				key := msg.String()
				switch key {
				case "q":
					return m, tea.Quit
				case "y":
					if len(m.results) > 0 {
						r := m.results[m.cursor]
						m.selectedRepo = &r
						m.selectedAction = ActionCopy
						return m, tea.Quit
					}
				case "r":
					if !m.refreshing {
						m.refreshing = true
						m.status = "Refreshing..."
						select {
						case m.refreshChan <- struct{}{}:
						default:
						}
					}
				case ",":
					return m, openConfigInEditor(m.config)
				default:
					m.query += key
					m.applySearch()
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *Model) applySearch() {
	q := strings.TrimSpace(m.query)
	if q == "" {
		m.results = m.all
		m.cursor = 0
		return
	}

	haystack := make([]string, 0, len(m.all))
	for _, r := range m.all {
		haystack = append(haystack, r.SearchText)
	}

	matches := fuzzy.Find(q, haystack)
	out := make([]Repository, 0, len(matches))
	for _, mt := range matches {
		out = append(out, m.all[mt.Index])
	}

	m.results = out
	if m.cursor >= len(m.results) {
		m.cursor = max(0, len(m.results)-1)
	}
}

type configEditedMsg struct {
	config Config
	err    error
}

func openConfigInEditor(cfg Config) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			return configEditedMsg{err: ErrNoEditor}
		}

		configPath := ConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := SaveConfig(cfg); err != nil {
				return configEditedMsg{err: err}
			}
		}

		cmd := exec.Command(editor, configPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return configEditedMsg{err: err}
		}

		newCfg, err := LoadConfig()
		return configEditedMsg{config: newCfg, err: err}
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("fuzzyrepo"))
	b.WriteString("\n\n")

	ownerW := 20
	nameW := 30
	localW := 5
	sepW := 3

	if m.width > 0 {
		ownerW = clamp(m.width/4, 10, 25)
		nameW = clamp(m.width/3, 15, 40)
	}

	b.WriteString(headerStyle.Render(
		padOrTrim("OWNER", ownerW) + " | " + padOrTrim("NAME", nameW) + " | " + padOrTrim("LOCAL", localW),
	))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, ownerW+nameW+localW+2*sepW))))
	b.WriteString("\n")

	maxRows := m.height - 9
	if maxRows < 3 {
		maxRows = 3
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := min(len(m.results), start+maxRows)

	if len(m.results) == 0 {
		b.WriteString(dimStyle.Render("No matches"))
		b.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			r := m.results[i]
			localStr := "no"
			if r.ExistsLocal {
				localStr = "yes"
			}
			row := padOrTrim(r.Owner, ownerW) +
				" | " + padOrTrim(r.Name, nameW) +
				" | " + padOrTrim(localStr, localW)

			if i == m.cursor {
				b.WriteString(selectedRow.Render(row))
			} else {
				b.WriteString(row)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.refreshing {
		b.WriteString(dimStyle.Render("refreshing..."))
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render(m.status))
	b.WriteString("\n")
	b.WriteString("> " + m.query)

	return b.String()
}

func ui(initial []Repository, config Config, uiMsgs <-chan tea.Msg, refreshChan chan<- struct{}) (*Repository, Action) {
	model := newModel(initial, config, refreshChan)
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		for msg := range uiMsgs {
			p.Send(msg)
		}
	}()

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m := finalModel.(Model)
	return m.selectedRepo, m.selectedAction
}
