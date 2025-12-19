package util

import "fmt"

func ColorTag(tag string) string {
	// Minimal, modern: only tag gets color.
	switch tag {
	case "INF":
		return fmt.Sprintf("[\x1b[36m%s\x1b[0m]", tag) // cyan
	case "WRN":
		return fmt.Sprintf("[\x1b[33m%s\x1b[0m]", tag) // yellow
	case "ERR":
		return fmt.Sprintf("[\x1b[31m%s\x1b[0m]", tag) // red
	case "DBG":
		return fmt.Sprintf("[\x1b[35m%s\x1b[0m]", tag) // magenta
	case "VRB":
		return fmt.Sprintf("[\x1b[90m%s\x1b[0m]", tag) // gray
	default:
		return fmt.Sprintf("[%s]", tag)
	}
}
