package event

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Writer struct {
	mu  sync.Mutex
	f   *os.File
	bw  *bufio.Writer
}

type Event struct {
	Time     string            `json:"time"`
	Type     string            `json:"type"`   // line|lifecycle|notify|job
	Stream   string            `json:"stream"` // stdout|stderr
	JobID    string            `json:"job_id,omitempty"`
	Command  string            `json:"command,omitempty"`
	Message  string            `json:"message,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

func New(path string) (*Writer, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &Writer{f: f, bw: bufio.NewWriterSize(f, 64*1024)}, nil
}

func (w *Writer) Write(e Event) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	if e.Time == "" {
		e.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := w.bw.Write(b); err != nil {
		return err
	}
	if err := w.bw.WriteByte('\n'); err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.bw.Flush()
	_ = w.f.Close()
}
