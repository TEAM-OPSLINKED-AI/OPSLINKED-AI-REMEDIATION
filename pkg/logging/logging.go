package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

func InitLogger(level string) *zap.Logger {
	logLevel, err := zapcore.ParseLevel(level)
	if err!= nil {
		log.Printf("Invalid log level '%s', defaulting to 'info'", level)
		logLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err!= nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	return logger
}