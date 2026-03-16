package daemon

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

// Handle processes incoming messages and sends responses
func Handle(msg protocol.Message, shell *Shell, jobs *JobManager, w *protocol.Writer) {
	switch msg.Type {

	case protocol.MsgRUN:
		output, exitCode, err := shell.Run(msg.Payload)
		if err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
			w.Send(protocol.MsgOUT, line)
		}
		w.Send(protocol.MsgEND, fmt.Sprintf("%d %s", exitCode, shell.CWD()))

	case protocol.MsgREAD:
		path := msg.Payload
		if !CheckRead(path) {
			w.Send(protocol.MsgERR, "permission denied: "+path)
			return
		}
		data, err := ReadFile(path)
		if err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		w.Send(protocol.MsgFILE, fmt.Sprintf("%d", len(data)))
		w.SendRaw(data)
		w.SendRaw([]byte("\n"))
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgWRITE:
		parts := strings.SplitN(msg.Payload, " ", 2)
		if len(parts) < 2 {
			w.Send(protocol.MsgERR, "usage: WRITE <path> <base64_content>")
			return
		}
		path := parts[0]
		if !CheckWrite(path) {
			w.Send(protocol.MsgERR, "permission denied: "+path)
			return
		}
		data, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			w.Send(protocol.MsgERR, "invalid base64: "+err.Error())
			return
		}
		if err := WriteFile(path, data); err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgEDIT:
		parts := strings.SplitN(msg.Payload, " ", 3)
		if len(parts) < 3 {
			w.Send(protocol.MsgERR, "usage: EDIT <path> <old_base64> <new_base64>")
			return
		}
		path := parts[0]
		if !CheckWrite(path) {
			w.Send(protocol.MsgERR, "permission denied: "+path)
			return
		}
		oldText, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			w.Send(protocol.MsgERR, "invalid old_text base64")
			return
		}
		newText, err := base64.StdEncoding.DecodeString(parts[2])
		if err != nil {
			w.Send(protocol.MsgERR, "invalid new_text base64")
			return
		}
		if err := EditFile(path, string(oldText), string(newText)); err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgGLOB:
		matches, err := GlobFiles(msg.Payload)
		if err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		for _, m := range matches {
			if CheckRead(m) {
				w.Send(protocol.MsgMATCH, m)
			}
		}
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgINFO:
		SendInfo(w)

	case protocol.MsgJOB:
		id := jobs.Start(msg.Payload)
		w.Send(protocol.MsgJOBID, fmt.Sprintf("%d", id))

	case protocol.MsgJOBS:
		for _, j := range jobs.List() {
			w.Send(protocol.MsgJOBINF, fmt.Sprintf("%d %s %q", j.ID, j.Status, j.Command))
		}
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgJOBOUT:
		id, _ := strconv.Atoi(msg.Payload)
		out, err := jobs.GetOutput(id)
		if err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			w.Send(protocol.MsgOUT, line)
		}
		w.Send(protocol.MsgEND, "0")

	case protocol.MsgKILL:
		id, _ := strconv.Atoi(msg.Payload)
		if err := jobs.Kill(id); err != nil {
			w.Send(protocol.MsgERR, err.Error())
			return
		}
		w.Send(protocol.MsgEND, "0")

	default:
		w.Send(protocol.MsgERR, "unknown command: "+msg.Type)
	}
}
