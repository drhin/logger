package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
)

var logger *Logger

func InitDevelopment() error {
	var err error
	logger, err = New(
		WithEnv(Development),
		WithServiceName(ServerName),
		WithVersionName(Version),
		WithRequestKey(RequestKey),
		WithUserKey(UserKey),
	)
	return err
}

func InitProduction() error {
	var err error
	logger, err = New(
		WithEnv(Production),
		WithServiceName(ServerName),
		WithVersionName(Version),
		WithRequestKey(RequestKey),
		WithUserKey(UserKey),
		WithRotate(true),
		WithRotatePath("logs/run.log"),
		WithRotateSize(10),
		WithRotateAge(30),
		WithRotateBackups(30),
		WithRotateCompress(false),
	)
	return err
}

func Init(opts ...Option) error {
	logger = &Logger{
		env:            Development,
		serviceName:    ServerName,
		versionName:    Version,
		requestKey:     RequestKey,
		userKey:        UserKey,
		rotate:         false,
		rotatePath:     "logs/run.log",
		rotateSize:     10,
		rotateAge:      7,
		rotateBackups:  10,
		rotateCompress: false,
	}

	for _, opt := range opts {
		opt(logger)
	}

	var err error
	logger, err = logger.newZap()
	return err
}

func With(fields ...zap.Field) *Logger {
	return &Logger{zap: logger.zap.With(fields...)}
}

func WithContext(ctx context.Context) *Logger {
	newLogger := logger.zap

	if requestID, ok := ctx.Value(logger.requestKey).(string); ok {
		newLogger = newLogger.With(zap.String(logger.requestKey, requestID))
	}

	if userID, ok := ctx.Value(logger.userKey).(string); ok {
		newLogger = newLogger.With(zap.String(logger.userKey, userID))
	}

	return &Logger{zap: newLogger}
}

func Debug(msg string, fields ...zap.Field) {
	logger.zap.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	logger.zap.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logger.zap.Warn(msg, fields...)
}

func Error(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	logger.zap.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	logger.zap.Fatal(msg, fields...)
}

func Trace(ctx context.Context, funcName string) func() {
	l := logger.WithContext(ctx)

	startTime := time.Now()
	l.Debug("Starting function", zap.String("function", funcName))

	return func() {
		duration := time.Since(startTime)
		l.Debug("Finished function",
			zap.String("function", funcName),
			zap.Duration("duration", duration),
		)
	}
}

func Sync() error {
	return logger.zap.Sync()
}
