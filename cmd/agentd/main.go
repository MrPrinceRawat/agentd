package main

import (
	"fmt"
	"io"
	"os"

	"github.com/MrPrinceRawat/agentd/internal/daemon"
	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

func main() {
	// Load permissions
	daemon.LoadPermissions()

	// Create persistent shell
	shell, err := daemon.NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start shell: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	// Create job manager
	jobs := daemon.NewJobManager()

	// Protocol reader/writer on stdin/stdout
	reader := protocol.NewReader(os.Stdin)
	writer := protocol.NewWriter(os.Stdout)

	// Signal ready
	writer.Send("READY", "agentd")

	// Main loop
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
