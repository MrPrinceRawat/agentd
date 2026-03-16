package protocol

import (
	"bufio"
	"io"
	"strings"
)

// Reader reads protocol messages from a stream
type Reader struct {
	scanner *bufio.Scanner
	raw     io.Reader
}

func NewReader(r io.Reader) *Reader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	return &Reader{scanner: scanner, raw: r}
}

// Read returns the next message or error
func (r *Reader) Read() (Message, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return Message{}, err
		}
		return Message{}, io.EOF
	}

	line := r.scanner.Text()
	parts := strings.SplitN(line, " ", 2)

	msg := Message{Type: parts[0]}
	if len(parts) > 1 {
		msg.Payload = parts[1]
	}

	return msg, nil
}

// ReadRawBytes reads exactly n bytes from the underlying reader
func (r *Reader) ReadRawBytes(n int) ([]byte, error) {
	// For raw byte reads, we need the underlying reader
	// This is used for file content transfer
	buf := make([]byte, n)
	_, err := io.ReadFull(r.raw, buf)
	return buf, err
}
