package protocol

// Client → Server message types
const (
	MsgRUN    = "RUN"
	MsgREAD   = "READ"
	MsgWRITE  = "WRITE"
	MsgEDIT   = "EDIT"
	MsgGLOB   = "GLOB"
	MsgINFO   = "INFO"
	MsgJOB    = "JOB"
	MsgJOBS   = "JOBS"
	MsgJOBOUT = "JOBOUT"
	MsgKILL   = "KILL"
)

// Server → Client message types
const (
	MsgOUT    = "OUT"
	MsgEND    = "END"
	MsgERR    = "ERR"
	MsgFILE   = "FILE"
	MsgMATCH  = "MATCH"
	MsgJOBID  = "JOB_ID"
	MsgJOBINF = "JOB"
	MsgOS     = "OS"
	MsgCPU    = "CPU"
	MsgMEM    = "MEM"
	MsgDISK   = "DISK"
	MsgPERM   = "PERM"
)

// Message represents a parsed protocol message
type Message struct {
	Type    string
	Payload string
}
