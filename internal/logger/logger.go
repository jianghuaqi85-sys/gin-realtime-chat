package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func Init(level string) {
	log = logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)
}

func Get() *logrus.Logger {
	return log
}

func Trace(msg string, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).Trace(msg)
	} else {
		log.Trace(msg)
	}
}

func Debug(msg string, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).Debug(msg)
	} else {
		log.Debug(msg)
	}
}

func Info(msg string, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).Info(msg)
	} else {
		log.Info(msg)
	}
}

func Warn(msg string, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).Warn(msg)
	} else {
		log.Warn(msg)
	}
}

func Error(msg string, err error, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).WithError(err).Error(msg)
	} else {
		log.WithError(err).Error(msg)
	}
}

func Fatal(msg string, err error, fields ...logrus.Fields) {
	if fields != nil {
		log.WithFields(fields[0]).WithError(err).Fatal(msg)
	} else {
		log.WithError(err).Fatal(msg)
	}
}
