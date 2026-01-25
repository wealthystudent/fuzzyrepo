package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	ErrNoEditor      = errors.New("$EDITOR is not set")
	ErrCloneFailed   = errors.New("git clone failed")
	ErrAlreadyExists = errors.New("local path already exists")
)

func CloneRepo(repo Repository, config Config) (string, error) {
	if repo.ExistsLocal && repo.LocalPath != "" {
		return repo.LocalPath, ErrAlreadyExists
	}

	cloneRoot := config.GetCloneRoot()
	destPath := filepath.Join(cloneRoot, repo.Owner, repo.Name)

	if _, err := os.Stat(destPath); err == nil {
		return destPath, ErrAlreadyExists
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
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

func OpenInEditor(path, repoName string) error {
	nvimAddr := os.Getenv("NVIM")
	if nvimAddr != "" {
		return openInNeovim(path, repoName, nvimAddr)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return ErrNoEditor
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func openInNeovim(path, repoName, nvimAddr string) error {
	nvimCmd := fmt.Sprintf("<C-\\><C-n>:tabnew | tcd %s<CR>", path)
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
