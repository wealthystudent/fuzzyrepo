package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MessageLevel indicates the severity/type of a status message
type MessageLevel int

const (
	InfoLevel MessageLevel = iota
	WarningLevel
	ErrorLevel
)

// StatusMessage represents a message to display in the UI
type StatusMessage struct {
	Text  string
	Level MessageLevel
}

// Colors for message box
var (
	infoBgColor = lipgloss.Color("#1a1a1a") // Slightly brighter than main bg (#0a0a0a)

	infoMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9999CC")). // Muted purple
			Background(infoBgColor)

	warningMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")). // Orange/Yellow
			Background(infoBgColor)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")). // Red
			Background(infoBgColor)

	infoBoxBgStyle = lipgloss.NewStyle().
			Background(infoBgColor)
)

// levelPrefix returns the prefix for the message level
func (m StatusMessage) levelPrefix() string {
	switch m.Level {
	case WarningLevel:
		return "WARNING"
	case ErrorLevel:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Render returns the styled message box with padding lines above and below
func (m StatusMessage) Render(width int) string {
	if m.Text == "" {
		return ""
	}

	// Format: --- LEVEL: [message] ---
	content := fmt.Sprintf("--- %s: %s ---", m.levelPrefix(), m.Text)

	// Center the message content
	contentWidth := lipgloss.Width(content)
	leftPad := 0
	if width > contentWidth {
		leftPad = (width - contentWidth) / 2
	}

	// Style the message text based on level
	var styledContent string
	switch m.Level {
	case WarningLevel:
		styledContent = warningMsgStyle.Render(content)
	case ErrorLevel:
		styledContent = errorMsgStyle.Render(content)
	default:
		styledContent = infoMsgStyle.Render(content)
	}

	// Build the message line with padding
	msgLine := infoBoxBgStyle.Render(strings.Repeat(" ", leftPad)) + styledContent
	// Pad to full width
	msgLineWidth := lipgloss.Width(msgLine)
	if width > msgLineWidth {
		msgLine += infoBoxBgStyle.Render(strings.Repeat(" ", width-msgLineWidth))
	}

	// Empty line with info box background
	emptyLine := infoBoxBgStyle.Render(strings.Repeat(" ", width))

	// Build box: empty line + message + empty line
	return emptyLine + "\n" + msgLine + "\n" + emptyLine
}

// RenderConfigBox returns a styled info box for the config overlay
// It spans full width with horizontal padding on left/right
// Text wraps if it exceeds the available width
func RenderConfigBox(text string, width int, hPadding int) string {
	if text == "" {
		return ""
	}

	innerWidth := width - (hPadding * 2)
	if innerWidth < 10 {
		innerWidth = width
		hPadding = 0
	}

	// Wrap text to fit within inner width
	wrappedLines := wrapText(text, innerWidth)

	// Build padding strings
	hPad := infoBoxBgStyle.Render(strings.Repeat(" ", hPadding))
	emptyInner := infoBoxBgStyle.Render(strings.Repeat(" ", innerWidth))

	// Empty line: hPad + innerWidth spaces + hPad
	emptyLine := hPad + emptyInner + hPad

	var result []string
	result = append(result, emptyLine)

	// Render each wrapped line, centered
	for _, line := range wrappedLines {
		styledText := infoMsgStyle.Render(line)
		textWidth := lipgloss.Width(styledText)

		// Center the text within the inner width
		leftTextPad := 0
		rightTextPad := innerWidth - textWidth
		if innerWidth > textWidth {
			leftTextPad = (innerWidth - textWidth) / 2
			rightTextPad = innerWidth - textWidth - leftTextPad
		} else {
			rightTextPad = 0
		}

		msgLine := hPad +
			infoBoxBgStyle.Render(strings.Repeat(" ", leftTextPad)) +
			styledText +
			infoBoxBgStyle.Render(strings.Repeat(" ", rightTextPad)) +
			hPad

		result = append(result, msgLine)
	}

	result = append(result, emptyLine)

	return strings.Join(result, "\n")
}

// wrapText wraps text to fit within maxWidth, breaking on word boundaries
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			// First word on line
			if len(word) > maxWidth {
				// Word itself is too long, force break it
				for len(word) > maxWidth {
					lines = append(lines, word[:maxWidth])
					word = word[maxWidth:]
				}
				currentLine = word
			} else {
				currentLine = word
			}
		} else if len(currentLine)+1+len(word) <= maxWidth {
			// Word fits on current line
			currentLine += " " + word
		} else {
			// Word doesn't fit, start new line
			lines = append(lines, currentLine)
			if len(word) > maxWidth {
				// Word itself is too long, force break it
				for len(word) > maxWidth {
					lines = append(lines, word[:maxWidth])
					word = word[maxWidth:]
				}
				currentLine = word
			} else {
				currentLine = word
			}
		}
	}

	// Don't forget the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// IsEmpty returns true if there's no message to display
func (m StatusMessage) IsEmpty() bool {
	return m.Text == ""
}
