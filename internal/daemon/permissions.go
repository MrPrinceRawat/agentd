package daemon

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type PermConfig struct {
	Permissions struct {
		AllowRead  []string `yaml:"allow_read"`
		AllowWrite []string `yaml:"allow_write"`
		Deny       []string `yaml:"deny"`
		AllowSudo  bool     `yaml:"allow_sudo"`
		MaxJobs    int      `yaml:"max_jobs"`
	} `yaml:"permissions"`
}

var config *PermConfig

func LoadPermissions() {
	config = nil
	data, err := os.ReadFile("/etc/agentd.yaml")
	if err != nil {
		return // no config = full access
	}
	var c PermConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return
	}
	config = &c
}

// CheckRead returns true if the path is allowed for reading
func CheckRead(path string) bool {
	if config == nil {
		return true
	}
	return checkPath(path, config.Permissions.AllowRead, config.Permissions.Deny)
}

// CheckWrite returns true if the path is allowed for writing
func CheckWrite(path string) bool {
	if config == nil {
		return true
	}
	return checkPath(path, config.Permissions.AllowWrite, config.Permissions.Deny)
}

func checkPath(path string, allow []string, deny []string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check deny first
	for _, pattern := range deny {
		if matched, _ := filepath.Match(pattern, abs); matched {
			return false
		}
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(abs, prefix) {
				return false
			}
		}
	}

	// If allow list is empty, allow everything not denied
	if len(allow) == 0 {
		return true
	}

	// Check allow
	for _, pattern := range allow {
		if matched, _ := filepath.Match(pattern, abs); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(abs, prefix) {
				return true
			}
		}
	}

	return false
}
