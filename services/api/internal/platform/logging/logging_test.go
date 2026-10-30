package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewWritesStructuredJSONAtConfiguredLevel(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := New(&output, slog.LevelWarn)

	logger.Info("filtered")
	logger.Warn("visible", slog.String("component", "test"))

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log record: %v", err)
	}
	if got := record["level"]; got != "WARN" {
		t.Errorf("level = %v, want WARN", got)
	}
	if got := record["msg"]; got != "visible" {
		t.Errorf("msg = %v, want visible", got)
	}
	if got := record["component"]; got != "test" {
		t.Errorf("component = %v, want test", got)
	}
}
