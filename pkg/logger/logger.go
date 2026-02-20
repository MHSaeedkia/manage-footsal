package logger

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
)

const DefaultServiceName = "futsal-bot"

// Config holds logger configuration (LOG_LEVEL, LOG_FORMAT, LOG_OUTPUT from env).
type Config struct {
	Level  string // debug, info, warn, error, fatal
	Format string // json, console
	Output string // stdout, stderr, or file path
}

// New creates a new zap logger from config. If config is nil, uses production defaults.
func New(config *Config, serviceName string) (*zap.Logger, error) {
	if config == nil {
		l, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create production logger: %w", err)
		}
		return l, nil
	}

	var zapConfig zap.Config
	var encoderConfig zapcore.EncoderConfig

	if strings.ToLower(config.Format) == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig = encoderConfig
	}

	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	switch {
	case config.Output != "" && config.Output != "stdout" && config.Output != "stderr":
		zapConfig.OutputPaths = []string{config.Output}
		zapConfig.ErrorOutputPaths = []string{config.Output}
	case config.Output == "stderr":
		zapConfig.OutputPaths = []string{"stderr"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	default:
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return zapLogger.With(zap.String(FieldService, serviceName)), nil
}

func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("%w: %s", ErrInvalidLogLevel, level)
	}
}
