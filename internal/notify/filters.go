package notify

import (
	"regexp"

	"github.com/haltman-io/gorunandcallme/internal/config"
)

type Filters struct {
	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

func NewFilters(cfg config.NotifyFilterConfig) (*Filters, error) {
	f := &Filters{}
	for _, p := range cfg.Include {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		f.include = append(f.include, r)
	}
	for _, p := range cfg.Exclude {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		f.exclude = append(f.exclude, r)
	}
	return f, nil
}

func (f *Filters) Allow(line string) bool {
	if f == nil {
		return true
	}
	if len(f.include) > 0 {
		ok := false
		for _, r := range f.include {
			if r.MatchString(line) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, r := range f.exclude {
		if r.MatchString(line) {
			return false
		}
	}
	return true
}

type Alerts struct {
	patterns []*regexp.Regexp
	context  int
}

func NewAlerts(cfg config.AlertsConfig) (*Alerts, error) {
	a := &Alerts{context: cfg.IncludeContextLines}
	for _, p := range cfg.Patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		a.patterns = append(a.patterns, r)
	}
	return a, nil
}

func (a *Alerts) Match(line string) bool {
	if a == nil || len(a.patterns) == 0 {
		return false
	}
	for _, r := range a.patterns {
		if r.MatchString(line) {
			return true
		}
	}
	return false
}

func (a *Alerts) ContextLines() int {
	if a == nil {
		return 0
	}
	return a.context
}
