package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads a file and returns its contents
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes content to a file
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// EditFile replaces old text with new text in a file
func EditFile(path string, oldText, newText string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return fmt.Errorf("old text not found in %s", path)
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	return os.WriteFile(path, []byte(newContent), 0644)
}

// GlobFiles returns files matching a pattern
func GlobFiles(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
