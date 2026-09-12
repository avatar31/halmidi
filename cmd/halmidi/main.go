package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/avatar31/dotfs/fileutils"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/avatar31/halmidi/cmd/halmidi/rest"
	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/pkg/logger"
)

const (
	// /var/lock/halmidi/halmidi.lock
	lockFilePath = "/var/lock/" + config.APP_NAME
	lockFile     = lockFilePath + "/" + config.APP_NAME + ".lock"
)

func main() {
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	start()
}

func start() {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			logger.GetLogger(ctx).Error("PANIC RECOVERED - shutting down gracefully. "+string(debug.Stack()),
				zap.Any("panic", r))
			gracefullyCloseAllResources(ctx)
			os.Exit(1)
		}
	}()

	unlock, err := acquireLock()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer unlock()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.InitLogger()
	log := logger.GetLogger(ctx)
	log.Info("Starting halmidi server")

	// Init rest service
	rest.InitRestService(ctx, config.DEFAULT_REST_PORT)

	waitForShutdown(ctx)
}

func acquireLock() (func(), error) {
	err := fileutils.CreateDirIfNotExists(lockFilePath)
	if err != nil {
		return nil, err
	}

	f, err := fileutils.OpenFile(lockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if err == unix.EWOULDBLOCK {
			return nil, fmt.Errorf("halmidi is already running")
		}
		return nil, fmt.Errorf("failed to aquire lock: %w", err)
	}

	unlock := func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}

	return unlock, nil
}

func waitForShutdown(ctx context.Context) {
	<-ctx.Done()
	logger.GetLogger(ctx).Info("Received OS signal to close. Shutting down...")

	gracefullyCloseAllResources(ctx)
}

func gracefullyCloseAllResources(ctx context.Context) {
	// Close rest service gracefully
	rest.Close(ctx)

	logger.GetLogger(ctx).Info("Shutdown complete")
}
