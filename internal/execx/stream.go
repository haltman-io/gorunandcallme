package execx

import (
	"bytes"
	"io"
)

// StreamAssembler converts a byte stream into lines, with support for carriage-return progress rewriting.
// It emits lines on '\n'. When '\r' occurs, the current line buffer is reset.
//
// This helps clean progress bars/spinners (common in security tools) that redraw the same line.
type StreamAssembler struct {
	buf bytes.Buffer
}

func (s *StreamAssembler) Feed(p []byte, emit func(line string)) {
	for _, b := range p {
		switch b {
		case '\r':
			s.buf.Reset()
		case '\n':
			emit(s.buf.String())
			s.buf.Reset()
		default:
			_ = s.buf.WriteByte(b)
		}
	}
}

func (s *StreamAssembler) Flush(emit func(line string)) {
	if s.buf.Len() > 0 {
		emit(s.buf.String())
		s.buf.Reset()
	}
}

type PrefixWriter struct {
	dst         io.Writer
	prefix      []byte
	atLineStart bool
}

func NewPrefixWriter(dst io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		dst:         dst,
		prefix:      []byte(prefix),
		atLineStart: true,
	}
}

func (w *PrefixWriter) Write(p []byte) (int, error) {
	if len(w.prefix) == 0 {
		return w.dst.Write(p)
	}

	consumed := 0
	for i := 0; i < len(p); i++ {
		if w.atLineStart {
			if _, err := w.dst.Write(w.prefix); err != nil {
				return consumed, err
			}
			w.atLineStart = false
		}
		if _, err := w.dst.Write([]byte{p[i]}); err != nil {
			return consumed, err
		}
		consumed++
		if p[i] == '\n' {
			w.atLineStart = true
		}
	}
	return consumed, nil
}
