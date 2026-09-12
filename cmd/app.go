package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"github.com/avatar31/halmidi/cmd/rest"
	"github.com/avatar31/halmidi/config"
	cachestore "github.com/avatar31/halmidi/internal/cache_store"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/fileutils"
	osfileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/metrics"
	"github.com/avatar31/halmidi/internal/persistent/storage"
	"github.com/avatar31/halmidi/internal/threads/schedular"
)

const (
	// /var/lock/halmidi/halmidi.lock
	lockFilePath = "/var/lock/" + config.APP_NAME
	lockFile     = lockFilePath + "/" + config.APP_NAME + ".lock"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:    "completion",
		Hidden: true,
		Run:    func(cmd *cobra.Command, args []string) {},
	})

	if config.IsDevEnv() {
		startCmd.Flags().String("dev-config", config.CONFIG_FILE_PATH, "Path to the dev config file")
		startCmd.Flags().Int("port", config.DEFAULT_REST_PORT, "Port to run the server on")
	}
	startCmd.Flags().Uint64("id", 0, "Node ID")
	startCmd.Flags().String("nodename", "", "Node Name")
	startCmd.Flags().String("peers", "", "Comma separated Node IP's. E.g. --peers 1=127.0.0.1:9102,2=127.0.0.1:9112")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

func Init(args []string) {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Start(port int, id uint64, nodename string, peers map[uint64]string) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			logger.GetLogger(ctx).WithField("panic", r).Errorf("PANIC RECOVERED - shutting down gracefully. %s", stack)
			gracefullyCloseAllResources(ctx)
			os.Exit(1)
		}
	}()

	if !config.IsDevEnv() {
		unlock, err := acquireLock()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		defer unlock()
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.InitLogger(ctx)
	log := logger.GetLogger(ctx)
	log.Info("Starting halmidi server")

	// Init metrics
	metrics.CollectRealTimeMetrics(ctx)

	// Init Cache
	cachestore.InitCacheStore(ctx)

	// Init Storage
	storage.Initialize(ctx)

	// Init DB
	dbstore.InitDBStore(ctx, id, nodename, peers)

	// Init rest service
	rest.InitRestService(ctx, port)

	// Srart Cron jobs
	err := startSchedulars(ctx)
	if err != nil {
		log.WithError(err).Panic("Failed to start schedulars")
	}

	waitForShutdown(ctx)
}

func acquireLock() (func(), error) {
	err := osfileutils.CreateDirIfNotExists(lockFilePath)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
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

func startSchedulars(ctx context.Context) error {
	sched := schedular.NewSchedular()

	// Tmp files cleaner
	cleaner := fileutils.NewTmpFilesCleaner(ctx)
	sched.Register(ctx, cleaner.Name, cleaner.Spec, cleaner)

	// // S3 Object Expiry Tracker
	// expiryTracker, err := s3.NewObjectExpiryTracker(ctx)
	// if err != nil {
	// 	return err
	// }
	// sched.Register(ctx, expiryTracker.Name, expiryTracker.Spec, expiryTracker)

	sched.Start(ctx)
	return nil
}

func waitForShutdown(ctx context.Context) {
	<-ctx.Done()
	logger.GetLogger(ctx).Info("Received OS signal to close. Shutting down...")

	gracefullyCloseAllResources(ctx)
}

func gracefullyCloseAllResources(ctx context.Context) {
	// Close rest service gracefully
	rest.Close(ctx)

	// Close DB gracefully
	dbstore.CloseDB(ctx)

	// Close Cache Store
	cachestore.GetCacheStore().Close()

	logger.GetLogger(ctx).Info("Shutdown complete")
}
