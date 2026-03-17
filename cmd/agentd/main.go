package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/MrPrinceRawat/agentd/internal/daemon"
	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

func main() {
	socketMode := flag.Bool("socket", false, "Listen on Unix socket instead of stdin/stdout")
	socketPath := flag.String("socket-path", "", "Custom socket path (default: auto-detect)")
	flag.Parse()

	// Load permissions
	daemon.LoadPermissions()

	if *socketMode {
		runSocket(*socketPath)
	} else {
		runStdio()
	}
}

// runStdio runs the original stdin/stdout protocol loop (for testing / pipe usage)
func runStdio() {
	shell, err := daemon.NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start shell: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	jobs := daemon.NewJobManager()
	reader := protocol.NewReader(os.Stdin)
	writer := protocol.NewWriter(os.Stdout)

	writer.Send("READY", "agentd")

	for {
		msg, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			writer.Send(protocol.MsgERR, err.Error())
			continue
		}
		daemon.Handle(msg, shell, jobs, writer)
	}
}

// runSocket runs agentd as a persistent daemon listening on a Unix socket
func runSocket(customPath string) {
	sockPath := customPath
	if sockPath == "" {
		sockPath = defaultSocketPath()
	}

	// Clean stale socket
	if _, err := os.Stat(sockPath); err == nil {
		// Check if another agentd is already listening
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			conn.Close()
			fmt.Fprintf(os.Stderr, "agentd already running on %s\n", sockPath)
			os.Exit(1)
		}
		// Stale socket — remove it
		os.Remove(sockPath)
	}

	// Ensure parent directory exists
	os.MkdirAll(filepath.Dir(sockPath), 0700)

	// Listen
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen on %s: %v\n", sockPath, err)
		os.Exit(1)
	}
	defer ln.Close()
	defer os.Remove(sockPath)

	// Set socket permissions to 0700
	os.Chmod(sockPath, 0700)

	// Shared shell and job manager across all connections
	shell, err := daemon.NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start shell: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	jobs := daemon.NewJobManager()

	fmt.Fprintf(os.Stderr, "agentd listening on %s\n", sockPath)

	// Accept connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, shell, jobs)
	}
}

func handleConn(conn net.Conn, shell *daemon.Shell, jobs *daemon.JobManager) {
	defer conn.Close()

	reader := protocol.NewReader(conn)
	writer := protocol.NewWriter(conn)

	writer.Send("READY", "agentd")

	for {
		msg, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return
			}
			writer.Send(protocol.MsgERR, err.Error())
			continue
		}
		daemon.Handle(msg, shell, jobs, writer)
	}
}

func defaultSocketPath() string {
	// Try XDG runtime dir first: /run/user/<uid>/agentd.sock
	uid := os.Getuid()
	xdgPath := fmt.Sprintf("/run/user/%d/agentd.sock", uid)
	if dir := filepath.Dir(xdgPath); dirExists(dir) {
		return xdgPath
	}

	// Fallback: ~/.agentd/agentd.sock
	u, err := user.Current()
	if err != nil {
		return "/tmp/agentd.sock"
	}
	return filepath.Join(u.HomeDir, ".agentd", "agentd.sock")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Ensure we clean up the socket on signals
func init() {
	// Ignore SIGHUP so we survive terminal close
	// (systemd sends SIGTERM for proper shutdown)
	signal := syscall.Signal(1) // SIGHUP
	_ = signal
}
