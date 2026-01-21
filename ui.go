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

// ---- Styles ----

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
)

// ---- Messages sent into the UI program ----

type reposUpdatedMsg []RepoDTO
type refreshStartedMsg struct{}
type refreshFinishedMsg struct{}

// ---- Bubble Tea model ----

type UIDTO struct {
	all     []RepoDTO
	query   string
	results []RepoDTO

	cursor int

	status     string
	refreshing bool

	width  int
	height int
}

func newModel(all []RepoDTO) UIDTO {
	m := UIDTO{
		all:   all,
		query: "",
	}
	m.applySearch()
	m.status = "Type to search · ↑/↓ to navigate · Enter to open · Esc/Ctrl+C to quit"
	return m
}

func (m UIDTO) Init() tea.Cmd { return nil }

func (m UIDTO) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case refreshStartedMsg:
		m.refreshing = true
		m.status = "Refreshing repo list in background..."
		return m, nil

	case refreshFinishedMsg:
		m.refreshing = false
		m.status = "Refresh complete."
		return m, nil

	case reposUpdatedMsg:
		m.all = []RepoDTO(msg)
		m.applySearch()

		if m.cursor >= len(m.results) {
			m.cursor = max(0, len(m.results)-1)
		}
		m.status = fmt.Sprintf("Repo list updated. Total: %d", len(m.all))
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

			// HOOK: replace with your real open/clone logic
			if err := openRepoDemo(r); err != nil {
				m.status = "Error: " + err.Error()
				return m, nil
			}
			m.status = "Opened: " + r.Name
			return m, nil

		case tea.KeyBackspace:
			if len(m.query) > 0 {
				// simple ASCII backspace; fine for typical repo names
				m.query = m.query[:len(m.query)-1]
				m.applySearch()
			}
			return m, nil

		default:
			// accept printable characters into the query (live update)
			if msg.Type == tea.KeyRunes {
				m.query += msg.String()
				m.applySearch()
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *UIDTO) applySearch() {
	q := strings.TrimSpace(m.query)
	if q == "" {
		m.results = m.all
		m.cursor = 0
		return
	}

	haystack := make([]string, 0, len(m.all))
	for _, r := range m.all {
		localStr := "remote"
		if r.ExistsLocal {
			localStr = "local"
		}
		haystack = append(haystack, r.Name+" "+r.Path+" "+localStr)
	}

	matches := fuzzy.Find(q, haystack)
	out := make([]RepoDTO, 0, len(matches))
	for _, mt := range matches {
		out = append(out, m.all[mt.Index])
	}

	m.results = out
	if m.cursor >= len(m.results) {
		m.cursor = max(0, len(m.results)-1)
	}
}

func (m UIDTO) View() string {
	var b strings.Builder

	// Column widths
	localW := 5
	sepW := 3
	pathW := clamp(m.width/2, 30, 80)
	nameW := max(10, m.width-(pathW+localW+2*sepW))

	// Header row
	b.WriteString(headerStyle.Render(
		padOrTrim("NAME", nameW) + " | " + padOrTrim("LOCAL", localW) + " | " + padOrTrim("PATH", pathW),
	))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", min(m.width, nameW+localW+pathW+2*sepW))))
	b.WriteString("\n")

	// Rows viewport
	maxRows := m.height - 7
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
			row := padOrTrim(r.Name, nameW) +
				" | " + padOrTrim(fmt.Sprintf("%v", r.ExistsLocal), localW) +
				" | " + padOrTrim(r.Path, pathW)

			if i == m.cursor {
				b.WriteString(selectedRow.Render(row))
			} else {
				b.WriteString(row)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Background refresh indicator
	if m.refreshing {
		b.WriteString(dimStyle.Render("refreshing in background..."))
		b.WriteString("\n")
	}

	// Status + Query
	b.WriteString(dimStyle.Render(m.status))
	b.WriteString("\n")
	b.WriteString("> " + m.query)

	return b.String()
}

// UI runner: pass initial repo list + a channel of tea.Msg from your background worker.
func ui(initial []RepoDTO, uiMsgs <-chan tea.Msg) {
	p := tea.NewProgram(newModel(initial), tea.WithAltScreen())

	// Forward background messages into Bubble Tea.
	go func() {
		for msg := range uiMsgs {
			p.Send(msg)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---- Hook demo: replace with your open/clone logic ----

func openRepoDemo(r RepoDTO) error {
	// Demo: if local, try `code <path>`
	if r.ExistsLocal {
		cmd := exec.Command("code", r.Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}
