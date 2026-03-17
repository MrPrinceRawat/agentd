package client

import (
	"fmt"
	"strings"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

// RunTier2 sends a RUN command via the protocol
func RunTier2(s *Session, command string) (string, int, error) {
	s.writer.Send(protocol.MsgRUN, command)

	var output strings.Builder
	exitCode := 0
	cwd := ""

	for {
		msg, err := s.reader.Read()
		if err != nil {
			return output.String(), -1, err
		}

		switch msg.Type {
		case protocol.MsgOUT:
			output.WriteString(msg.Payload)
			output.WriteString("\n")
		case protocol.MsgEND:
			parts := strings.SplitN(msg.Payload, " ", 2)
			fmt.Sscanf(parts[0], "%d", &exitCode)
			if len(parts) > 1 {
				cwd = parts[1]
			}
			_ = cwd
			return strings.TrimRight(output.String(), "\n"), exitCode, nil
		case protocol.MsgERR:
			return "", -1, fmt.Errorf("%s", msg.Payload)
		}
	}
}

// Run dispatches to the correct tier
func Run(s *Session, command string) (string, int, error) {
	if s.Tier == Tier2 || s.Tier == Tier3 {
		return RunTier2(s, command)
	}
	return RunTier1(s, command)
}
