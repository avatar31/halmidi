package apihandlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	s3nativeapihandlers "github.com/avatar31/halmidi/cmd/rest/s3_native_api_handlers"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
)

func SendErrorResp(c *gin.Context, err error, headers map[string]string) {
	s3Error, ok := err.(s3common.S3Error)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log := logger.GetLogger(c.Request.Context())
	log.WithFields(map[string]any{
		"statusCode": s3Error.HttpStatusCode,
		"errMessage": s3Error.Message,
		"s3Code":     s3Error.S3ErrorCode,
	}).Error("Error in api. Throwing error")

	c.Header("Content-Type", "application/json")
	for k, v := range headers {
		c.Header(k, v)
	}

	resp := ErrorResp{
		Code:      s3Error.S3ErrorCode,
		Message:   s3Error.Message,
		RequestID: c.GetHeader(s3nativeapihandlers.AMZ_REQUEST_ID),
	}

	c.AbortWithStatusJSON(s3Error.HttpStatusCode, resp)
}

func SendSuccessResp(c *gin.Context, httpStatus int, respHeaders map[string]string, resp any) {
	c.Header("Content-Type", "application/json")
	for k, v := range respHeaders {
		c.Header(k, v)
	}

	c.JSON(httpStatus, resp)
}
