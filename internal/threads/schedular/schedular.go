package schedular

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/avatar31/halmidi/internal/logger"
)

type JobHandler interface {
	SchedHandler(ctx context.Context)
}

type Schedular struct {
	c *cron.Cron
}

func NewSchedular() Schedular {
	return Schedular{
		c: cron.New(cron.WithSeconds()),
	}
}

func (sc Schedular) Register(ctx context.Context, name string, spec string, jobHandler JobHandler) {
	log := logger.GetLogger(ctx).WithField("schedular", name)
	log.Infof("Registering schedular")
	_, err := sc.c.AddFunc(spec, func() { jobHandler.SchedHandler(ctx) })
	if err != nil {
		// TODO: P0: Handle this error properly, maybe by returning it to the caller instead of just logging it.
		log.WithError(err).Error("Failed to register schedular")
	}
}

func (sc Schedular) Start(ctx context.Context) {
	log := logger.GetLogger(ctx)

	log.Info("Starting all schedulars")
	sc.c.Start()

	<-ctx.Done()

	log.Info("Stopping all schedulars")
	stopCtx := sc.c.Stop()
	<-stopCtx.Done() // wait for running jobs to complete
}
