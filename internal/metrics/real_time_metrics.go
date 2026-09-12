package metrics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/logger"
)

func CollectRealTimeMetrics(ctx context.Context) {
	log := logger.GetLogger(ctx)
	log.Info("Starting real-time metrics collection thread")

	errChan := make(chan error)
	go func(errChan chan<- error) {
		pid := int32(os.Getpid())
		proc, err := process.NewProcess(pid)
		if err != nil {
			errChan <- err
		}

		logfilename := filepath.Join(config.GetConfig().Logging.Path, "metrics.log")
		file, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			errChan <- fmt.Errorf("couldn't open metrics log file: %w", err)
		}
		defer func() {
			_ = file.Close()
		}()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Stopping real-time metrics collection.")
				return
			case t := <-ticker.C:
				err := collect(ctx, proc, t, file)
				if err != nil {
					log.WithError(err).Error("Error collecting real-time metrics")
					continue
				}
			}
		}
	}(errChan)

	select {
	case err := <-errChan:
		log.WithError(err).Panic("Real-time metrics collection thread exited with error")
	default:
	}
}

func collect(ctx context.Context, proc *process.Process, t time.Time, logfile *os.File) error {
	log := logger.GetLogger(ctx)

	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		log.WithError(err).Error("Error getting CPU percent")
		return err
	}

	memInfo, err := proc.MemoryInfo()
	if err != nil {
		log.WithError(err).Error("Error getting Memory info")
		return err
	}

	rssMB := float64(memInfo.RSS) / 1024 / 1024
	logLine := fmt.Sprintf("[%s] CPU: %.2f%% | Memory (RSS): %.2f MB\n",
		t.Format(time.RFC3339), cpuPercent, rssMB)

	_, err = logfile.WriteString(logLine)
	return err
}
