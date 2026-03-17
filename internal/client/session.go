package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

type Tier int

const (
	Tier1 Tier = 1 // bash + marker
	Tier2 Tier = 2 // agentd protocol (stdin/stdout pipe — legacy)
	Tier3 Tier = 3 // agentd daemon (socket tunnel — persistent)
)

type Session struct {
	Name       string
	Host       HostConfig
	Tier       Tier
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	reader     *protocol.Reader
	writer     *protocol.Writer
	conn       net.Conn    // for tier 3 socket connection
	tunnelCmd  *exec.Cmd   // SSH tunnel process for tier 3
	localSock  string      // local tunnel socket path
}

var (
	sessions    = make(map[string]*Session)
	defaultName string
	sessionMu   sync.Mutex
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
// Tries: tier 3 (daemon socket) → tier 2 (agentd pipe) → tier 1 (bash)
func Connect(name string, host HostConfig) (*Session, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	// Try tier 3 first (agentd daemon via socket tunnel)
	session, err := connectTier3(name, host)
	if err == nil {
		sessions[name] = session
		defaultName = name
		return session, nil
	}

	// Check if user wants to install agentd
	if promptInstall(name, host) {
		// Retry tier 3 after install
		session, err = connectTier3(name, host)
		if err == nil {
			sessions[name] = session
			defaultName = name
			return session, nil
		}
	}

	// Try tier 2 (agentd via stdin/stdout — legacy)
	session, err = connectTier2(name, host)
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

// detectRemoteSocket checks if agentd daemon socket exists on remote
func detectRemoteSocket(host HostConfig) (string, error) {
	// Use a simple test command — check common socket paths
	args := append(sshArgs(host),
		"test", "-S", "/run/user/$(id -u)/agentd.sock",
		"&&", "echo", "/run/user/$(id -u)/agentd.sock",
	)
	cmd := exec.Command("ssh", args...)
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out)), nil
	}

	// Try home directory fallback
	args2 := append(sshArgs(host), "test", "-S", "$HOME/.agentd/agentd.sock",
		"&&", "echo", "$HOME/.agentd/agentd.sock",
	)
	cmd2 := exec.Command("ssh", args2...)
	out2, err := cmd2.Output()
	if err == nil && strings.TrimSpace(string(out2)) != "" {
		return strings.TrimSpace(string(out2)), nil
	}

	return "", fmt.Errorf("agentd socket not found")
}

// connectTier3 establishes an SSH tunnel to the remote agentd Unix socket
func connectTier3(name string, host HostConfig) (*Session, error) {
	// Detect remote socket
	remoteSock, err := detectRemoteSocket(host)
	if err != nil {
		return nil, err
	}

	// Local tunnel socket
	localSock := filepath.Join(os.TempDir(), fmt.Sprintf("agentctl-%s.sock", name))
	os.Remove(localSock) // clean stale

	// Start SSH tunnel: -L localSock:remoteSock -N (no command)
	tunnelArgs := append(sshArgs(host),
		"-L", fmt.Sprintf("%s:%s", localSock, remoteSock),
		"-N", // no remote command
		"-o", "ExitOnForwardFailure=yes",
	)
	tunnelCmd := exec.Command("ssh", tunnelArgs...)
	if err := tunnelCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start tunnel: %v", err)
	}

	// Wait for tunnel to be ready (socket file appears)
	ready := false
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(localSock); err == nil {
			ready = true
			break
		}
	}
	if !ready {
		tunnelCmd.Process.Kill()
		return nil, fmt.Errorf("tunnel socket not ready")
	}

	// Connect to local end of tunnel
	conn, err := net.Dial("unix", localSock)
	if err != nil {
		tunnelCmd.Process.Kill()
		os.Remove(localSock)
		return nil, fmt.Errorf("failed to connect to tunnel: %v", err)
	}

	reader := protocol.NewReader(conn)
	writer := protocol.NewWriter(conn)

	// Wait for READY from agentd
	msg, err := reader.Read()
	if err != nil || msg.Type != "READY" {
		conn.Close()
		tunnelCmd.Process.Kill()
		os.Remove(localSock)
		return nil, fmt.Errorf("agentd daemon not responding")
	}

	return &Session{
		Name:      name,
		Host:      host,
		Tier:      Tier3,
		reader:    reader,
		writer:    writer,
		conn:      conn,
		tunnelCmd: tunnelCmd,
		localSock: localSock,
	}, nil
}

// promptInstall asks user if they want to install agentd on the remote
func promptInstall(name string, host HostConfig) bool {
	fmt.Printf("agentd not installed on %s. Install? [y/n] ", name)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" && answer != "yes" {
		return false
	}

	fmt.Printf("Installing agentd on %s...\n", name)
	args := append(sshArgs(host),
		"bash", "-c",
		`curl -sSL https://raw.githubusercontent.com/MrPrinceRawat/agentd/main/install.sh | bash`,
	)
	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Install failed: %v\n", err)
		return false
	}

	// Wait a moment for service to start
	time.Sleep(2 * time.Second)
	fmt.Println("Install complete.")
	return true
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

	// Clean up based on tier
	if s.conn != nil {
		s.conn.Close()
	}
	if s.tunnelCmd != nil {
		s.tunnelCmd.Process.Kill()
		s.tunnelCmd.Wait()
	}
	if s.localSock != "" {
		os.Remove(s.localSock)
	}
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.cmd != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

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
