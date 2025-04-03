package logger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Development = "development"
	Production  = "production"
	RequestKey  = "request_id"
	UserKey     = "user_id"
	ServerName  = "rhino_logger"
	Version     = "v1.0.0"
)

type Logger struct {
	// env 服务的环境, development or production
	env string
	// serviceName 服务名, 例如：rhino_logger
	serviceName string
	// versionName 服务版本, 例如：v1.0.0
	versionName string
	// requestKey 请求上下文的请求ID名称, 例如：request_id
	requestKey string
	// userKey 请求上下文的用户ID名称, 例如：user_id
	userKey string
	// rotate 是否开启日志文件分割, 默认不开启
	rotate bool
	// rotatePath 日志文件的路径, 默认是当前目录下的logs文件夹, 例如：./logs/run.log
	rotatePath string
	// rotateSize 日志文件的大小, 默认是10MB
	rotateSize int
	// rotateAge 日志文件的保留时间, 默认是7天
	rotateAge int
	// rotateBackups 日志文件的备份数量, 默认是10个
	rotateBackups int
	// rotateCompress 是否压缩日志文件, 默认是不压缩
	rotateCompress bool
	// zap 日志库的实例
	zap *zap.Logger
}

type Option func(*Logger)

func WithEnv(env string) Option {
	return func(l *Logger) {
		l.env = env
	}
}

func WithServiceName(serviceName string) Option {
	return func(l *Logger) {
		l.serviceName = serviceName
	}
}

func WithVersionName(versionName string) Option {
	return func(l *Logger) {
		l.versionName = versionName
	}
}

func WithRequestKey(requestKey string) Option {
	return func(l *Logger) {
		l.requestKey = requestKey
	}
}

func WithUserKey(userKey string) Option {
	return func(l *Logger) {
		l.userKey = userKey
	}
}

func WithRotate(rotate bool) Option {
	return func(l *Logger) {
		l.rotate = rotate
	}
}

func WithRotatePath(rotatePath string) Option {
	return func(l *Logger) {
		l.rotatePath = rotatePath
	}
}

func WithRotateSize(rotateSize int) Option {
	return func(l *Logger) {
		l.rotateSize = rotateSize
	}
}

func WithRotateAge(rotateAge int) Option {
	return func(l *Logger) {
		l.rotateAge = rotateAge
	}
}

func WithRotateBackups(rotateBackups int) Option {
	return func(l *Logger) {
		l.rotateBackups = rotateBackups
	}
}

func WithRotateCompress(rotateCompress bool) Option {
	return func(l *Logger) {
		l.rotateCompress = rotateCompress
	}
}

func NewDevelopment() (*Logger, error) {
	return New(
		WithEnv(Development),
		WithServiceName(ServerName),
		WithVersionName(Version),
		WithRequestKey(RequestKey),
		WithUserKey(UserKey),
	)
}

func New(opts ...Option) (*Logger, error) {
	l := &Logger{
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
		opt(l)
	}

	return l.newZap()
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := l.zap

	if requestID, ok := ctx.Value(l.requestKey).(string); ok {
		newLogger = newLogger.With(zap.String(l.requestKey, requestID))
	}

	if userID, ok := ctx.Value(l.userKey).(string); ok {
		newLogger = newLogger.With(zap.String(l.userKey, userID))
	}

	return &Logger{zap: newLogger}
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.zap.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

func (l *Logger) Trace(ctx context.Context, funcName string) func() {
	logger := l.WithContext(ctx)

	startTime := time.Now()
	logger.Debug("Starting function", zap.String("function", funcName))

	return func() {
		duration := time.Since(startTime)
		logger.Debug("Finished function",
			zap.String("function", funcName),
			zap.Duration("duration", duration),
		)
	}
}

func (l *Logger) Sync() error {
	return l.zap.Sync()
}

func (l *Logger) newZap() (*Logger, error) {
	zapFields := []zap.Field{
		zap.String("env", l.env),
	}
	if l.serviceName != "" {
		zapFields = append(zapFields, zap.String("service", l.serviceName))
	}
	if l.versionName != "" {
		zapFields = append(zapFields, zap.String("version", l.versionName))
	}

	if l.env == Development {
		l, err := l.newZapDevelopment(zapFields...)
		if err != nil {
			return nil, err
		}
		return &Logger{zap: l}, nil
	} else if l.env == Production {
		l, err := l.newZapProduction(zapFields...)
		if err != nil {
			return nil, err
		}
		return &Logger{zap: l}, nil
	}

	return nil, errors.New("invalid environment,  use development or production")
}

func (l *Logger) newZapDevelopment(fields ...zap.Field) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = formatTime

	if l.rotate {
		logWriter := l.getLogWriter()
		encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
		fileCore := zapcore.NewCore(encoder, logWriter, config.Level)

		consoleWriter := zapcore.Lock(os.Stdout)
		consoleCore := zapcore.NewCore(encoder, consoleWriter, config.Level)
		core := zapcore.NewTee(fileCore, consoleCore)

		logger := zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.Fields(
				fields...,
			),
		)
		return logger, nil
	} else {
		encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
		consoleWriter := zapcore.Lock(os.Stdout)
		core := zapcore.NewCore(encoder, consoleWriter, config.Level)

		logger := zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.Fields(
				fields...,
			),
		)
		return logger, nil
	}
}

func (l *Logger) newZapProduction(fields ...zap.Field) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = formatTime

	if l.rotate {
		logWriter := l.getLogWriter()
		encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
		core := zapcore.NewCore(encoder, logWriter, config.Level)

		logger := zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.Fields(
				fields...,
			),
		)
		return logger, nil
	} else {
		err := checkFile(l.rotatePath)
		if err != nil {
			return nil, err
		}

		file, err := os.OpenFile(l.rotatePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}

		encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
		core := zapcore.NewCore(encoder, zapcore.AddSync(file), config.Level)

		logger := zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.Fields(
				fields...,
			),
		)
		return logger, nil
	}
}

func (l *Logger) getLogWriter() zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   l.rotatePath,     // 日志文件的位置
		MaxSize:    l.rotateSize,     // 在进行切割之前, 日志文件的最大大小（以MB为单位）
		MaxBackups: l.rotateBackups,  // 保留旧文件的最大个数
		MaxAge:     l.rotateAge,      // 保留旧文件的最大天数
		Compress:   l.rotateCompress, // 是否压缩/归档旧文件
	})
}

func formatTime(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
	pae.AppendString(t.Format("2006-01-02 15:04:05.000Z0700"))
}

func checkFile(path string) error {
	if isExist(path) {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if !createFile(path) {
		return errors.New("create file failed")
	}

	return nil
}

// isExist 检查一个文件或目录是否存在
func isExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

// createFile 在路径中创建一个文件
func createFile(path string) bool {
	file, err := os.Create(path)
	if err != nil {
		return false
	}

	defer file.Close()
	return true
}
