package notify

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/haltman-io/gorunandcallme/internal/config"
)

type Redactor struct {
	rules []*regexp.Regexp
}

func NewRedactor(cfg config.RedactionConfig) (*Redactor, error) {
	r := &Redactor{}
	if cfg.Defaults {
		for _, p := range defaultRedactions() {
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, err
			}
			r.rules = append(r.rules, re)
		}
	}
	for _, p := range cfg.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		r.rules = append(r.rules, re)
	}
	if cfg.File != "" {
		if err := r.loadFile(cfg.File); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Redactor) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		re, err := regexp.Compile(line)
		if err != nil {
			return err
		}
		r.rules = append(r.rules, re)
	}
	return sc.Err()
}

func (r *Redactor) Apply(s string) string {
	if r == nil || len(r.rules) == 0 {
		return s
	}
	out := s
	for _, re := range r.rules {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

func defaultRedactions() []string {
	return []string{
		`(?i)(api[_-]?key|token|secret|password|passwd|pwd)=\S+`,
		`(?i)(authorization:\s*bearer)\s+\S+`,
		`(?i)(x-api-key:)\s*\S+`,
		`(?i)(client_secret)\s*[:=]\s*\S+`,
	}
}
