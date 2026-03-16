package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func socketDir() string {
	return filepath.Join(os.TempDir(), "agentctl")
}

func socketPath(name string) string {
	return filepath.Join(socketDir(), name+".sock")
}

// StartSessionServer runs a local Unix socket server that holds the SSH session
// and accepts run commands from other agentctl processes
func StartSessionServer(name string, session *Session) error {
	os.MkdirAll(socketDir(), 0700)

	sockPath := socketPath(name)
	os.Remove(sockPath) // clean up stale socket

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}

	// Write PID file
	pidPath := filepath.Join(socketDir(), name+".pid")
	os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0600)

	fmt.Printf("Session server running on %s\n", sockPath)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, session)
	}
}

func handleConn(conn net.Conn, session *Session) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	command := strings.TrimSpace(line)
	if command == "__STATUS__" {
		fmt.Fprintf(conn, "tier %d %s@%s\n", session.Tier, session.Host.User, session.Host.Host)
		return
	}

	output, exitCode, err := Run(session, command)
	if err != nil {
		fmt.Fprintf(conn, "ERR %s\n", err.Error())
		return
	}

	fmt.Fprintf(conn, "EXIT %d\n", exitCode)
	fmt.Fprintf(conn, "%s\n", output)
	fmt.Fprintf(conn, "___DONE___\n")
}

// SendCommand connects to an existing session server and sends a command
func SendCommand(name string, command string) (string, int, error) {
	sockPath := socketPath(name)

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return "", -1, fmt.Errorf("not connected to %s", name)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\n", command)

	reader := bufio.NewReader(conn)

	// Read exit code line
	exitLine, err := reader.ReadString('\n')
	if err != nil {
		return "", -1, err
	}
	exitLine = strings.TrimSpace(exitLine)

	if strings.HasPrefix(exitLine, "ERR ") {
		return "", -1, fmt.Errorf("%s", strings.TrimPrefix(exitLine, "ERR "))
	}

	exitCode := 0
	fmt.Sscanf(strings.TrimPrefix(exitLine, "EXIT "), "%d", &exitCode)

	// Read output until done marker
	var output strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "___DONE___" {
			break
		}
		output.WriteString(line)
		output.WriteString("\n")
	}

	return strings.TrimRight(output.String(), "\n"), exitCode, nil
}

// GetSessionStatus checks if a session server is running
func GetSessionStatus(name string) (string, error) {
	conn, err := net.Dial("unix", socketPath(name))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "__STATUS__\n")
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// ListActiveSessions returns names of sessions with running servers
func ListActiveSessions() []string {
	entries, err := os.ReadDir(socketDir())
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sock") {
			name := strings.TrimSuffix(e.Name(), ".sock")
			if _, err := GetSessionStatus(name); err == nil {
				names = append(names, name)
			} else {
				// Stale socket, clean up
				os.Remove(socketPath(name))
				os.Remove(filepath.Join(socketDir(), name+".pid"))
			}
		}
	}
	return names
}
