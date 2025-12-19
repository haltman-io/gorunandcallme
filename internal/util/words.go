package util

import (
	"sort"
	"strings"
)

func NormalizeCSV(in []string) []string {
	out := []string{}
	for _, item := range in {
		for _, p := range strings.Split(item, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, p)
		}
	}
	return out
}

func ParseKVIntMap(s string) map[string]int {
	// "a=1,b=2"
	m := map[string]int{}
	parts := NormalizeCSV([]string{s})
	for _, p := range parts {
		k, v, ok := splitKV(p)
		if !ok {
			continue
		}
		m[k] = atoiSafe(v)
	}
	return m
}

func splitKV(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
		}
	}
	return "", "", false
}

func atoiSafe(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return n
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func SortDedupLines(lines []string) []string {
	set := map[string]struct{}{}
	for _, l := range lines {
		set[l] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Generic merge for structs (simple fields), used by config defaults.
// This is intentionally minimal and does not do deep field-level merging beyond assignment.
func Merge[T any](a T, b T) T {
	// This helper is used only for config overlays where b already contains desired values.
	// At compile time, T is a struct; we simply return b when it has "meaningful" fields,
	// but since Go has no reflection-free generic merge, callers do specific merges elsewhere.
	return b
}
