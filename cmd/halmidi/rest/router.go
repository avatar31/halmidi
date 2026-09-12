package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	apihandlers "github.com/avatar31/halmidi/cmd/halmidi/rest/api_handlers"
	"github.com/avatar31/halmidi/pkg/logger"
)

const (
	RequestTimeout = 5 * time.Second
)

var srv = &http.Server{
	WriteTimeout: 15 * time.Second,
	ReadTimeout:  15 * time.Second,
}

func InitRestService(ctx context.Context, port int) {
	gin.SetMode(gin.ReleaseMode)

	log := logger.GetLogger(ctx)
	r := gin.New()
	r.Use(WithLoggingAndPanicRecovery())

	// Prometheus Default Runtime Collector Endpoint
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))

	// Profiler Routes
	debugRouter := r.Group("/debug/pprof")
	addDebugRoutes(debugRouter)

	apiRouter := r.Group("/api/v1")
	addApiRoutes(apiRouter)

	addr := fmt.Sprintf("localhost:%d", port)
	srv.Addr = addr
	srv.Handler = r
	errChan := make(chan error)
	go func(errChan chan<- error) {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}(errChan)

	select {
	case err := <-errChan:
		log.Panic("Failed to start halmidi rest server", zap.Error(err))
	case <-time.After(100 * time.Millisecond):
		// Do a quick health check
		_, err := http.Get(fmt.Sprintf("http://%s/api/v1/health", addr))
		if err != nil {
			log.Panic("Server started but not responding", zap.Error(err))
		}
	}

	log.Info("started halmidi rest server", zap.String("addr", addr))
}

func Close(ctx context.Context) {
	log := logger.GetLogger(ctx)
	log.Info("Shutting down rest service...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Error closing rest service", zap.Error(err))
	}
}

func addApiRoutes(r *gin.RouterGroup) {
	// Health
	r.GET("/health", apihandlers.HealthHandler)
}

// Usage:
//   - go tool pprof -http=:8081 http://127.0.0.1:9090/debug/pprof/heap
//   - go tool pprof http://127.0.0.1:9090/debug/pprof/profile?seconds=30
//   - curl -o trace.out http://127.0.0.1:9090/debug/pprof/trace?seconds=5
//     go tool trace trace.out
func addDebugRoutes(r *gin.RouterGroup) {
	r.GET("/", gin.WrapF(pprof.Index))
	r.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	r.GET("/profile", gin.WrapF(pprof.Profile))
	r.POST("/symbol", gin.WrapF(pprof.Symbol))
	r.GET("/symbol", gin.WrapF(pprof.Symbol))
	r.GET("/trace", gin.WrapF(pprof.Trace))
	r.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
	r.GET("/block", gin.WrapH(pprof.Handler("block")))
	r.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	r.GET("/heap", gin.WrapH(pprof.Handler("heap")))
	r.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
	r.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
}
