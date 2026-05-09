package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

type Config struct {
	Enabled bool
	Level   string
	Format  string
}

func Init(cfg *Config) {
	if !cfg.Enabled {
		logger = zap.NewNop().Sugar()
		return
	}

	level := parseLevel(cfg.Level)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = l.Sugar()
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

func Debug(args ...interface{}) {
	if logger != nil {
		logger.Debug(args...)
	} else {
		log.Println(args...)
	}
}

func Debugf(template string, args ...interface{}) {
	if logger != nil {
		logger.Debugf(template, args...)
	} else {
		log.Printf(template, args...)
	}
}

func Info(args ...interface{}) {
	if logger != nil {
		logger.Info(args...)
	} else {
		log.Println(args...)
	}
}

func Infof(template string, args ...interface{}) {
	if logger != nil {
		logger.Infof(template, args...)
	} else {
		log.Printf(template, args...)
	}
}

func Warn(args ...interface{}) {
	if logger != nil {
		logger.Warn(args...)
	} else {
		log.Println(args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if logger != nil {
		logger.Warnf(template, args...)
	} else {
		log.Printf(template, args...)
	}
}

func Error(args ...interface{}) {
	if logger != nil {
		logger.Error(args...)
	} else {
		log.Println(args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(template, args...)
	} else {
		log.Printf(template, args...)
	}
}

func Fatal(args ...interface{}) {
	if logger != nil {
		logger.Fatal(args...)
	} else {
		log.Fatal(args...)
	}
}

func Fatalf(template string, args ...interface{}) {
	if logger != nil {
		logger.Fatalf(template, args...)
	} else {
		log.Fatalf(template, args...)
	}
}
