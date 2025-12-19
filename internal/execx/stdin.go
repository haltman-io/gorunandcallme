package execx

import (
	"os"
)

func HasPipedStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// If not a char device, stdin is piped or redirected.
	return (fi.Mode() & os.ModeCharDevice) == 0
}
