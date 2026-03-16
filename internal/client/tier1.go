package client

import (
	"fmt"
	"io"
	"strings"
)

const tier1Marker = "___AGENTCTL_END_"

// RunTier1 sends a command with marker, reads until marker
func RunTier1(s *Session, command string) (string, int, error) {
	// Send command + marker
	fullCmd := fmt.Sprintf("%s; echo \"%s$?___\"\n", command, tier1Marker)
	if _, err := io.WriteString(s.stdin, fullCmd); err != nil {
		return "", -1, err
	}

	// Read until marker
	var output strings.Builder
	exitCode := 0

	for {
		msg, err := s.reader.Read()
		if err != nil {
			return output.String(), -1, err
		}

		line := msg.Type
		if msg.Payload != "" {
			line = msg.Type + " " + msg.Payload
		}

		// Check for marker
		if strings.HasPrefix(line, tier1Marker) {
			codeStr := strings.TrimPrefix(line, tier1Marker)
			codeStr = strings.TrimSuffix(codeStr, "___")
			fmt.Sscanf(codeStr, "%d", &exitCode)
			break
		}

		output.WriteString(line)
		output.WriteString("\n")
	}

	return strings.TrimRight(output.String(), "\n"), exitCode, nil
}
