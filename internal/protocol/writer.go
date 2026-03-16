package protocol

import (
	"fmt"
	"io"
	"sync"
)

// Writer writes protocol messages to a stream
type Writer struct {
	w  io.Writer
	mu sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Send writes a message with type and payload
func (w *Writer) Send(msgType string, payload string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if payload == "" {
		_, err := fmt.Fprintf(w.w, "%s\n", msgType)
		return err
	}
	_, err := fmt.Fprintf(w.w, "%s %s\n", msgType, payload)
	return err
}

// SendRaw writes raw bytes (for file content)
func (w *Writer) SendRaw(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.w.Write(data)
	return err
}
