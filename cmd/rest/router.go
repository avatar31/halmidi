package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apihandlers "github.com/avatar31/halmidi/cmd/rest/api_handlers"
	s3nativeapihandlers "github.com/avatar31/halmidi/cmd/rest/s3_native_api_handlers"
	"github.com/avatar31/halmidi/internal/logger"
)

const (
	s3RootEndpoint   = "/"
	s3BucketEndpoint = "/:bucket"
	s3ObjectEndpoint = "/:bucket/*object"
)

var (
	srv = &http.Server{
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
)

func addS3NativeAPIRoutes(r *gin.RouterGroup) {
	r.GET(s3RootEndpoint, s3nativeapihandlers.ListBucketsHandler)
	r.POST(s3RootEndpoint, s3nativeapihandlers.IAMActionsHandler)

	r.HEAD(s3BucketEndpoint, s3nativeapihandlers.BucketHeadHandler)
	r.GET(s3BucketEndpoint, s3nativeapihandlers.BucketGetHandler)
	r.PUT(s3BucketEndpoint, s3nativeapihandlers.BucketPutHandler)
	r.DELETE(s3BucketEndpoint, s3nativeapihandlers.BucketDeleteHandler)

	r.HEAD(s3ObjectEndpoint, s3nativeapihandlers.ObjectHeadHandler)
	r.GET(s3ObjectEndpoint, s3nativeapihandlers.ObjectGetHandler)
	r.POST(s3ObjectEndpoint, s3nativeapihandlers.ObjectPostHandler)
	r.PUT(s3ObjectEndpoint, s3nativeapihandlers.ObjectPutHandler)
	r.DELETE(s3ObjectEndpoint, s3nativeapihandlers.ObjectDeleteHandler)
}

func addRoutes(r *gin.RouterGroup) {
	// Halmidi server api routes i.e /api/v1/{buckets|objects|users}

	// Health
	r.GET("/health", apihandlers.HealthHandler)

	// Users
	r.POST("/users", apihandlers.CreateUserHandler)
	r.POST("/users/:userName/accesskey", apihandlers.CreateAccessKeyHandler)
	r.POST("/users/:userName/attach-policy", apihandlers.AttachPolicyHandler)

	// Policies
	r.GET("/policies", apihandlers.ListPolicyHandler)
	r.POST("/policies", apihandlers.CreatePolicyHandler)
}

func InitRestService(ctx context.Context, port int) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(WithLoggingAndPanicRecovery())

	apiRouter := r.Group("/api/v1")
	addRoutes(apiRouter)

	s3Router := r.Group("")
	s3Router.Use(WithSigV4Authentication())
	// TODO: P1: Add rate limiter middleware to s3Router
	// s3Router.Use(WithRateLimiter())
	addS3NativeAPIRoutes(s3Router)

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
		logger.GetLogger(ctx).WithError(err).Panic("Failed to start halmidi rest server")
	case <-time.After(100 * time.Millisecond):
		// Do a quick health check
		_, err := http.Get(fmt.Sprintf("http://%s/api/v1/health", addr))
		if err != nil {
			logger.GetLogger(ctx).WithError(err).Panic("Server started but not responding")
		}
	}

	logger.GetLogger(ctx).Infof("started halmidi rest server in %s", addr)
}

func Close(ctx context.Context) {
	log := logger.GetLogger(ctx)
	log.Info("Shutting down rest service...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("Error closing rest service")
	}
}
