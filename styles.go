package main

import "github.com/charmbracelet/lipgloss"

var (
	bgColor         = lipgloss.Color("#1a1a1a")
	bgSelectedColor = lipgloss.Color("#2a2a2a")
	fgColor         = lipgloss.Color("#c0c0c0")
	fgDimColor      = lipgloss.Color("#555555")
	borderColor     = lipgloss.Color("#3d3d3d")
	cyanColor       = lipgloss.Color("#5dade2")
	greenColor      = lipgloss.Color("#6dce6d")
	redColor        = lipgloss.Color("#f38ba8")
	yellowColor     = lipgloss.Color("#f9e2af")
	whiteColor      = lipgloss.Color("#ffffff")
	magentaColor    = lipgloss.Color("#cba6f7")
)

var (
	bgOnlyStyle = lipgloss.NewStyle().
			Background(bgColor)

	repoNameStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			Background(bgColor)

	ownerStyle = lipgloss.NewStyle().
			Foreground(fgDimColor).
			Background(bgColor)

	localYesStyle = lipgloss.NewStyle().
			Foreground(greenColor).
			Background(bgColor)

	localNoStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Background(bgColor)

	cursorStyle = lipgloss.NewStyle().
			Foreground(whiteColor).
			Background(bgSelectedColor)

	cursorSepStyle = lipgloss.NewStyle().
			Background(bgSelectedColor)

	localYesCursorStyle = lipgloss.NewStyle().
				Foreground(greenColor).
				Background(bgSelectedColor)

	localNoCursorStyle = lipgloss.NewStyle().
				Foreground(redColor).
				Background(bgSelectedColor)

	headerStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Background(bgColor)

	dimStyle = lipgloss.NewStyle().
			Foreground(fgDimColor).
			Background(bgColor)

	keybindStyle = lipgloss.NewStyle().
			Foreground(yellowColor).
			Background(bgColor)

	inputTextStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Background(bgColor)

	promptStyle = lipgloss.NewStyle().
			Foreground(fgDimColor).
			Background(bgColor)

	queryStyle = lipgloss.NewStyle().
			Foreground(whiteColor).
			Background(bgColor)

	configLabelStyle = lipgloss.NewStyle().
				Foreground(fgDimColor).
				Background(bgColor)

	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			BorderBackground(bgColor).
			Padding(0, 1).
			Background(bgColor)
)
