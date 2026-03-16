package daemon

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/MrPrinceRawat/agentd/internal/protocol"
)

// SendInfo sends machine information to the client
func SendInfo(w *protocol.Writer) {
	// OS
	osName := runtime.GOOS
	osVersion := ""
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osVersion = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}
	w.Send(protocol.MsgOS, fmt.Sprintf("%s %s", osName, osVersion))

	// CPU
	w.Send(protocol.MsgCPU, fmt.Sprintf("%d", runtime.NumCPU()))

	// Memory
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail uint64
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d kB", &total)
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %d kB", &avail)
			}
		}
		used := total - avail
		w.Send(protocol.MsgMEM, fmt.Sprintf("%dMB %dMB", total/1024, used/1024))
	}

	// Disk
	if data, err := os.ReadFile("/proc/mounts"); err == nil {
		_ = data // simplified — would use syscall.Statfs in production
	}
	// Use df output as fallback
	w.Send(protocol.MsgDISK, "check with df -h")

	w.Send(protocol.MsgEND, "0")
}
