package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

var (
	bgColor = lipgloss.Color("#0a0a0a")

	repoNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	ownerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	localYesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#325555"))

	localNoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#913333"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#1a1a1a"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	keybindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	queryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	configLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	configInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff"))

	configModalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#444444")).
				Padding(1, 2).
				Background(lipgloss.Color("#111111"))
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

const (
	cfgRepoRoots = iota
	cfgCloneRoot
	cfgAffiliation
	cfgOrgs
	cfgMaxResults
	cfgFieldCount
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

	showConfig  bool
	configFocus int
	inputs      []textinput.Model
}

func newModel(all []Repository, config Config, refreshChan chan<- struct{}) Model {
	m := Model{
		all:         all,
		query:       "",
		config:      config,
		refreshChan: refreshChan,
		inputs:      make([]textinput.Model, cfgFieldCount),
	}

	for i := 0; i < cfgFieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 500
		ti.Width = 50
		m.inputs[i] = ti
	}

	m.inputs[cfgRepoRoots].Placeholder = "/path/to/repos,/another/path"
	m.inputs[cfgCloneRoot].Placeholder = "/path/to/clone/root"
	m.inputs[cfgAffiliation].Placeholder = "owner,collaborator,organization_member"
	m.inputs[cfgOrgs].Placeholder = "org1,org2"
	m.inputs[cfgMaxResults].Placeholder = "0 (unlimited)"

	m.applySearch()
	return m
}

func (m *Model) loadConfigIntoInputs() {
	m.inputs[cfgRepoRoots].SetValue(m.config.RepoRoots)
	m.inputs[cfgCloneRoot].SetValue(m.config.CloneRoot)
	m.inputs[cfgAffiliation].SetValue(m.config.GitHub.Affiliation)
	m.inputs[cfgOrgs].SetValue(m.config.GitHub.Orgs)
	m.inputs[cfgMaxResults].SetValue(strconv.Itoa(m.config.MaxResults))
}

func (m *Model) saveConfigFromInputs() error {
	maxResults, err := strconv.Atoi(m.inputs[cfgMaxResults].Value())
	if err != nil {
		maxResults = 0
	}

	cfg := Config{
		RepoRoots: m.inputs[cfgRepoRoots].Value(),
		CloneRoot: m.inputs[cfgCloneRoot].Value(),
		GitHub: GitHubConfig{
			Affiliation: m.inputs[cfgAffiliation].Value(),
			Orgs:        m.inputs[cfgOrgs].Value(),
		},
		MaxResults: maxResults,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	m.config = cfg
	return nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showConfig {
		return m.updateConfig(msg)
	}
	return m.updateMain(msg)
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.showConfig = false
			m.status = "config cancelled"
			return m, nil

		case tea.KeyEnter:
			if err := m.saveConfigFromInputs(); err != nil {
				m.status = fmt.Sprintf("config error: %v", err)
			} else {
				m.status = "config saved"
				select {
				case m.refreshChan <- struct{}{}:
				default:
				}
			}
			m.showConfig = false
			return m, nil

		case tea.KeyTab, tea.KeyDown:
			m.inputs[m.configFocus].Blur()
			m.configFocus = (m.configFocus + 1) % cfgFieldCount
			m.inputs[m.configFocus].Focus()
			return m, nil

		case tea.KeyShiftTab, tea.KeyUp:
			m.inputs[m.configFocus].Blur()
			m.configFocus = (m.configFocus - 1 + cfgFieldCount) % cfgFieldCount
			m.inputs[m.configFocus].Focus()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.configFocus], cmd = m.inputs[m.configFocus].Update(msg)
	return m, cmd
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
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
					m.showConfig = true
					m.configFocus = 0
					m.loadConfigIntoInputs()
					for i := range m.inputs {
						m.inputs[i].Blur()
					}
					m.inputs[0].Focus()
					return m, nil
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

func (m Model) View() string {
	if m.showConfig {
		return m.viewConfig()
	}
	return m.viewMain()
}

func (m Model) viewConfig() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("fuzzyrepo"))
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("config"))
	b.WriteString("\n\n")

	labels := []string{
		"repo_roots",
		"clone_root",
		"github.affiliation",
		"github.orgs",
		"max_results",
	}

	for i, label := range labels {
		b.WriteString(configLabelStyle.Render(fmt.Sprintf("%-20s", label)))
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(keybindStyle.Render("tab/↑↓ navigate   enter save   esc cancel"))

	return configModalStyle.Render(b.String())
}

func (m Model) viewMain() string {
	var b strings.Builder

	baseStyle := lipgloss.NewStyle().Background(bgColor)

	nameW := 35
	ownerW := 20
	localW := 6

	if m.width > 0 {
		nameW = clamp(m.width/3, 15, 45)
		ownerW = clamp(m.width/4, 10, 25)
	}

	b.WriteString(titleStyle.Render("fuzzyrepo"))
	b.WriteString("\n\n")

	header := headerStyle.Render(
		padOrTrim("REPO", nameW) + "  " + padOrTrim("LOCAL", localW) + "  " + padOrTrim("OWNER", ownerW),
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

			localText := "remote"
			localStyled := localNoStyle.Render(padOrTrim(localText, localW))
			if r.ExistsLocal {
				localText = "local"
				localStyled = localYesStyle.Render(padOrTrim(localText, localW))
			}

			if i == m.cursor {
				row := cursorStyle.Render(
					padOrTrim(r.Name, nameW) + "  " +
						padOrTrim(localText, localW) + "  " +
						padOrTrim(r.Owner, ownerW),
				)
				b.WriteString(row)
			} else {
				namePart := repoNameStyle.Render(padOrTrim(r.Name, nameW))
				ownerPart := ownerStyle.Render(padOrTrim(r.Owner, ownerW))
				row := namePart + "  " + localStyled + "  " + ownerPart
				b.WriteString(row)
			}
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

	keybinds := "↑↓ navigate   enter open   y copy   r refresh   , config   q quit"
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
