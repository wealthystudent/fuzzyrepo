package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

var (
	bgColor       = lipgloss.Color("#0a0a0a")
	cursorBgColor = lipgloss.Color("#1a1a1a")
)

var (
	bgOnlyStyle = lipgloss.NewStyle().
			Background(bgColor)

	repoNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Background(bgColor)

	ownerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Background(bgColor)

	localYesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#325555")).
			Background(bgColor)

	localNoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#913333")).
			Background(bgColor)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(cursorBgColor)

	cursorSepStyle = lipgloss.NewStyle().
			Background(cursorBgColor)

	localYesCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#325555")).
				Background(cursorBgColor)

	localNoCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#913333")).
				Background(cursorBgColor)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Background(bgColor)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Background(bgColor)

	keybindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FFAF")).
			Background(bgColor)

	inputText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00CECD")).
			Background(bgColor)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Background(bgColor)

	queryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(bgColor)

	configLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Background(bgColor)

	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			BorderBackground(bgColor).
			Padding(0, 1).
			Background(bgColor)
)

type reposUpdatedMsg []Repository
type refreshStartedMsg struct{}
type refreshFinishedMsg struct{}
type errorMsg struct{ err error }

type Action int

const (
	ActionNone Action = iota
	ActionOpen
	ActionCopy
	ActionBrowse
	ActionPRs
	ActionQuit
)

const (
	cfgRepoRoots = iota
	cfgCloneRoot
	cfgAffiliation
	cfgOrgs
	cfgFieldCount
)

type Model struct {
	all     []Repository
	query   string
	results []Repository
	usage   UsageData
	cursor  int

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

	showCommands  bool
	commandCursor int
}

type command struct {
	key    string
	name   string
	action Action
	fn     func(*Model)
}

func newModel(all []Repository, config Config, refreshChan chan<- struct{}) Model {
	usage, _ := LoadUsage()

	m := Model{
		all:         all,
		query:       "",
		config:      config,
		usage:       usage,
		refreshChan: refreshChan,
		inputs:      make([]textinput.Model, cfgFieldCount),
	}

	for i := 0; i < cfgFieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 500
		ti.Width = 50
		ti.TextStyle = lipgloss.NewStyle().Background(bgColor).Foreground(lipgloss.Color("#ffffff"))
		ti.PlaceholderStyle = lipgloss.NewStyle().Background(bgColor).Foreground(lipgloss.Color("#444444"))
		ti.PromptStyle = lipgloss.NewStyle().Background(bgColor)
		ti.Cursor.Style = lipgloss.NewStyle().Background(bgColor)
		m.inputs[i] = ti
	}

	m.inputs[cfgRepoRoots].Placeholder = "/path/to/repos,/another/path"
	m.inputs[cfgCloneRoot].Placeholder = "/path/to/clone/root"
	m.inputs[cfgAffiliation].Placeholder = "owner,collaborator,organization_member"
	m.inputs[cfgOrgs].Placeholder = "org1,org2"

	m.applySearch()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showConfig {
		return m.updateConfig(msg)
	}
	return m.updateMain(msg)
}

func (m Model) View() string {
	return m.viewMain()
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.showConfig = false
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
		if m.status == "refreshing..." {
			m.status = ""
		}
		return m, nil

	case errorMsg:
		m.status = msg.err.Error()
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
		if m.showCommands {
			return m.updateCommands(msg)
		}

		switch msg.Type {

		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEsc:
			if m.query != "" {
				m.query = ""
				m.applySearch()
				return m, nil
			}
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

		case tea.KeySpace:
			if m.query == "" {
				m.showCommands = true
				m.commandCursor = 0
				return m, nil
			}
			m.query += " "
			m.applySearch()
			return m, nil

		default:
			if msg.Type == tea.KeyRunes {
				m.query += msg.String()
				m.applySearch()
				return m, nil
			}
		}
	}

	return m, nil
}

func (m Model) updateCommands(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmds := m.getCommands()

	switch msg.Type {
	case tea.KeyEsc, tea.KeySpace:
		m.showCommands = false
		return m, nil

	case tea.KeyUp:
		if m.commandCursor > 0 {
			m.commandCursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.commandCursor < len(cmds)-1 {
			m.commandCursor++
		}
		return m, nil

	case tea.KeyEnter:
		cmd := cmds[m.commandCursor]
		m.showCommands = false
		if cmd.action == ActionQuit {
			return m, tea.Quit
		}
		if cmd.action != ActionNone {
			if len(m.results) > 0 {
				r := m.results[m.cursor]
				m.selectedRepo = &r
				m.selectedAction = cmd.action
				return m, tea.Quit
			}
			return m, nil
		}
		if cmd.fn != nil {
			cmd.fn(&m)
		}
		return m, nil

	default:
		if msg.Type == tea.KeyRunes {
			key := msg.String()
			for _, cmd := range cmds {
				if cmd.key == key {
					m.showCommands = false
					if cmd.action == ActionQuit {
						return m, tea.Quit
					}
					if cmd.action != ActionNone {
						if len(m.results) > 0 {
							r := m.results[m.cursor]
							m.selectedRepo = &r
							m.selectedAction = cmd.action
							return m, tea.Quit
						}
						return m, nil
					}
					if cmd.fn != nil {
						cmd.fn(&m)
					}
					return m, nil
				}
			}
		}
	}

	return m, nil
}

func (m Model) viewMain() string {
	var b strings.Builder

	localW := 6
	separators := 4
	ownerW := 20
	nameW := 35

	if m.width > 0 {
		ownerW = clamp(m.width/4, 10, 30)
		nameW = m.width - ownerW - localW - separators
		nameW = max(15, nameW)
	}

	sep := bgOnlyStyle.Render("  ")

	header := headerStyle.Render(padOrTrim("REPO", nameW)) + sep +
		headerStyle.Render(padOrTrim("LOCAL", localW)) + sep +
		headerStyle.Render(padOrTrim("OWNER", ownerW))
	b.WriteString(header)
	b.WriteString("\n")

	maxRows := 8
	if m.height > 0 {
		maxRows = max(5, m.height-6)
	}

	total := len(m.results)
	end := total
	start := max(0, end-maxRows)
	if m.cursor < start {
		start = m.cursor
		end = min(total, start+maxRows)
	}
	if m.cursor >= end {
		end = m.cursor + 1
		start = max(0, end-maxRows)
	}

	for i := 0; i < maxRows-(end-start); i++ {
		b.WriteString("\n")
	}

	overlayOpen := m.showCommands || m.showConfig

	if total == 0 {
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

			if i == m.cursor && !overlayOpen {
				cursorSep := cursorSepStyle.Render("  ")
				namePart := cursorStyle.Render(padOrTrim(r.Name, nameW))
				ownerPart := cursorStyle.Render(padOrTrim(r.Owner, ownerW))

				localPart := localNoCursorStyle.Render(padOrTrim(localText, localW))
				if r.ExistsLocal {
					localPart = localYesCursorStyle.Render(padOrTrim(localText, localW))
				}

				b.WriteString(namePart + cursorSep + localPart + cursorSep + ownerPart)
			} else {
				namePart := repoNameStyle.Render(padOrTrim(r.Name, nameW))
				ownerPart := ownerStyle.Render(padOrTrim(r.Owner, ownerW))
				b.WriteString(namePart + sep + localStyled + sep + ownerPart)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	searchLeft := promptStyle.Render("> ") + queryStyle.Render(m.query)
	if m.query == "" {
		searchLeft += inputText.Render("type to search")
	}

	hints := keybindStyle.Render("space commands  enter open  ")
	searchLeftWidth := lipgloss.Width(searchLeft)
	hintsWidth := lipgloss.Width(hints)
	padding := m.width - searchLeftWidth - hintsWidth
	if padding < 2 {
		padding = 2
	}

	b.WriteString(searchLeft + bgOnlyStyle.Render(strings.Repeat(" ", padding)) + hints)

	mainContent := b.String()
	baseStyle := lipgloss.NewStyle().
		Background(bgColor).
		Width(m.width).
		Height(m.height)
	mainRendered := baseStyle.Render(mainContent)

	if m.showConfig {
		return m.overlayCenter(mainRendered, m.buildConfigBox())
	}
	if m.showCommands {
		return m.overlayCenter(mainRendered, m.buildCommandBox())
	}

	return mainRendered
}

func (m Model) buildCommandBox() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Background(bgColor).
		Width(3)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(bgColor)

	selectedKeyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(bgColor).
		Width(3)

	selectedNameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(bgColor)

	cmds := m.getCommands()
	var lines []string
	lines = append(lines, inputText.Render("Commands"))

	for i, cmd := range cmds {
		if i == m.commandCursor {
			line := selectedKeyStyle.Render(cmd.key) + bgOnlyStyle.Render(" ") + selectedNameStyle.Render(cmd.name)
			lines = append(lines, line)
		} else {
			line := keyStyle.Render(cmd.key) + bgOnlyStyle.Render(" ") + nameStyle.Render(cmd.name)
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	lines = append(lines, keybindStyle.Render("↑↓ navigate  enter select  esc close"))

	return overlayStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) buildConfigBox() string {
	labels := []string{
		"repo_roots",
		"clone_root",
		"github.affiliation",
		"github.orgs",
	}

	var lines []string
	lines = append(lines, inputText.Render("Config"))
	lines = append(lines, "")

	for i, label := range labels {
		line := configLabelStyle.Render(fmt.Sprintf("%-20s", label)) + m.inputs[i].View()
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, keybindStyle.Render("tab/↑↓ navigate   enter save   esc close"))

	return overlayStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) overlayCenter(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayH := len(overlayLines)
	overlayW := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > overlayW {
			overlayW = w
		}
	}

	startRow := (m.height - overlayH) / 2
	startCol := (m.width - overlayW) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	for i, overlayLine := range overlayLines {
		row := startRow + i
		if row >= len(baseLines) {
			continue
		}

		baseLine := baseLines[row]
		baseRunes := []rune(stripAnsi(baseLine))

		for len(baseRunes) < m.width {
			baseRunes = append(baseRunes, ' ')
		}

		prefix := string(baseRunes[:startCol])
		suffix := ""
		if startCol+overlayW < len(baseRunes) {
			suffix = string(baseRunes[startCol+overlayW:])
		}

		lineW := lipgloss.Width(overlayLine)
		if lineW < overlayW {
			overlayLine += bgOnlyStyle.Render(strings.Repeat(" ", overlayW-lineW))
		}

		baseLines[row] = dimStyle.Render(prefix) + overlayLine + "\x1b[0m" + dimStyle.Render(suffix)
	}

	return strings.Join(baseLines, "\n")
}

func (m *Model) applySearch() {
	q := strings.TrimSpace(m.query)
	if q == "" {
		m.results = SortByUsage(m.all, m.usage)
		m.cursor = max(0, len(m.results)-1)
		return
	}

	haystack := make([]string, 0, len(m.all))
	for _, r := range m.all {
		haystack = append(haystack, r.SearchText)
	}

	matches := fuzzy.Find(q, haystack)

	type scoredRepo struct {
		repo       Repository
		fuzzyScore int
		usageBoost float64
		combined   float64
	}

	scored := make([]scoredRepo, 0, len(matches))
	for _, mt := range matches {
		repo := m.all[mt.Index]
		usageBoost := GetUsageBoost(m.usage, repo)
		combined := float64(mt.Score) + usageBoost*50
		scored = append(scored, scoredRepo{
			repo:       repo,
			fuzzyScore: mt.Score,
			usageBoost: usageBoost,
			combined:   combined,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].combined < scored[j].combined
	})

	m.results = make([]Repository, 0, len(scored))
	for _, s := range scored {
		m.results = append(m.results, s.repo)
	}

	m.cursor = max(0, len(m.results)-1)
}

func (m *Model) loadConfigIntoInputs() {
	m.inputs[cfgRepoRoots].SetValue(strings.Join(m.config.RepoRoots, ", "))
	m.inputs[cfgCloneRoot].SetValue(m.config.CloneRoot)
	m.inputs[cfgAffiliation].SetValue(m.config.GitHub.Affiliation)
	m.inputs[cfgOrgs].SetValue(m.config.GitHub.Orgs)
}

func (m *Model) saveConfigFromInputs() error {
	var repoRoots []string
	for _, p := range strings.Split(m.inputs[cfgRepoRoots].Value(), ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			repoRoots = append(repoRoots, p)
		}
	}

	cfg := Config{
		RepoRoots: repoRoots,
		CloneRoot: m.inputs[cfgCloneRoot].Value(),
		GitHub: GitHubConfig{
			Affiliation: m.inputs[cfgAffiliation].Value(),
			Orgs:        m.inputs[cfgOrgs].Value(),
		},
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

func (m *Model) getCommands() []command {
	return []command{
		{key: "o", name: "open in editor", action: ActionOpen},
		{key: "y", name: "copy path", action: ActionCopy},
		{key: "b", name: "open in browser", action: ActionBrowse},
		{key: "p", name: "open pull requests", action: ActionPRs},
		{key: "r", name: "refresh", fn: func(m *Model) {
			if !m.refreshing {
				m.refreshing = true
				m.status = "refreshing..."
				select {
				case m.refreshChan <- struct{}{}:
				default:
				}
			}
		}},
		{key: "c", name: "config", fn: func(m *Model) {
			m.showConfig = true
			m.configFocus = 0
			m.loadConfigIntoInputs()
			for i := range m.inputs {
				m.inputs[i].Blur()
			}
			m.inputs[0].Focus()
		}},
		{key: "q", name: "quit", action: ActionQuit},
	}
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
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
