package util

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ParseExtendedDuration supports time.ParseDuration plus:
// d (day=24h), w (week=7d), mo (month=30d), y (year=365d).
//
// Examples: 10s, 5m, 1h, 2d, 1w, 3mo, 1y
func ParseExtendedDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty duration")
	}

	// Try native first.
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Extended suffixes.
	type unit struct {
		suffix string
		mult   time.Duration
	}
	units := []unit{
		{"mo", 30 * 24 * time.Hour},
		{"y", 365 * 24 * time.Hour},
		{"w", 7 * 24 * time.Hour},
		{"d", 24 * time.Hour},
	}

	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			num := strings.TrimSuffix(s, u.suffix)
			if num == "" {
				return 0, errors.New("invalid duration number")
			}
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0, err
			}
			return time.Duration(f * float64(u.mult)), nil
		}
	}

	return 0, errors.New("unsupported duration format")
}
