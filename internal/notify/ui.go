package notify

// UI is a minimal UI/log contract for the notify package.
// It must not depend on internal/app to avoid import cycles.
type UI interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)

	Verbosef(format string, args ...any)
	Debugf(format string, args ...any)
}
