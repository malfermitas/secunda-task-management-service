package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"secunda-task-management-service/internal/infrastructure/config"
)

func TestNewJSONFieldsAndLevel(t *testing.T) {
	var buffer bytes.Buffer
	logger, err := newLogger(
		config.Log{Level: "info", Format: "json"},
		"task-service",
		"test",
		"1.2.3",
		zapcore.AddSync(&buffer),
	)
	if err != nil {
		t.Fatalf("newLogger() error = %v", err)
	}

	logger.Debug("hidden")
	logger.Info("created", zap.Int64("task_id", 42))
	logger.Error("failed", zap.Error(errors.New("boom")))

	var info map[string]any
	if err := json.NewDecoder(&buffer).Decode(&info); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if info["msg"] != "created" || info["task_id"] != float64(42) {
		t.Errorf("log = %#v, want created task_id=42", info)
	}
	if info["service"] != "task-service" || info["env"] != "test" || info["version"] != "1.2.3" {
		t.Errorf("identity fields = %#v", info)
	}
	if _, ok := info["ts"]; !ok {
		t.Error("log missing ts")
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	_, err := newLogger(
		config.Log{Level: "verbose", Format: "json"},
		"task-service",
		"test",
		"1.2.3",
		zapcore.AddSync(&bytes.Buffer{}),
	)
	if err == nil {
		t.Fatal("newLogger() error = nil, want parse error")
	}
}
