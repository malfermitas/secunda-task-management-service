// Package logging builds the application logger from configuration.
package logging

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"secunda-task-management-service/internal/infrastructure/config"
)

// New creates a zap logger based on the log configuration.
func New(cfg config.Log, service, env, version string) (*zap.Logger, error) {
	return newLogger(cfg, service, env, version, zapcore.AddSync(os.Stdout))
}

func newLogger(
	cfg config.Log,
	service, env, version string,
	sink zapcore.WriteSyncer,
) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(strings.ToLower(strings.TrimSpace(cfg.Level)))
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     encodeUTC,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "text") {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, sink, level)
	return zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.Fields(
			zap.String("service", service),
			zap.String("env", env),
			zap.String("version", version),
		),
	), nil
}

func encodeUTC(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(t.UTC().Format(time.RFC3339Nano))
}
