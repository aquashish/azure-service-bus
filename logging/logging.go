package logging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Level string

var (
	logger *zap.SugaredLogger

	levelMap = map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"warn":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"fatal": zapcore.FatalLevel,
	}
)

// init sets up a default logger to ensure we can always get output

type DatadogWriter struct{}

func (d *DatadogWriter) Write(p []byte) (n int, err error) {
	bufCopy := make([]byte, len(p))
	copy(bufCopy, p)
	go func(msg []byte) {
		var m map[string]any
		err := json.Unmarshal(msg, &m) //check if json is fully formed
		if err != nil {
			return
		}

		body := []datadogV2.HTTPLogItem{
			{
				Ddsource: datadog.PtrString("zap"),
				Ddtags: datadog.PtrString(fmt.Sprintf("env:%s,version:%s",
					os.Getenv("DD_ENV"),
					os.Getenv("DD_VERSION"))),
				Message: string(msg),
				Service: datadog.PtrString(os.Getenv("DD_SERVICE")),
			},
		}

		dctx := datadog.NewDefaultContext(context.Background())
		configuration := datadog.NewConfiguration()
		apiClient := datadog.NewAPIClient(configuration)
		api := datadogV2.NewLogsApi(apiClient)

		// ignore errors if we can't get it in
		_, _, _ = api.SubmitLog(dctx, body, *datadogV2.NewSubmitLogOptionalParameters())
	}(bufCopy)

	return os.Stdout.Write(p)
}

func (d *DatadogWriter) Sync() error {
	return os.Stdout.Sync()
}

func LoggerWithTags(ctx context.Context) *zap.SugaredLogger {
	requestId, ok := ctx.Value("requestid").(string)
	if !ok {
		requestId = ""
	}

	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		spanID := span.Context().SpanID()
		traceID := span.Context().TraceID()

		return logger.With(
			"dd.service", os.Getenv("DD_SERVICE"),
			"dd.env", os.Getenv("DD_ENV"),
			"dd.version", os.Getenv("DD_VERSION"),
			"dd.span_id", spanID,
			"dd.trace_id", traceID,
			"request_id", requestId,
		)

	}
	return logger.With(
		"dd.service", os.Getenv("DD_SERVICE"),
		"dd.env", os.Getenv("DD_ENV"),
		"dd.version", os.Getenv("DD_VERSION"),
		"request_id", requestId,
	)
}

func Debug(ctx context.Context, args ...any) {
	LoggerWithTags(ctx).Debug(args...)
}

func Info(ctx context.Context, args ...any) {
	LoggerWithTags(ctx).Info(args...)
}

func Warn(ctx context.Context, args ...any) {
	LoggerWithTags(ctx).Warn(args...)
}

func Error(ctx context.Context, args ...any) {
	LoggerWithTags(ctx).Error(args...)
}

func Fatal(ctx context.Context, args ...any) {
	LoggerWithTags(ctx).Fatal(args...)
}

func Debugf(ctx context.Context, template string, args ...any) {
	LoggerWithTags(ctx).Debugf(template, args...)
}

func Infof(ctx context.Context, template string, args ...any) {
	LoggerWithTags(ctx).Infof(template, args...)
}

func Warnf(ctx context.Context, template string, args ...any) {
	LoggerWithTags(ctx).Warnf(template, args...)
}

func Errorf(ctx context.Context, template string, args ...any) {
	LoggerWithTags(ctx).Errorf(template, args...)
}

func Fatalf(ctx context.Context, template string, args ...any) {
	LoggerWithTags(ctx).Fatalf(template, args...)
}
