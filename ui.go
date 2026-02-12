package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type reposUpdatedMsg []Repository
type refreshStartedMsg struct{}
type refreshFinishedMsg struct{}
type errorMsg struct{ err error }
type cacheCheckTickMsg struct{} // Periodic tick to check cache file changes
type clearMessageMsg struct{}   // Timer to clear status message
type configEditedMsg struct{}   // Config file was edited externally

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
	cfgUseCloneRules
	cfgOrgs
	cfgShowOwner
	cfgShowCollaborator
	cfgShowOrgMember
	cfgShowLocal
	cfgFieldCount
)

type Model struct {
	cache   []Repository // Full unfiltered cache
	all     []Repository // Filtered repos for display
	query   string
	results []Repository
	usage   UsageData
	cursor  int

	message    StatusMessage
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

	// Cache file watching
	cacheMtime time.Time

	// First run state
	firstRun bool
}

// setMessage sets the status message with the given level
func (m *Model) setMessage(text string, level MessageLevel) {
	m.message = StatusMessage{Text: text, Level: level}
}

// clearMessage clears the status message
func (m *Model) clearMessage() {
	m.message = StatusMessage{}
}

// clearMessageAfter returns a command that clears the message after a delay
func clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

type command struct {
	key    string
	name   string
	action Action
	fn     func(*Model)
}

func newModel(cache []Repository, config Config, refreshChan chan<- struct{}, cacheMtime time.Time, firstRun bool) Model {
	usage, _ := LoadUsage()

	// Apply filter to get display repos
	filtered := filterRepos(cache, config)

	m := Model{
		cache:       cache,
		all:         filtered,
		query:       "",
		config:      config,
		usage:       usage,
		refreshChan: refreshChan,
		inputs:      make([]textinput.Model, cfgFieldCount),
		cacheMtime:  cacheMtime,
		firstRun:    firstRun,
	}

	for i := 0; i < cfgFieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 500
		ti.Width = 50
		ti.TextStyle = lipgloss.NewStyle().Background(bgColor).Foreground(lipgloss.Color("#ffffff"))
		ti.PlaceholderStyle = lipgloss.NewStyle().Background(bgColor).Foreground(lipgloss.Color("#444444"))
		ti.PromptStyle = lipgloss.NewStyle().Background(bgColor)
		ti.Cursor.Style = lipgloss.NewStyle().Background(bgColor)
		ti.Cursor.TextStyle = lipgloss.NewStyle().Background(bgColor)
		m.inputs[i] = ti
	}

	m.inputs[cfgRepoRoots].Placeholder = "/path/to/repos,/another/path"
	m.inputs[cfgCloneRoot].Placeholder = "/path/to/clone/root"
	m.inputs[cfgUseCloneRules].Placeholder = "no"
	m.inputs[cfgOrgs].Placeholder = "org1,org2 (empty = all)"
	m.inputs[cfgShowOwner].Placeholder = "yes"
	m.inputs[cfgShowCollaborator].Placeholder = "yes"
	m.inputs[cfgShowOrgMember].Placeholder = "yes"
	m.inputs[cfgShowLocal].Placeholder = "yes"

	// Shorter width for boolean fields
	m.inputs[cfgUseCloneRules].Width = 5
	m.inputs[cfgShowOwner].Width = 5
	m.inputs[cfgShowCollaborator].Width = 5
	m.inputs[cfgShowOrgMember].Width = 5
	m.inputs[cfgShowLocal].Width = 5

	m.applySearch()
	return m
}

// cacheCheckInterval defines how often to check for cache file changes
const cacheCheckInterval = 2 * time.Second

// tickCacheCheck returns a command that sends a tick after the interval
func tickCacheCheck() tea.Cmd {
	return tea.Tick(cacheCheckInterval, func(t time.Time) tea.Msg {
		return cacheCheckTickMsg{}
	})
}

// openConfigInEditor opens the config file in the user's $EDITOR
func openConfigInEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback
	}
	configPath := xdgConfigPath()
	c := exec.Command(editor, configPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return configEditedMsg{}
	})
}

// Init starts the cache file watcher ticker
func (m Model) Init() tea.Cmd {
	return tickCacheCheck()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle messages that should be processed regardless of overlay state
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case clearMessageMsg:
		m.clearMessage()
		return m, nil

	case cacheCheckTickMsg:
		// Check if cache file has been updated by external process
		currentMtime := GetCacheMtime()
		if !currentMtime.IsZero() && currentMtime.After(m.cacheMtime) {
			// Cache file was updated, reload it
			if repos, err := loadRepoCache(); err == nil && len(repos) > 0 {
				m.cache = repos
				m.all = filterRepos(repos, m.config)
				m.applySearch()
				if m.cursor >= len(m.results) {
					m.cursor = max(0, len(m.results)-1)
				}
				m.cacheMtime = currentMtime
				m.refreshing = false // Clear refreshing state since sync completed
				m.setMessage(fmt.Sprintf("%d repos loaded", len(m.all)), InfoLevel)
				// Clear message after 5 seconds
				return m, tea.Batch(tickCacheCheck(), clearMessageAfter(5*time.Second))
			}
		}
		// Always schedule next tick
		return m, tickCacheCheck()
	}

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
			changes, err := m.saveConfigFromInputs()
			if err != nil {
				m.setMessage(fmt.Sprintf("config error: %v", err), ErrorLevel)
			} else {
				// Always reapply filters after config save
				m.all = filterRepos(m.cache, m.config)
				m.applySearch()
				if m.cursor >= len(m.results) {
					m.cursor = max(0, len(m.results)-1)
				}
				m.setMessage(fmt.Sprintf("config saved, %d repos", len(m.all)), InfoLevel)

				// If repo_roots changed, trigger local scan to update repos
				if changes.repoRootsChanged {
					m.setMessage("config saved, scanning local repos...", InfoLevel)
					if updated, err := runLocalScan(m.config, m.cache); err == nil {
						m.cache = updated
						m.all = filterRepos(m.cache, m.config)
						m.applySearch()
						m.cacheMtime = GetCacheMtime()
						m.setMessage(fmt.Sprintf("config saved, %d repos", len(m.all)), InfoLevel)
					}
				}

				// On first run, spawn background sync to fetch remote repos
				if m.firstRun {
					m.firstRun = false
					if !isSyncRunning() {
						if spawnDetachedSync() {
							m.setMessage("Config saved, syncing repositories...", InfoLevel)
							m.refreshing = true
						}
					}
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

		case tea.KeySpace:
			return m, openConfigInEditor()
		}
	case configEditedMsg:
		// Reload config after external edit and close overlay
		if cfg, err := LoadConfig(); err == nil {
			m.config = cfg
			m.loadConfigIntoInputs()
			m.all = filterRepos(m.cache, m.config)
			m.applySearch()
			m.setMessage("config reloaded", InfoLevel)
		} else {
			m.setMessage(fmt.Sprintf("config reload error: %v", err), ErrorLevel)
		}
		m.showConfig = false // Close config overlay after external edit
		return m, nil
	}

	var cmd tea.Cmd
	m.inputs[m.configFocus], cmd = m.inputs[m.configFocus].Update(msg)
	return m, cmd
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case refreshStartedMsg:
		m.refreshing = true
		m.setMessage("refreshing...", InfoLevel)
		return m, nil

	case refreshFinishedMsg:
		m.refreshing = false
		if m.message.Text == "refreshing..." {
			m.setMessage("Sync complete", InfoLevel)
			return m, clearMessageAfter(5 * time.Second)
		}
		return m, nil

	case errorMsg:
		m.setMessage(msg.err.Error(), ErrorLevel)
		return m, nil

	case reposUpdatedMsg:
		m.cache = []Repository(msg)
		m.all = filterRepos(m.cache, m.config)
		m.applySearch()

		if m.cursor >= len(m.results) {
			m.cursor = max(0, len(m.results)-1)
		}
		m.setMessage(fmt.Sprintf("%d repos loaded", len(m.all)), InfoLevel)
		// Update our tracked mtime since we just got new data
		m.cacheMtime = GetCacheMtime()
		return m, clearMessageAfter(5 * time.Second)

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
			m.showCommands = true
			m.commandCursor = 0
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				r := msg.String()
				if r == " " {
					m.showCommands = true
					m.commandCursor = 0
					return m, nil
				}
				m.query += r
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
	// Use sensible defaults if window size not yet received
	width := m.width
	height := m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	var b strings.Builder

	localW := 6
	separators := 4
	ownerW := 20
	nameW := 35

	if width > 0 {
		ownerW = clamp(width/4, 10, 30)
		nameW = width - ownerW - localW - separators
		nameW = max(15, nameW)
	}

	sep := bgOnlyStyle.Render("  ")

	header := headerStyle.Render(padOrTrim("REPO", nameW)) + sep +
		headerStyle.Render(padOrTrim("LOCAL", localW)) + sep +
		headerStyle.Render(padOrTrim("OWNER", ownerW))
	b.WriteString(padLineToWidth(header, width, bgOnlyStyle))
	b.WriteString("\n")

	maxRows := 8
	if height > 0 {
		// Account for: header(1) + separator(1) + message box(0-3) + search(1) + buffer
		reserved := 4
		if !m.message.IsEmpty() {
			reserved += 3 // empty line + message + empty line
		}
		maxRows = max(5, height-reserved)
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

	// Empty lines above the list (padding)
	emptyLine := bgOnlyStyle.Render(strings.Repeat(" ", width))
	for i := 0; i < maxRows-(end-start); i++ {
		b.WriteString(emptyLine)
		b.WriteString("\n")
	}

	overlayOpen := m.showCommands || m.showConfig

	if total == 0 {
		b.WriteString(padLineToWidth(dimStyle.Render("no matches"), width, bgOnlyStyle))
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

			var line string
			if i == m.cursor && !overlayOpen {
				cursorSep := cursorSepStyle.Render("  ")
				namePart := cursorStyle.Render(padOrTrim(r.Name, nameW))
				ownerPart := cursorStyle.Render(padOrTrim(r.Owner, ownerW))

				localPart := localNoCursorStyle.Render(padOrTrim(localText, localW))
				if r.ExistsLocal {
					localPart = localYesCursorStyle.Render(padOrTrim(localText, localW))
				}

				line = namePart + cursorSep + localPart + cursorSep + ownerPart
				b.WriteString(padLineToWidth(line, width, cursorSepStyle))
			} else {
				namePart := repoNameStyle.Render(padOrTrim(r.Name, nameW))
				ownerPart := ownerStyle.Render(padOrTrim(r.Owner, ownerW))
				line = namePart + sep + localStyled + sep + ownerPart
				b.WriteString(padLineToWidth(line, width, bgOnlyStyle))
			}
			b.WriteString("\n")
		}
	}

	// Separator line between list and search input
	b.WriteString(emptyLine)
	b.WriteString("\n")

	// Message box (only if there's a message) - includes padding lines above/below
	if !m.message.IsEmpty() {
		b.WriteString(m.message.Render(width))
		b.WriteString("\n")
	}

	searchLeft := promptStyle.Render("> ") + queryStyle.Render(m.query)
	if m.query == "" {
		searchLeft += inputTextStyle.Render("type to search")
	}

	hints := keybindStyle.Render("space commands  enter open  ")
	searchLeftWidth := lipgloss.Width(searchLeft)
	hintsWidth := lipgloss.Width(hints)
	padding := width - searchLeftWidth - hintsWidth
	if padding < 2 {
		padding = 2
	}

	b.WriteString(searchLeft + bgOnlyStyle.Render(strings.Repeat(" ", padding)) + hints)

	mainContent := b.String()
	mainRendered := lipgloss.Place(width, height, lipgloss.Left, lipgloss.Bottom, mainContent, lipgloss.WithWhitespaceBackground(bgColor))

	if m.showConfig {
		configContent := m.buildConfigBox()
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, configContent, lipgloss.WithWhitespaceBackground(bgColor))
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
	lines = append(lines, inputTextStyle.Render("Commands"))

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
	// Main config labels (text inputs)
	mainLabels := []string{
		"Repository Dirs",
		"Clone Directory",
		"Use Clone Rules",
		"GitHub Orgs",
	}

	// Filter labels (yes/no inputs)
	filterLabels := []string{
		"Show Owned",
		"Show Collaborator",
		"Show Org Member",
		"Show Local Only",
	}

	// Calculate the width of the config content (label + input)
	// Label is 20 chars, input is 50 chars wide
	contentWidth := 20 + 50

	// Respect terminal width - leave some margin for the overlay border/padding
	maxWidth := contentWidth
	if m.width > 0 && m.width-6 < maxWidth {
		maxWidth = m.width - 6
		if maxWidth < 40 {
			maxWidth = 40
		}
	}

	var lines []string
	lines = append(lines, inputTextStyle.Render("Config"))
	lines = append(lines, "")

	// Main config fields
	for i, label := range mainLabels {
		line := configLabelStyle.Render(fmt.Sprintf("%-20s", label)) + m.inputs[i].View()
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Filters:"))

	// Filter fields
	for i, label := range filterLabels {
		fieldIdx := cfgShowOwner + i
		line := configLabelStyle.Render(fmt.Sprintf("%-20s", label)) + m.inputs[fieldIdx].View()
		lines = append(lines, line)
	}

	lines = append(lines, "")

	// Show description for focused field using the info box style
	if desc, ok := ConfigFieldDescriptions[m.configFocus]; ok {
		// Render info box - text will wrap if needed
		descBox := RenderConfigBox(desc, maxWidth, 1)
		// Add each line of the box
		for _, line := range strings.Split(descBox, "\n") {
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	lines = append(lines, keybindStyle.Render("tab/↑↓ navigate   enter save   space edit file   esc close"))

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
	m.inputs[cfgUseCloneRules].SetValue(boolToYesNo(m.config.UseCloneRules))
	m.inputs[cfgOrgs].SetValue(m.config.GitHub.Orgs)
	m.inputs[cfgShowOwner].SetValue(boolToYesNo(m.config.ShowOwner))
	m.inputs[cfgShowCollaborator].SetValue(boolToYesNo(m.config.ShowCollaborator))
	m.inputs[cfgShowOrgMember].SetValue(boolToYesNo(m.config.ShowOrgMember))
	m.inputs[cfgShowLocal].SetValue(boolToYesNo(m.config.ShowLocal))
}

// boolToYesNo converts a bool to "yes" or "no"
func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// yesNoToBool converts "yes"/"no" string to bool (defaults to true for empty/invalid)
func yesNoToBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s != "no" && s != "n" && s != "false" && s != "0"
}

type configChanges struct {
	repoRootsChanged bool
	filtersChanged   bool
}

func (m *Model) saveConfigFromInputs() (changes configChanges, err error) {
	var repoRoots []string
	for _, p := range strings.Split(m.inputs[cfgRepoRoots].Value(), ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			repoRoots = append(repoRoots, p)
		}
	}

	// Check if repo_roots changed
	oldRoots := m.config.RepoRoots
	changes.repoRootsChanged = !stringSlicesEqual(oldRoots, repoRoots)

	// Parse filter settings
	showOwner := yesNoToBool(m.inputs[cfgShowOwner].Value())
	showCollaborator := yesNoToBool(m.inputs[cfgShowCollaborator].Value())
	showOrgMember := yesNoToBool(m.inputs[cfgShowOrgMember].Value())
	showLocal := yesNoToBool(m.inputs[cfgShowLocal].Value())

	// Check if filters changed
	changes.filtersChanged = showOwner != m.config.ShowOwner ||
		showCollaborator != m.config.ShowCollaborator ||
		showOrgMember != m.config.ShowOrgMember ||
		showLocal != m.config.ShowLocal

	cfg := Config{
		RepoRoots:     repoRoots,
		CloneRoot:     m.inputs[cfgCloneRoot].Value(),
		UseCloneRules: yesNoToBool(m.inputs[cfgUseCloneRules].Value()),
		CloneRules:    m.config.CloneRules, // Preserve existing clone rules (edited via config file)
		GitHub: GitHubConfig{
			Affiliation: "owner,collaborator,organization_member", // Always fetch all
			Orgs:        m.inputs[cfgOrgs].Value(),
		},
		ShowOwner:        showOwner,
		ShowCollaborator: showCollaborator,
		ShowOrgMember:    showOrgMember,
		ShowLocal:        showLocal,
	}

	if err := cfg.Validate(); err != nil {
		return configChanges{}, err
	}

	if err := SaveConfig(cfg); err != nil {
		return configChanges{}, err
	}

	m.config = cfg
	return changes, nil
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
				m.setMessage("refreshing...", InfoLevel)
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

func ui(initial []Repository, config Config, uiMsgs <-chan tea.Msg, refreshChan chan<- struct{}, cacheMtime time.Time, syncInProgress bool, firstRun bool) (*Repository, Action, Config) {
	model := newModel(initial, config, refreshChan, cacheMtime, firstRun)

	// Set initial status if background sync was spawned
	if syncInProgress {
		model.setMessage("Syncing repositories in background...", InfoLevel)
		model.refreshing = true
	}

	// On first run, auto-open config overlay so user can set up
	if firstRun {
		model.showConfig = true
		model.configFocus = 0
		model.setMessage("Welcome! Please configure your settings.", InfoLevel)
		model.loadConfigIntoInputs()
		for i := range model.inputs {
			model.inputs[i].Blur()
		}
		model.inputs[0].Focus()
	}

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
	return m.selectedRepo, m.selectedAction, m.config
}
