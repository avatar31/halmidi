package rest

import (
	"context"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	s3handlers "github.com/avatar31/halmidi/cmd/halmidi/rest/s3_handlers"
	"github.com/avatar31/halmidi/pkg/logger"
)

func WithLoggingAndPanicRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
		defer cancel()

		traceID := c.GetHeader(s3handlers.AMZ_REQUEST_ID)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Header(s3handlers.AMZ_REQUEST_ID, traceID)

		log := logger.GetLogger(ctx).With(zap.String("traceId", traceID))
		c.Request = c.Request.WithContext(logger.WithContext(ctx))

		fields := make([]zap.Field, 6)
		fields[0] = zap.String("method", c.Request.Method)
		fields[1] = zap.String("path", c.Request.URL.Path)
		fields[2] = zap.String("ip", c.ClientIP())

		if log.Core().Enabled(zap.DebugLevel) {
			str, _ := httputil.DumpRequest(c.Request, false)
			log.Debug("Request received", append(fields[:3], zap.ByteString("request", str))...)
		}

		start := time.Now()
		defer func() {
			fields[3] = zap.Int("status", c.Writer.Status())
			fields[4] = zap.Any("headers", c.Writer.Header())
			fields[5] = zap.Duration("latency", time.Since(start))

			log.Info("Handled Incoming HTTP Request", fields...)
		}()

		defer func() {
			if r := recover(); r != nil {
				log.Error("[PANIC] "+string(debug.Stack()), zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
		}()

		c.Next()
	}
}

// func WithSigV4Authentication() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		userName, err := iam.VerifyRequest(c.Request)
// 		if err == nil {
// 			ctx := context.WithValue(c.Request.Context(), utils.UserNameCtxKey, userName)
// 			c.Request = c.Request.WithContext(ctx)
// 			c.Next()
// 			return
// 		}

// 		s3nativeapihandlers.SendIAMErrorResp(c, err, nil)
// 		c.Abort()
// 	}
// }
