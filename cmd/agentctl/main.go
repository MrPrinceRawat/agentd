package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/MrPrinceRawat/agentd/internal/client"
)

func usage() {
	fmt.Println(`agentctl — remote execution for AI agents

Usage:
  agentctl connect <name>                     Connect to a host
  agentctl run <command>                      Run command on active session
  agentctl run --bg <command>                 Run in background, return job ID
  agentctl run --on <name> <command>          Run on a specific session
  agentctl jobs                               List background jobs
  agentctl job <id>                           Get job output
  agentctl kill <id>                          Kill a background job
  agentctl disconnect [name]                  Close session
  agentctl hosts add <name> <user@host> [-i key] [-p port]
  agentctl hosts list                         List configured hosts
  agentctl hosts remove <name>                Remove a host
  agentctl status                             Show active sessions
  agentctl install <name>                     Install agentd on remote`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "connect":
		cmdConnect()
	case "run":
		cmdRun()
	case "disconnect":
		cmdDisconnect()
	case "hosts":
		cmdHosts()
	case "status":
		cmdStatus()
	case "jobs":
		cmdJobs()
	case "job":
		cmdJob()
	case "kill":
		cmdKill()
	case "install":
		cmdInstall()
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func cmdConnect() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl connect <name>")
		os.Exit(1)
	}

	name := os.Args[2]
	cfg, err := client.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	host, ok := cfg.GetHost(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "host not found: %s\nRun: agentctl hosts add %s user@host -i key\n", name, name)
		os.Exit(1)
	}

	fmt.Printf("Connecting to %s (%s@%s)...\n", name, host.User, host.Host)

	session, err := client.Connect(name, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to %s (tier %d)\n", name, session.Tier)

	// Run session server (blocks — holds the SSH connection alive)
	if err := client.StartSessionServer(name, session); err != nil {
		fmt.Fprintf(os.Stderr, "session server error: %v\n", err)
		os.Exit(1)
	}
}

func cmdRun() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl run [--bg] [--on name] <command>")
		os.Exit(1)
	}

	sessionName := ""
	background := false
	commandStart := 2

	// Parse flags
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--bg":
			background = true
			commandStart = i + 1
		case "--on":
			if i+1 < len(os.Args) {
				sessionName = os.Args[i+1]
				commandStart = i + 2
				i++
			}
		default:
			commandStart = i
			goto done
		}
	}
done:

	command := strings.Join(os.Args[commandStart:], " ")

	// Find which session to use
	if sessionName == "" {
		sessions := client.ListActiveSessions()
		if len(sessions) == 0 {
			fmt.Fprintln(os.Stderr, "not connected. Run: agentctl connect <name> &")
			os.Exit(1)
		}
		sessionName = sessions[0]
	}

	if background {
		output, _, err := client.SendCommand(sessionName, "__BG__ "+command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("job %s\n", output)
		return
	}

	output, exitCode, err := client.SendCommand(sessionName, command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		fmt.Println(output)
	}
	os.Exit(exitCode)
}

func cmdJobs() {
	sessions := client.ListActiveSessions()
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "not connected")
		os.Exit(1)
	}

	output, _, err := client.SendCommand(sessions[0], "__JOBS__")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if output != "" {
		fmt.Println(output)
	} else {
		fmt.Println("No jobs")
	}
}

func cmdJob() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl job <id>")
		os.Exit(1)
	}

	jobID := os.Args[2]
	sessions := client.ListActiveSessions()
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "not connected")
		os.Exit(1)
	}

	output, _, err := client.SendCommand(sessions[0], "__JOBOUT__ "+jobID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if output != "" {
		fmt.Println(output)
	}
}

func cmdKill() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl kill <id>")
		os.Exit(1)
	}

	jobID := os.Args[2]
	sessions := client.ListActiveSessions()
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "not connected")
		os.Exit(1)
	}

	output, _, err := client.SendCommand(sessions[0], "__KILL__ "+jobID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(output)
}

func cmdDisconnect() {
	name := ""
	if len(os.Args) >= 3 {
		name = os.Args[2]
	}

	if name == "" {
		sessions := client.ListActiveSessions()
		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return
		}
		name = sessions[0]
	}

	// Kill the session server process
	pidPath := fmt.Sprintf("%s/agentctl/%s.pid", os.TempDir(), name)
	if data, err := os.ReadFile(pidPath); err == nil {
		var pid int
		fmt.Sscanf(string(data), "%d", &pid)
		if p, err := os.FindProcess(pid); err == nil {
			p.Kill()
		}
	}
	os.Remove(fmt.Sprintf("%s/agentctl/%s.sock", os.TempDir(), name))
	os.Remove(pidPath)
	fmt.Printf("Disconnected from %s\n", name)
}

func cmdHosts() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl hosts add|list|remove")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "add":
		cmdHostsAdd()
	case "list":
		cmdHostsList()
	case "remove":
		cmdHostsRemove()
	default:
		fmt.Fprintf(os.Stderr, "unknown hosts command: %s\n", os.Args[2])
	}
}

func cmdHostsAdd() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "usage: agentctl hosts add <name> <user@host> [-i key] [-p port]")
		os.Exit(1)
	}

	name := os.Args[3]
	target := os.Args[4]

	parts := strings.SplitN(target, "@", 2)
	if len(parts) != 2 {
		fmt.Fprintln(os.Stderr, "target must be user@host")
		os.Exit(1)
	}

	host := client.HostConfig{
		User: parts[0],
		Host: parts[1],
	}

	// Parse optional flags
	for i := 5; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-i":
			if i+1 < len(os.Args) {
				host.Key = os.Args[i+1]
				i++
			}
		case "-p":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &host.Port)
				i++
			}
		}
	}

	cfg, _ := client.LoadConfig()
	cfg.AddHost(name, host)
	if err := client.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Added %s (%s@%s)\n", name, host.User, host.Host)
}

func cmdHostsList() {
	cfg, err := client.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Hosts) == 0 {
		fmt.Println("No hosts configured. Run: agentctl hosts add <name> user@host -i key")
		return
	}

	for name, host := range cfg.Hosts {
		key := ""
		if host.Key != "" {
			key = fmt.Sprintf(" (key: %s)", host.Key)
		}
		fmt.Printf("  %s → %s@%s%s\n", name, host.User, host.Host, key)
	}
}

func cmdHostsRemove() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: agentctl hosts remove <name>")
		os.Exit(1)
	}

	name := os.Args[3]
	cfg, _ := client.LoadConfig()
	cfg.RemoveHost(name)
	client.SaveConfig(cfg)
	fmt.Printf("Removed %s\n", name)
}

func cmdStatus() {
	sessions := client.ListActiveSessions()
	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return
	}
	for _, name := range sessions {
		status, err := client.GetSessionStatus(name)
		if err != nil {
			fmt.Printf("  %s → error: %v\n", name, err)
		} else {
			fmt.Printf("  %s → %s\n", name, status)
		}
	}
}

func cmdInstall() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl install <name>")
		os.Exit(1)
	}
	// TODO: detect remote arch, scp correct binary
	fmt.Println("Install not yet implemented. SCP the agentd binary manually:")
	fmt.Println("  scp bin/agentd-linux-amd64 user@host:/usr/local/bin/agentd")
}
