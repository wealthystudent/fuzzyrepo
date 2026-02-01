package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrNoEditor      = errors.New("$EDITOR is not set")
	ErrInvalidEditor = errors.New("$EDITOR contains invalid characters")
	ErrCloneFailed   = errors.New("git clone failed")
	ErrAlreadyExists = errors.New("local path already exists")
	ErrInvalidPath   = errors.New("path contains invalid characters")
)

func CloneRepo(repo Repository, config Config) (string, error) {
	if repo.ExistsLocal && repo.LocalPath != "" {
		return repo.LocalPath, ErrAlreadyExists
	}

	destPath := config.GetClonePath(repo.FullName, repo.Name)
	destDir := filepath.Dir(destPath)

	if _, err := os.Stat(destPath); err == nil {
		return destPath, ErrAlreadyExists
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create clone directory: %w", err)
	}

	cloneURL := repo.SSHURL
	if cloneURL == "" {
		cloneURL = fmt.Sprintf("git@github.com:%s/%s.git", repo.Owner, repo.Name)
	}

	cmd := exec.Command("git", "clone", cloneURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}

	return destPath, nil
}

// isValidEditor checks if the editor value is safe to execute.
// Only allows simple command names or absolute paths, no shell metacharacters.
func isValidEditor(editor string) bool {
	// Check for shell metacharacters that could be used for injection
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "\n", "\r", "\\"}
	for _, char := range dangerous {
		if strings.Contains(editor, char) {
			return false
		}
	}
	// Must be non-empty and not start with a dash (could be interpreted as a flag)
	if editor == "" || strings.HasPrefix(editor, "-") {
		return false
	}
	return true
}

func OpenInEditor(path, repoName string) error {
	nvimAddr := os.Getenv("NVIM")
	if nvimAddr != "" {
		return openInNeovim(path, repoName, nvimAddr)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return ErrNoEditor
	}

	if !isValidEditor(editor) {
		return ErrInvalidEditor
	}

	cmd := exec.Command(editor, path)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FUZZYREPO=1")

	return cmd.Run()
}

// escapeLuaString escapes a string for safe use in a Lua string literal.
// This prevents injection attacks when interpolating paths into Lua code.
func escapeLuaString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\'':
			b.WriteString("\\'")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\x00':
			b.WriteString("\\0")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func openInNeovim(path, repoName, nvimAddr string) error {
	// Escape the path for safe use in Lua string
	safePath := escapeLuaString(path)
	nvimCmd := fmt.Sprintf("<C-\\><C-n>:lua package.loaded['fuzzyrepo']=nil; require('fuzzyrepo').open_repo(\"%s\")<CR>", safePath)
	cmd := exec.Command("nvim", "--server", nvimAddr, "--remote-send", nvimCmd)
	return cmd.Run()
}

func CopyToClipboard(text string) {
	encoded := base64Encode(text)
	fmt.Printf("\033]52;c;%s\007", encoded)
}

func base64Encode(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	data := []byte(s)
	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i

		n = uint32(data[i]) << 16
		if remaining > 1 {
			n |= uint32(data[i+1]) << 8
		}
		if remaining > 2 {
			n |= uint32(data[i+2])
		}

		result = append(result, alphabet[(n>>18)&0x3F])
		result = append(result, alphabet[(n>>12)&0x3F])

		if remaining > 1 {
			result = append(result, alphabet[(n>>6)&0x3F])
		} else {
			result = append(result, '=')
		}

		if remaining > 2 {
			result = append(result, alphabet[n&0x3F])
		} else {
			result = append(result, '=')
		}
	}

	return string(result)
}

func EnsureLocal(repo Repository, config Config) (string, error) {
	if repo.ExistsLocal && repo.LocalPath != "" {
		return repo.LocalPath, nil
	}

	return CloneRepo(repo, config)
}

func OpenInBrowser(repo Repository) error {
	url := fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name)
	return openURL(url)
}

func OpenPRs(repo Repository) error {
	url := fmt.Sprintf("https://github.com/%s/%s/pulls", repo.Owner, repo.Name)
	return openURL(url)
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Run()
}
