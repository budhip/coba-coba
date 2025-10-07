package log

import (
	"context"

	xlog "bitbucket.org/Amartha/go-x/log"
)

func LogJob(ctx context.Context, jobName, version, date string, err error) {
	field := []xlog.Field{
		xlog.String("job-name", jobName),
		xlog.String("version", version),
		xlog.String("execution-date", date),
	}
	if err != nil {
		field = append(field, xlog.String("status", "fail"), xlog.Err(err))
		xlog.Warn(ctx, "[JOB]", field...)
	} else {
		field = append(field, xlog.String("status", "success"))
		xlog.Info(ctx, "[JOB]", field...)
	}
}
