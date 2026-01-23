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
	bgColor = lipgloss.Color("#0a0a0a")

	repoNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true)

	ownerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	localYesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	localNoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#1a1a1a"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	keybindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	queryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))
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
		m.status = "refreshing..."
		return m, nil

	case refreshFinishedMsg:
		m.refreshing = false
		m.status = ""
		return m, nil

	case reposUpdatedMsg:
		m.all = []Repository(msg)
		m.applySearch()

		if m.cursor >= len(m.results) {
			m.cursor = max(0, len(m.results)-1)
		}
		m.status = fmt.Sprintf("%d repos", len(m.all))
		return m, nil

	case configEditedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("config error: %v", msg.err)
		} else {
			m.config = msg.config
			m.status = "config reloaded"
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
						m.status = "refreshing..."
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

	baseStyle := lipgloss.NewStyle().Background(bgColor)

	nameW := 35
	localW := 3
	ownerW := 20

	if m.width > 0 {
		nameW = clamp(m.width/2-10, 20, 50)
		ownerW = clamp(m.width/4, 10, 25)
	}

	b.WriteString(titleStyle.Render("fuzzyrepo"))
	b.WriteString("\n\n")

	header := headerStyle.Render(
		padOrTrim("REPO", nameW) + "  " + padOrTrim("LOCAL", localW),
	)
	b.WriteString(header)
	b.WriteString("\n")

	maxRows := 8
	if m.height > 0 {
		maxRows = clamp(m.height-8, 5, 15)
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := min(len(m.results), start+maxRows)

	if len(m.results) == 0 {
		b.WriteString(dimStyle.Render("no matches"))
		b.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			r := m.results[i]

			localStr := "·"
			localStyled := localNoStyle.Render(localStr)
			if r.ExistsLocal {
				localStr = "✓"
				localStyled = localYesStyle.Render(localStr)
			}

			namePart := repoNameStyle.Render(padOrTrim(r.Name, nameW-ownerW-2))
			ownerPart := ownerStyle.Render(padOrTrim(r.Owner, ownerW))

			nameCell := namePart + "  " + ownerPart

			row := nameCell + "  " + localStyled

			if i == m.cursor {
				row = cursorStyle.Width(m.width).Render(
					padOrTrim(r.Name, nameW-ownerW-2) + "  " +
						padOrTrim(r.Owner, ownerW) + "  " + localStr,
				)
			}

			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	for i := end - start; i < maxRows; i++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(promptStyle.Render("> ") + queryStyle.Render(m.query))
	if m.query == "" {
		b.WriteString(dimStyle.Render("type to search"))
	}
	b.WriteString("\n\n")

	if m.status != "" {
		b.WriteString(dimStyle.Render(m.status))
		b.WriteString("\n")
	}

	keybinds := "↑↓ navigate  enter open  y copy  r refresh  , config  q quit"
	b.WriteString(keybindStyle.Render(keybinds))

	return baseStyle.Render(b.String())
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
