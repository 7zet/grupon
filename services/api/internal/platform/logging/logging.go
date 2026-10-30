// Package logging constructs the structured logger used by every process.
package logging

import (
	"io"
	"log/slog"
)

// New returns a JSON logger configured at the requested minimum level.
func New(output io.Writer, level slog.Leveler) *slog.Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}
