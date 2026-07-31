package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
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

	if logLevel >= logrus.DebugLevel {
		log.SetReportCaller(true)
	}
}

func Get() *logrus.Logger {
	if log == nil {
		Init("info")
	}
	return log
}

// Ctx 提取 Context 中的 OpenTelemetry TraceID/SpanID
func Ctx(ctx context.Context) *logrus.Entry {
	l := Get()
	entry := l.WithContext(ctx)

	if ctx == nil {
		return entry
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		entry = entry.WithField("trace_id", spanCtx.TraceID().String())
	}
	if spanCtx.HasSpanID() {
		entry = entry.WithField("span_id", spanCtx.SpanID().String())
	}

	return entry
}

// WithFields 链式调用入口
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

func Trace(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Trace(msg)
	} else {
		Get().Trace(msg)
	}
}

func Debug(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Debug(msg)
	} else {
		Get().Debug(msg)
	}
}

func Info(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Info(msg)
	} else {
		Get().Info(msg)
	}
}

func Warn(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Warn(msg)
	} else {
		Get().Warn(msg)
	}
}

func Error(msg string, err error, fields ...logrus.Fields) {
	entry := Get().WithError(err)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	if err != nil {
		entry.Error(msg)
	} else {
		Get().Error(msg)
	}
}

func Fatal(msg string, err error, fields ...logrus.Fields) {
	entry := Get().WithError(err)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	if err != nil {
		entry.Fatal(msg)
	} else {
		Get().Fatal(msg)
	}
}
