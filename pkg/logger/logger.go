package logger

import (
	"context"
	"log"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 是日誌介面
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
}

// loggerImpl 是 Logger 介面的實作
type loggerImpl struct {
	logger *zap.Logger
}

func (l *loggerImpl) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

func (l *loggerImpl) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *loggerImpl) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *loggerImpl) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

func (l *loggerImpl) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *loggerImpl) With(fields ...zap.Field) Logger {
	return &loggerImpl{logger: l.logger.With(fields...)}
}

// NewLogger 創建一個新的日誌記錄器
func NewLogger() (Logger, error) {
	// 創建基本的 encoder 配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 配置日誌核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(zapcore.InfoLevel),
	)

	// 創建日誌記錄器
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &loggerImpl{logger: logger}, nil
}

// ProvideLogger 提供 Logger 實例，用於 fx
func ProvideLogger(lc fx.Lifecycle) (Logger, error) {
	logger, err := NewLogger()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Logger initialized successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Closing logger...")
			return nil
		},
	})

	return logger, nil
}

// Module 創建 fx 模組，包含所有日誌相關組件
var Module = fx.Module("logger",
	fx.Provide(
		ProvideLogger,
	),
)
