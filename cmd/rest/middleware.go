package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	s3nativeapihandlers "github.com/avatar31/halmidi/cmd/rest/s3_native_api_handlers"
	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

const RequestTimeout = 5 * time.Second

// func WithLeaderForward() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		readOnly := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead
// 		if !readOnly && dbstore.GetDBStore(c.Request.Context()).IsLeader() {
// 			c.Next()
// 			return
// 		}

// 		// TODO: P2: Forward the request to the leader node
// 		// c.Redirect(http.StatusTemporaryRedirect, "http://newlocation.com")
// 		c.Status(http.StatusTemporaryRedirect)
// 		c.Abort()
// 	}
// }

func WithLoggingAndPanicRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
		defer cancel()

		traceID := c.GetHeader(s3nativeapihandlers.AMZ_REQUEST_ID)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Header(s3nativeapihandlers.AMZ_REQUEST_ID, traceID)

		log := logger.GetLogger(ctx).WithFields(map[string]any{"traceId": traceID})
		c.Request = c.Request.WithContext(log.WithLogger(ctx))

		logFields := map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		}
		str, _ := httputil.DumpRequest(c.Request, false)
		log.WithFields(logFields).Debugf("Request received: %s", string(str))

		start := time.Now()
		defer func() {
			logFields["status"] = c.Writer.Status()
			logFields["headers"] = c.Writer.Header()
			logFields["latency"] = fmt.Sprintf("%.1fms", float64(time.Since(start).Microseconds())/1000)
			log.WithFields(logFields).Info("Handled Incoming HTTP Request")
		}()

		defer func() {
			if r := recover(); r != nil {
				log.WithFields(map[string]any{"panic": r, "traceId": traceID}).Errorf("[PANIC] %s", debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
		}()

		c.Next()
	}
}

func WithSigV4Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		userName, err := iam.VerifyRequest(c.Request)
		if err == nil {
			ctx := context.WithValue(c.Request.Context(), utils.UserNameCtxKey, userName)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		s3nativeapihandlers.SendIAMErrorResp(c, err, nil)
		c.Abort()
	}
}
