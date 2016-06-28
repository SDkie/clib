package logger

import (
	"github.com/Sirupsen/logrus"
)

var log *logrus.Logger

func Init(logLevel logrus.Level) {
	log = logrus.New()
	log.Formatter = new(logrus.JSONFormatter)
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.DebugLevel

	log.Level = logLevel
	Info("Logger Init: Successful")
}

func GetLogger() *logrus.Logger {
	if log != nil {
		return log
	}

	Init(logrus.DebugLevel)
	return log
}

func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Err(args ...interface{}) {
	GetLogger().Error(args...)
}

func Errf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}
