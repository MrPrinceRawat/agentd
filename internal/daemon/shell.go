package daemon

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

const endMarker = "___AGENTD_END_%d___"

// Shell maintains a persistent bash process
type Shell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	cwd    string
}

func NewShell() (*Shell, error) {
	cmd := exec.Command("bash", "--norc", "--noprofile")
	cmd.Env = append(cmd.Environ(), "PS1=")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Merge stderr into stdout
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	s := &Shell{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReaderSize(stdout, 1024*1024),
		cwd:    "/",
	}

	// Get initial cwd
	if out, _, err := s.Run("pwd"); err == nil {
		s.cwd = strings.TrimSpace(out)
	}

	return s, nil
}

// Run executes a command and returns output, exit code, error
func (s *Shell) Run(command string) (string, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send command + marker
	marker := fmt.Sprintf(endMarker, 0)
	fullCmd := fmt.Sprintf("%s; __ec=$?; pwd > /dev/null; echo \"%s_${__ec}\"; echo \"___CWD_$(pwd)___\"\n", command, marker)

	if _, err := io.WriteString(s.stdin, fullCmd); err != nil {
		return "", -1, err
	}

	// Read until marker
	var output strings.Builder
	exitCode := 0

	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return output.String(), -1, err
		}

		line = strings.TrimRight(line, "\n\r")

		// Check for CWD marker
		if strings.HasPrefix(line, "___CWD_") && strings.HasSuffix(line, "___") {
			s.cwd = strings.TrimSuffix(strings.TrimPrefix(line, "___CWD_"), "___")
			break
		}

		// Check for end marker
		if strings.HasPrefix(line, fmt.Sprintf(endMarker, 0)) {
			parts := strings.Split(line, "_")
			if len(parts) > 0 {
				fmt.Sscanf(parts[len(parts)-1], "%d", &exitCode)
			}
			continue
		}

		output.WriteString(line)
		output.WriteString("\n")
	}

	return output.String(), exitCode, nil
}

// CWD returns current working directory
func (s *Shell) CWD() string {
	return s.cwd
}

// Close kills the shell
func (s *Shell) Close() {
	s.stdin.Close()
	s.cmd.Process.Kill()
	s.cmd.Wait()
}
