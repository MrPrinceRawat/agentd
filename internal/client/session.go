package client

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

type Tier int

const (
	Tier1 Tier = 1 // bash + marker
	Tier2 Tier = 2 // agentd protocol
)

type Session struct {
	Name   string
	Host   HostConfig
	Tier   Tier
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *protocol.Reader
	writer *protocol.Writer
}

var (
	sessions     = make(map[string]*Session)
	defaultName  string
	sessionMu    sync.Mutex
)

func sshArgs(host HostConfig) []string {
	args := []string{}
	if host.Key != "" {
		key := host.Key
		if strings.HasPrefix(key, "~/") {
			home, _ := os.UserHomeDir()
			key = filepath.Join(home, key[2:])
		}
		args = append(args, "-i", key)
	}
	if host.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", host.Port))
	}
	args = append(args, "-o", "StrictHostKeyChecking=no")
	args = append(args, "-o", "BatchMode=yes")
	target := fmt.Sprintf("%s@%s", host.User, host.Host)
	return append(args, target)
}

// Connect establishes a persistent session to a host
func Connect(name string, host HostConfig) (*Session, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	// Try tier 2 first (agentd)
	session, err := connectTier2(name, host)
	if err == nil {
		sessions[name] = session
		defaultName = name
		return session, nil
	}

	// Fall back to tier 1 (bash)
	session, err = connectTier1(name, host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	sessions[name] = session
	defaultName = name
	return session, nil
}

func connectTier2(name string, host HostConfig) (*Session, error) {
	args := append(sshArgs(host), "agentd")
	cmd := exec.Command("ssh", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	reader := protocol.NewReader(stdout)
	writer := protocol.NewWriter(stdin)

	// Wait for READY
	msg, err := reader.Read()
	if err != nil || msg.Type != "READY" {
		cmd.Process.Kill()
		return nil, fmt.Errorf("agentd not available")
	}

	return &Session{
		Name:   name,
		Host:   host,
		Tier:   Tier2,
		cmd:    cmd,
		stdin:  stdin,
		reader: reader,
		writer: writer,
	}, nil
}

func connectTier1(name string, host HostConfig) (*Session, error) {
	args := append(sshArgs(host), "bash", "--norc", "--noprofile")
	cmd := exec.Command("ssh", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	reader := protocol.NewReader(stdout)
	writer := protocol.NewWriter(stdin)

	return &Session{
		Name:   name,
		Host:   host,
		Tier:   Tier1,
		cmd:    cmd,
		stdin:  stdin,
		reader: reader,
		writer: writer,
	}, nil
}

// GetSession returns a session by name, or default
func GetSession(name string) (*Session, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if name == "" {
		name = defaultName
	}
	s, ok := sessions[name]
	if !ok {
		return nil, fmt.Errorf("no session: %s", name)
	}
	return s, nil
}

// Disconnect closes a session
func Disconnect(name string) error {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if name == "" {
		name = defaultName
	}
	s, ok := sessions[name]
	if !ok {
		return fmt.Errorf("no session: %s", name)
	}
	s.stdin.Close()
	s.cmd.Process.Kill()
	s.cmd.Wait()
	delete(sessions, name)

	if defaultName == name {
		defaultName = ""
		for n := range sessions {
			defaultName = n
			break
		}
	}
	return nil
}

// ListSessions returns all active sessions
func ListSessions() map[string]*Session {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	return sessions
}
