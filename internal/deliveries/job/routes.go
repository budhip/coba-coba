package job

import (
	"context"
	"errors"
	"time"

	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/log"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	v1file "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/job/v1/file"
	v1report "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/job/v1/report"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/google/uuid"
)

type JobRoutes map[string]map[string]func(ctx context.Context, date time.Time, flag flag.Job) error

type Job struct {
	Routes JobRoutes
}

func New(cfg config.Config, srv *services.Services) *Job {
	v1group := "v1"

	jobRoutes := map[string]map[string]func(ctx context.Context, date time.Time, flag flag.Job) error{
		v1group: v1report.Routes(srv.Transaction, services.NewReconBalanceService(srv)),
		v1group: v1file.Routes(srv.File),
		// add other version routes
	}

	return &Job{jobRoutes}
}

func (j *Job) Start(ctx context.Context, flag flag.Job) {
	if fn, ok := j.Routes[flag.Version][flag.JobName]; ok {
		var (
			runningDate time.Time
			err         error
		)
		ctx = ctxdata.Sets(ctx, ctxdata.SetCorrelationId(uuid.New().String()))

		defer func() {
			log.LogJob(ctx, flag.JobName, flag.Version, flag.Date, err)
		}()

		if flag.Date != "" {
			runningDate, err = common.ParseStringToDatetime(common.DateFormatYYYYMMDD, flag.Date)
			if err != nil {
				return
			}
		}
		if err = fn(ctx, runningDate, flag); err != nil {
			return
		}
	} else {
		log.LogJob(ctx, flag.JobName, flag.Version, flag.Date, errors.New("invalid version or job name"))
	}
}
