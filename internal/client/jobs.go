package client

import (
	"fmt"
	"strings"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

// RunBackground sends a JOB command and returns the job ID
func RunBackground(s *Session, command string) (string, error) {
	s.writer.Send(protocol.MsgJOB, command)

	msg, err := s.reader.Read()
	if err != nil {
		return "", err
	}

	switch msg.Type {
	case protocol.MsgJOBID:
		return msg.Payload, nil
	case protocol.MsgERR:
		return "", fmt.Errorf("%s", msg.Payload)
	default:
		return "", fmt.Errorf("unexpected response: %s", msg.Type)
	}
}

// ListJobs sends JOBS command and returns the job list output
func ListJobs(s *Session) (string, error) {
	s.writer.Send(protocol.MsgJOBS, "")

	var output strings.Builder
	for {
		msg, err := s.reader.Read()
		if err != nil {
			return output.String(), err
		}
		switch msg.Type {
		case protocol.MsgJOBINF:
			output.WriteString(msg.Payload)
			output.WriteString("\n")
		case protocol.MsgEND:
			return strings.TrimRight(output.String(), "\n"), nil
		case protocol.MsgERR:
			return "", fmt.Errorf("%s", msg.Payload)
		}
	}
}

// GetJobOutput sends JOBOUT command and returns job output
func GetJobOutput(s *Session, jobID string) (string, error) {
	s.writer.Send(protocol.MsgJOBOUT, jobID)

	var output strings.Builder
	for {
		msg, err := s.reader.Read()
		if err != nil {
			return output.String(), err
		}
		switch msg.Type {
		case protocol.MsgOUT:
			output.WriteString(msg.Payload)
			output.WriteString("\n")
		case protocol.MsgEND:
			return strings.TrimRight(output.String(), "\n"), nil
		case protocol.MsgERR:
			return "", fmt.Errorf("%s", msg.Payload)
		}
	}
}

// KillJob sends KILL command
func KillJob(s *Session, jobID string) error {
	s.writer.Send(protocol.MsgKILL, jobID)

	msg, err := s.reader.Read()
	if err != nil {
		return err
	}
	switch msg.Type {
	case protocol.MsgEND:
		return nil
	case protocol.MsgERR:
		return fmt.Errorf("%s", msg.Payload)
	default:
		return nil
	}
}
