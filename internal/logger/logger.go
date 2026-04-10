package logger

import (
	"context"
	"github.com/wb-go/wbf/logger"
)

var log logger.Logger

func Init(level, appName, env string) error {
	lvl := logger.InfoLevel
	switch level {
	case "debug":
		lvl = logger.DebugLevel
	case "warn":
		lvl = logger.WarnLevel
	case "error":
		lvl = logger.ErrorLevel
	}
	cfg := &logger.GlobalConfig{
		Level:   lvl,
		AppName: appName,
		Env:     env,
		Stdout:  true,
	}
	var err error
	log = logger.NewZerologAdapter(appName, env, func(c *logger.GlobalConfig) {
		*c = *cfg
	})
	return err
}

func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

func Ctx(ctx context.Context) logger.Logger {
	return log.Ctx(ctx)
}
